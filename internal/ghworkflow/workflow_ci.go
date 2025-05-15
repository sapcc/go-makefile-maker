// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	"fmt"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

func ciWorkflow(cfg core.Configuration, sr golang.ScanResult) {
	ghwCfg := cfg.GitHubWorkflow
	ignorePaths := ghwCfg.CI.IgnorePaths
	if len(ignorePaths) == 0 {
		ignorePaths = append(ignorePaths, "**.md")
	}

	w := newWorkflow("CI", ghwCfg.Global.DefaultBranch, ignorePaths)
	w.On.WorkflowDispatch.manualTrigger = true

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
	testCmd := []string{
		"make build/cover.out",
	}
	if sr.UsesPostgres {
		testCmd = append([]string{
			"sudo /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y",
			"sudo apt-get install -y --no-install-recommends postgresql-" + core.DefaultPostgresVersion,
			fmt.Sprintf("export PATH=/usr/lib/postgresql/%s/bin:$PATH", core.DefaultPostgresVersion),
		}, testCmd...)
	}
	testJob.addStep(jobStep{
		Name: "Run tests and generate coverage report",
		Run:  makeMultilineYAMLString(testCmd),
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
