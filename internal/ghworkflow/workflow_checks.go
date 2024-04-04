// Copyright 2022 SAP SE
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
// limitations under the License.

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
	j := baseJobWithGo("Checks", cfg)

	if ghwCfg.SecurityChecks.Enabled {
		j.addStep(jobStep{
			Name: "Dependency Licenses Review",
			Run:  "make check-dependency-licenses",
		})

		j.addStep(jobStep{
			Name: "Run govulncheck",
			Uses: core.GovulncheckAction,
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

	if ghwCfg.License.Enabled {
		j.addStep(jobStep{
			Name: "Check if source code files have license header",
			Run:  "make check-license-headers",
		})
	}

	w.Jobs = map[string]job{"checks": j}

	writeWorkflowToFile(w)
}
