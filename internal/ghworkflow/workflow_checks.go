// SPDX-FileCopyrightText: Copyright 2022 SAP SE
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	"fmt"
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

// basically a collection of other linters and checks which run fast to reduce the amount of created githbu action workflows
func checksWorkflow(cfg core.Configuration) {
	ghwCfg := cfg.GitHubWorkflow
	w := newWorkflow("Checks", ghwCfg.Global.DefaultBranch, nil)
	w.On.WorkflowDispatch.manualTrigger = true
	j := baseJobWithGo("Checks", cfg)

	// see https://github.com/golangci/golangci-lint-action#annotations
	w.Permissions.Checks = tokenScopeWrite
	j.addStep(jobStep{
		Name: "Run golangci-lint",
		Uses: core.GolangciLintAction,
		With: map[string]any{
			"version": "latest",
		},
	})

	if ghwCfg.SecurityChecks.Enabled == nil || *ghwCfg.SecurityChecks.Enabled {
		j.addStep(jobStep{
			Name: "Dependency Licenses Review",
			Run:  "make check-dependency-licenses",
		})

		// we are not using golang/govulncheck-action because that always wants to install go again
		// https://github.com/golang/govulncheck-action/blob/master/action.yml
		j.addStep(jobStep{
			Name: "Install govulncheck",
			Run:  "go install golang.org/x/vuln/cmd/govulncheck@latest",
		})

		j.addStep(jobStep{
			Name: "Run govulncheck",
			Run:  "govulncheck -format text ./...",
		})
	}

	if !ghwCfg.IsSelfHostedRunner {
		with := map[string]any{
			"exclude":       "./vendor/*",
			"reporter":      "github-check",
			"fail_on_error": true,
			"github_token":  "${{ secrets.GITHUB_TOKEN }}",
			"ignore":        "importas", //nolint:misspell //importas is a valid linter name, so we always ignore it
		}
		ignoreWords := cfg.SpellCheck.IgnoreWords
		if len(ignoreWords) > 0 {
			with["ignore"] = fmt.Sprintf("%s,%s", with["ignore"], strings.Join(ignoreWords, ","))
		}

		w.Permissions.Checks = tokenScopeWrite // for nicer output in pull request diffs
		j.addStep(jobStep{
			Name: "Check for spelling errors",
			Uses: core.MisspellAction,
			With: with,
		})
	}

	if ghwCfg.License.Enabled == nil || *ghwCfg.License.Enabled {
		j.addStep(jobStep{
			Name: "Check if source code files have license header",
			Run:  "make check-license-headers",
		})
	}

	if cfg.Reuse.IsEnabled() {
		j.addStep(jobStep{
			Name: "REUSE Compliance Check",
			Uses: core.ReuseAction,
		})
	}

	w.Jobs = map[string]job{"checks": j}

	writeWorkflowToFile(w)
}
