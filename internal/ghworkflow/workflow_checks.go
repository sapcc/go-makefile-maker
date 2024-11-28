// Copyright 2022 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
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

	w.Jobs = map[string]job{"checks": j}

	writeWorkflowToFile(w)
}
