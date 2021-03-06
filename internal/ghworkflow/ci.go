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
	"errors"
	"io/ioutil"
	"strings"

	"golang.org/x/mod/modfile"
)

func ciWorkflow(cfg *Configuration) error {
	goVersion := cfg.CI.GoVersion
	if goVersion == "" {
		filename := "go.mod"
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		f, err := modfile.Parse(filename, data, nil)
		if err != nil {
			return err
		}
		goVersion = f.Go.Version
		if goVersion == "" {
			return errors.New("could not find Go version from go.mod file, consider defining manually by setting githubWorkflows.ci.goVersion")
		}
	}
	ignorePaths := cfg.Global.IgnorePaths
	if cfg.CI.IgnorePaths != nil {
		ignorePaths = cfg.CI.IgnorePaths
	}

	w := &workflow{
		Name: "CI",
		On:   eventTriggers(cfg.Global.DefaultBranch, ignorePaths),
		Jobs: make(map[string]job),
	}

	makeOpts := ""
	if cfg.Vendoring {
		makeOpts = "GO_BUILDFLAGS='-mod vendor'"
	}

	// 01. Lint codebase.
	lintJob := baseJobWithGo("Lint", goVersion)
	lintJob.If = `!contains(github.event.head_commit.message, '[ci skip]')`
	if cfg.GolangciLint {
		// Don't need actions/cache here; golangci-lint has built-in caching.
		lintJob.addStep(jobStep{
			Name: "Run golangci-lint",
			Uses: "golangci/golangci-lint-action@v2",
			With: map[string]interface{}{
				"version":              "latest",
				"skip-go-installation": true,
			},
		})
	} else {
		if !cfg.Vendoring {
			lintJob.addStep(cacheGoModules(true, lintJob.RunsOn))
		}
		lintJob.addStep(jobStep{
			Name: "Run gofmt, golint, go vet",
			Run:  stringsJoinAndTrimSpace([]string{makeOpts, "make", "static-check"}),
		})
	}
	w.Jobs["lint"] = lintJob

	buildTestOpts := buildTestJobOpts{
		goVersion:    goVersion,
		vendoring:    cfg.Vendoring,
		runnerOSList: cfg.CI.RunnerOSList,
	}

	// 02. Make build.
	buildTestOpts.name = "Build"
	buildJob := buildOrTestBaseJob(buildTestOpts)
	buildJob.Needs = []string{"lint"} // this is the <job_id> for the lint job
	buildJob.addStep(jobStep{
		Name: "Make build",
		Run:  stringsJoinAndTrimSpace([]string{makeOpts, "make", "build-all"}),
	})
	w.Jobs["build"] = buildJob

	// 03. Run tests and generate/upload test coverage.
	buildTestOpts.name = "Test"
	testJob := buildOrTestBaseJob(buildTestOpts)
	testJob.Needs = []string{"build"} // this is the <job_id> for the build job
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
	testJob.addStep(jobStep{
		Name: "Run tests and generate coverage report",
		Run:  stringsJoinAndTrimSpace([]string{makeOpts, "make", "build/cover.out"}),
	})
	if cfg.CI.Coveralls {
		multipleOS := len(cfg.CI.RunnerOSList) > 1
		env := map[string]string{
			"GIT_BRANCH":      "${{ github.head_ref }}",
			"COVERALLS_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		}
		getCmd := "GO111MODULE=off go get -u github.com/mattn/goveralls"
		cmd := "goveralls -service=github -coverprofile=build/cover.out"
		if multipleOS {
			cmd += ` -parallel -flagname="Unit-${{ matrix.os }}`
		}
		testJob.addStep(jobStep{
			Name: "Upload coverage report to Coveralls",
			Run:  makeMultilineYAMLString([]string{getCmd, cmd}),
			Env:  env,
		})

		if multipleOS {
			// 04. Tell Coveralls to merge coverage results.
			finishJob := baseJobWithGo("Finish", goVersion)
			finishJob.Needs = []string{"test"} // this is the <job_id> for the test job
			finishJob.addStep(jobStep{
				Name: "Coveralls post build webhook",
				Run:  makeMultilineYAMLString([]string{getCmd, "goveralls -parallel-finish"}),
				Env:  env,
			})
			w.Jobs["finish"] = finishJob
		}
	}
	w.Jobs["test"] = testJob

	return writeWorkflowToFile(w)
}

type buildTestJobOpts struct {
	name         string
	goVersion    string
	vendoring    bool
	runnerOSList []string
}

func buildOrTestBaseJob(opts buildTestJobOpts) job {
	j := baseJobWithGo(opts.name, opts.goVersion)
	buildCache := true
	switch len(opts.runnerOSList) {
	case 0:
		// baseJobWithGo() sets j.RunsOn to defaultRunnerOS
	case 1:
		j.RunsOn = opts.runnerOSList[0]
	default:
		buildCache = false
		j.RunsOn = "${{ matrix.os }}"
		j.Strategy.Matrix.OS = opts.runnerOSList
	}
	if !opts.vendoring {
		j.addStep(cacheGoModules(buildCache, j.RunsOn))
	}
	return j
}
