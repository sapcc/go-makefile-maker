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
func checksWorkflow(cfg *core.GithubWorkflowConfiguration, ignoreWords []string) {
	w := newWorkflow("Checks", cfg.Global.DefaultBranch, nil)
	w.Jobs = make(map[string]job)

	if cfg.SecurityChecks.Enabled {
		j := baseJob("Dependency Review")
		j.addStep(jobStep{
			Name: "Dependency Review",
			Uses: dependencyReviewAction,
			With: map[string]interface{}{
				"base-ref":         "${{ github.event.pull_request.base.sha || 'main' }}",
				"head-ref":         "${{ github.event.pull_request.head.sha || github.ref }}",
				"fail-on-severity": "moderate",
				"deny-licenses":    "AGPL-1.0, AGPL-3.0, GPL-1.0, GPL-2.0, GPL-3.0, LGPL-2.0, LGPL-2.1, LGPL-3.0",
			},
		})
		w.Jobs["review"] = j
	}

	if cfg.SpellCheck.Enabled {
		with := map[string]interface{}{
			"exclude":       "./vendor/*",
			"reporter":      "github-check",
			"fail_on_error": true,
			"github_token":  "${{ secrets.GITHUB_TOKEN }}",
			"ignore":        "importas", //nolint:misspell //importas is a valid linter name, so we always ignore it
		}
		if len(ignoreWords) > 0 {
			with["ignore"] = fmt.Sprintf("%s,%s", with["ignore"], strings.Join(ignoreWords, ","))
		}

		j := baseJob("Spellcheck")
		j.Permissions.Checks = tokenScopeWrite // for nicer output in pull request diffs
		j.addStep(jobStep{
			Name: "Check for spelling errors",
			Uses: misspellAction,
			With: with,
		})
		w.Jobs["misspell"] = j
	}

	if cfg.License.Enabled {
		// Default behavior is to check all Go files excluding the vendor directory.
		patterns := []string{"**/*.go"}
		if len(cfg.License.Patterns) > 0 {
			patterns = cfg.License.Patterns
		}

		ignorePatterns := []string{"vendor/**"}
		if len(cfg.License.IgnorePatterns) > 0 {
			ignorePatterns = append(ignorePatterns, cfg.License.IgnorePatterns...)
		}
		// Each ignore pattern is quoted to avoid glob expansion and prefixed with the
		// `-ignore` flag.
		for i, v := range ignorePatterns {
			ignorePatterns[i] = fmt.Sprintf("-ignore %q", v)
		}

		j := baseJobWithGo("License header", cfg.Global.GoVersion)
		j.addStep(jobStep{
			Name: "Check if source code files have license header",
			Run: makeMultilineYAMLString([]string{
				"shopt -s globstar", // so that we can use '**' in file patterns
				"go install github.com/google/addlicense@latest",
				fmt.Sprintf("addlicense --check %s -- %s",
					strings.Join(ignorePatterns, " "),
					strings.Join(patterns, " "),
				),
			}),
		})
		w.Jobs["addlicense"] = j
	}

	writeWorkflowToFile(w)
}
