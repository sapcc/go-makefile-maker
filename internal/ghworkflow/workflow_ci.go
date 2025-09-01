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
	w.On.Push.Branches = []string{ghwCfg.Global.DefaultBranch}

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

	testJob := baseJobWithGo("Test", cfg)
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

	const coverageArtifactName = "code-coverage"
	if !ghwCfg.CI.Coveralls {
		// see https://github.com/fgrosse/go-coverage-report#usage
		testJob.addStep(jobStep{
			Name: "Archive code coverage results",
			Uses: core.GetUploadArtifactAction(ghwCfg.IsSelfHostedRunner),
			With: map[string]any{
				"name": coverageArtifactName,
				"path": "build/cover.out",
			},
		})
	}

	w.Jobs["test"] = testJob

	if ghwCfg.CI.Coveralls {
		multipleOS := len(ghwCfg.CI.RunsOn) > 1
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
	} else {
		// see https://github.com/fgrosse/go-coverage-report#usage
		codeCov := baseJob("Code coverage report", cfg.GitHubWorkflow)
		codeCov.If = "github.event_name == 'pull_request'"
		codeCov.Needs = []string{"test"}
		codeCov.Permissions = permissions{
			Contents:     "read",
			Actions:      "read",
			PullRequests: "write",
		}
		codeCov.addStep(jobStep{
			Name: "Post coverage report",
			Uses: core.GoCoverageReportAction,
			With: map[string]any{
				"coverage-artifact-name": coverageArtifactName,
				"coverage-file-name":     "cover.out",
			},
		})
		w.Jobs["code_coverage"] = codeCov
	}

	w.Jobs["test"] = testJob

	writeWorkflowToFile(w)
}
