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
		maybeSudo := "sudo "
		var installPostgresCommands []string

		if ghwCfg.IsSelfHostedRunner {
			// Our self hosted runners do not have sudo installed
			maybeSudo = ""
			// I hate it, too...
			testJob.Container = "keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/debian:stable"
			installPostgresCommands = append([]string{
				"apt-get update",
				// Not pre-installed on our runners unlike to the official ones
				// make is not in the default debian image we must use because of restrictions in the default runner image
				"apt-get install -y --no-install-recommends postgresql-client-common make",
				// https is blocked but luckily not http. Since releases are gpg signed anyhow, this is not a problem.
				"sed -i 's/https:/http:/g' /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh",
			}, installPostgresCommands...)

			// and ofcourse we also need to download the self signed certificates
			testJob.Steps = append([]jobStep{{
				Name: "Get self signed certs",
				Run: makeMultilineYAMLString([]string{
					"apt update",
					"apt install -y --no-install-recommends ca-certificates curl",
					"curl -sSfL 'http://aia.pki.co.sap.com/aia/SAPNetCA_G2.crt' -o /usr/local/share/ca-certificates/SAPNetCA_G2.crt",
					"curl -sSfL 'http://aia.pki.co.sap.com/aia/SAP%20Global%20Root%20CA.crt' -o /usr/local/share/ca-certificates/SAP_Global_Root_CA.crt",
					"update-ca-certificates",
				}),
			}}, testJob.Steps...)
		}

		installPostgresCommands = append(installPostgresCommands,
			maybeSudo+"/usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y",
			maybeSudo+"apt-get install -y --no-install-recommends postgresql-"+core.DefaultPostgresVersion,
			fmt.Sprintf("export PATH=/usr/lib/postgresql/%s/bin:$PATH", core.DefaultPostgresVersion),
		)

		testCmd = append(installPostgresCommands, testCmd...)
	}
	testJob.addStep(jobStep{
		Name: "Run tests and generate coverage report",
		Run:  makeMultilineYAMLString(testCmd),
	})

	// see https://github.com/fgrosse/go-coverage-report#usage
	coverageArtifactName := "code-coverage"
	testJob.addStep(jobStep{
		Name: "Archive code coverage results",
		Uses: core.GetUploadArtifactAction(ghwCfg.IsSelfHostedRunner),
		With: map[string]any{
			// TODO: upload without zipping (i.e. set "archive": "true") once we can use v7+ everywhere
			// Ref: <https://github.com/actions/upload-artifact/releases/tag/v7.0.0>
			"name": coverageArtifactName,
			"path": "build/cover.out",
		},
	})

	w.Jobs["test"] = testJob

	// coverage is only available on github.com because tj-actions/changed-files is blocked due to their famour securits incident
	if !ghwCfg.IsSelfHostedRunner {
		// see https://github.com/fgrosse/go-coverage-report#usage
		codeCov := baseJob("Code coverage report", cfg.GitHubWorkflow)
		codeCov.If = "github.event_name == 'pull_request' && github.event.pull_request.head.repo.full_name == github.repository"
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

	writeWorkflowToFile(w)
}
