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

func spellCheckWorkflow(cfg *core.GithubWorkflowConfiguration, ignoreWords []string) error {
	ignorePaths := cfg.Global.IgnorePaths
	if cfg.SpellCheck.IgnorePaths != nil {
		ignorePaths = cfg.SpellCheck.IgnorePaths
	}

	with := map[string]interface{}{
		"exclude":       "./vendor/*",
		"reporter":      "github-check",
		"fail_on_error": true,
		"github_token":  "${{ secrets.GITHUB_TOKEN }}",
	}
	if len(ignoreWords) > 0 {
		with["ignore"] = strings.Join(ignoreWords, ",")
	}

	w := newWorkflow("Spell", cfg.Global.DefaultBranch, ignorePaths)
	j := baseJob("Check")
	j.Permissions.Checks = tokenScopeWrite // for nicer output in pull request diffs
	j.Steps = append(j.Steps, jobStep{
		Name: "Check for spelling errors",
		Uses: "reviewdog/action-misspell@v1",
		With: with,
	})
	w.Jobs = map[string]job{"misspell": j}

	return writeWorkflowToFile(w)
}
