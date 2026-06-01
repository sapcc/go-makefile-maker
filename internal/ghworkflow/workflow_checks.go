// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	. "go.xyrillian.de/gg/option"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

// basically a collection of other linters and checks which run fast to reduce the amount of created githbu action workflows
func checksWorkflow(cfg core.Configuration) Option[workflow] {
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
			"version": core.GolangCiLintVersion,
		},
	})

	if cfg.ShellCheck.IsEnabled() {
		// delete the pretty out of date installed version of shellcheck so that make install-shellcheck installs the current version
		if !ghwCfg.IsSelfHostedRunner {
			j.addStep(jobStep{
				Name: "Delete pre-installed shellcheck",
				Run:  `sudo rm -f "$(which shellcheck)"`,
			})
		}
		j.addStep(jobStep{
			Name: "Run shellcheck",
			Run:  "make run-shellcheck",
		})
	}

	if ghwCfg.SecurityChecks.IsEnabled() {
		j.addStep(jobStep{
			Name: "Dependency Licenses Review",
			Run:  "make check-dependency-licenses",
		})
	}

	j.addStep(jobStep{
		Name: "Check for spelling errors",
		Uses: core.TyposAction,
		Env: map[string]string{
			"CLICOLOR": "1",
		},
	})

	if ghwCfg.License.IsEnabled() {
		j.addStep(jobStep{
			Name: "Check if source code files have license header",
			Run:  "make check-addlicense",
		})
	}

	if cfg.Reuse.IsEnabled() {
		j.addStep(jobStep{
			Name: "REUSE Compliance Check",
			Uses: core.ReuseAction,
		})
	}

	w.Jobs = map[string]job{"checks": j}
	return Some(w)
}
