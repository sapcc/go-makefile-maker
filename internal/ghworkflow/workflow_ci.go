// Copyright 2021 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ghworkflow

import (
	"fmt"
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

func ciWorkflow(cfg *core.GithubWorkflowConfiguration, vendoring, hasBinaries bool) {
	goVersion := cfg.Global.GoVersion

	w := newWorkflow("CI", cfg.Global.DefaultBranch, cfg.CI.IgnorePaths)
	w.Jobs = make(map[string]job)

	// 01. Lint codebase.
	lintJob := baseJobWithGo("Lint", goVersion)
	lintJob.addStep(jobStep{
		Name: "Run golangci-lint",
		Uses: golangciLintAction,
		With: map[string]interface{}{
			"version": "latest",
		},
	})
	w.Jobs["lint"] = lintJob

	buildTestOpts := buildTestJobOpts{
		goVersion:    goVersion,
		vendoring:    vendoring,
		runnerOSList: cfg.CI.RunnerOSList,
	}

	// 02. Make build.
	if hasBinaries {
		buildTestOpts.name = "Build"
		buildJob := buildOrTestBaseJob(buildTestOpts)
		buildJob.Needs = []string{"lint"} // this is the <job_id> for the lint job
		buildJob.addStep(jobStep{
			Name: "Make build",
			Run:  "make build-all",
		})
		w.Jobs["build"] = buildJob
	}

	// 03. Run tests and generate/upload test coverage.
	buildTestOpts.name = "Test"
	testJob := buildOrTestBaseJob(buildTestOpts)
	testJob.Needs = []string{"lint"} // this is the <job_id> for the lint job
	if hasBinaries {
		testJob.Needs = []string{"build"} // this is the <job_id> for the build job
	}
	if cfg.CI.Postgres.Enabled {
		version := defaultPostgresVersion
		if cfg.CI.Postgres.Version != "" {
			version = cfg.CI.Postgres.Version
		}
		testJob.Services = map[string]jobService{"postgres": {
			Image: "postgres:" + version,
			Env:   map[string]string{"POSTGRES_PASSWORD": "postgres"},
			Ports: []string{"54321:5432"},
			Options: strings.Join([]string{
				// Set health checks to wait until postgres has started
				"--health-cmd pg_isready",
				"--health-interval 10s",
				"--health-timeout 5s",
				"--health-retries 5",
			}, " "),
		}}
	}
	if cfg.CI.KubernetesEnvtest.Enabled {
		testJob.addStep(jobStep{
			ID:   "cache-envtest",
			Name: "Cache envtest binaries",
			Uses: cacheAction,
			With: map[string]interface{}{
				"path": "test/bin",
				"key":  `${{ runner.os }}-envtest-${{ hashFiles('Makefile.maker.yaml') }}`,
			},
		})
		// Download the envtest binaries, in case of cache miss.
		envtestVersion := defaultK8sEnvtestVersion
		if cfg.CI.KubernetesEnvtest.Version != "" {
			envtestVersion = cfg.CI.KubernetesEnvtest.Version
		}
		testJob.addStep(jobStep{
			Name: "Download envtest binaries",
			If:   "steps.cache-envtest.outputs.cache-hit != 'true'",
			Run: makeMultilineYAMLString([]string{
				"go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest",
				"mkdir -p test/bin", // create dir if it doesn't exist already
				fmt.Sprintf("setup-envtest --bin-dir test/bin use %s", envtestVersion),
			}),
		})
	}
	testJob.addStep(jobStep{
		Name: "Run tests and generate coverage report",
		Run:  "make build/cover.out",
	})
	if cfg.CI.Coveralls {
		multipleOS := len(cfg.CI.RunnerOSList) > 1
		env := map[string]string{
			"GIT_BRANCH":      "${{ github.head_ref }}",
			"COVERALLS_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		}
		installGoveralls := "go install github.com/mattn/goveralls@latest"
		cmd := "goveralls -service=github -coverprofile=build/cover.out"
		if multipleOS {
			cmd += ` -parallel -flagname="Unit-${{ matrix.os }}"`
		}
		testJob.addStep(jobStep{
			Name: "Upload coverage report to Coveralls",
			Run:  makeMultilineYAMLString([]string{installGoveralls, cmd}),
			Env:  env,
		})

		if multipleOS {
			// 04. Tell Coveralls to merge coverage results.
			finishJob := baseJobWithGo("Finish", goVersion)
			finishJob.Needs = []string{"test"} // this is the <job_id> for the test job
			finishJob.addStep(jobStep{
				Name: "Coveralls post build webhook",
				Run:  makeMultilineYAMLString([]string{installGoveralls, "goveralls -parallel-finish"}),
				Env:  env,
			})
			w.Jobs["finish"] = finishJob
		}
	}
	w.Jobs["test"] = testJob

	writeWorkflowToFile(w)
}

type buildTestJobOpts struct {
	name         string
	goVersion    string
	vendoring    bool
	runnerOSList []string
}

func buildOrTestBaseJob(opts buildTestJobOpts) job {
	j := baseJobWithGo(opts.name, opts.goVersion)
	switch len(opts.runnerOSList) {
	case 0:
		// baseJobWithGo() will set j.RunsOn to defaultRunnerOS.
	case 1:
		j.RunsOn = opts.runnerOSList[0]
	default:
		j.RunsOn = "${{ matrix.os }}"
		j.Strategy.Matrix.OS = opts.runnerOSList
	}
	return j
}
