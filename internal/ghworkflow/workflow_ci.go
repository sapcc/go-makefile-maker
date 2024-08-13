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
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

func ciWorkflow(cfg core.Configuration) {
	ghwCfg := cfg.GitHubWorkflow
	ignorePaths := ghwCfg.CI.IgnorePaths
	if len(ignorePaths) == 0 {
		ignorePaths = append(ignorePaths, "**.md")
	}
	w := newWorkflow("CI", ghwCfg.Global.DefaultBranch, ignorePaths)

	if w.deleteIf(ghwCfg.CI.Enabled) {
		return
	}

	w.Jobs = make(map[string]job)
	build := baseJobWithGo("Build", cfg)
	if len(cfg.Binaries) > 0 {
		build.addStep(jobStep{
			Name: "Build all binaries",
			Run:  "make build-all",
		})
	}

	w.Jobs["build"] = build

	testJob := buildOrTestBaseJob("Test", cfg)
	testJob.Needs = []string{"build"}
	if ghwCfg.CI.Postgres {
		testJob.Services = map[string]jobService{"postgres": {
			Image: "postgres:" + core.DefaultPostgresVersion,
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
	testJob.addStep(jobStep{
		Name: "Run tests and generate coverage report",
		Run:  "make build/cover.out",
	})
	if ghwCfg.CI.Coveralls && !ghwCfg.IsSelfHostedRunner {
		multipleOS := len(ghwCfg.CI.RunnerType) > 1
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
			finishJob := baseJobWithGo("Finish", cfg)
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

func buildOrTestBaseJob(name string, cfg core.Configuration) job {
	ghwCfg := cfg.GitHubWorkflow
	j := baseJobWithGo(name, cfg)
	switch len(ghwCfg.CI.RunnerType) {
	case 0:
		// baseJobWithGo() will set j.RunsOn to DefaultGitHubComRunnerType.
	case 1:
		j.RunsOn = ghwCfg.CI.RunnerType[0]
	default:
		j.RunsOn = "${{ matrix.os }}"
		j.Strategy.Matrix.OS = ghwCfg.CI.RunnerType
	}
	return j
}
