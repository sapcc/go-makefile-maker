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
// limitations under the License.

package ghworkflow

import "github.com/sapcc/go-makefile-maker/internal/core"

func codeQLWorkflow(cfg *core.GithubWorkflowConfiguration) error {
	ignorePaths := cfg.Global.IgnorePaths
	if cfg.CodeQL.IgnorePaths != nil {
		ignorePaths = cfg.CodeQL.IgnorePaths
	}

	w := newWorkflow("CodeQL", cfg.Global.DefaultBranch, ignorePaths)
	w.Permissions.Actions = tokenScopeRead         // for github/codeql-action/init to get workflow details
	w.Permissions.SecurityEvents = tokenScopeWrite // for github/codeql-action/analyze to upload SARIF results

	// Overwrite because CodeQL expects the pull_request.branches to be a subset of
	// push.branches.
	w.On.PullRequest.Branches = []string{cfg.Global.DefaultBranch}
	w.On.Schedule = []cronExpr{{Cron: "00 07 * * 1"}} // every Monday at 07:00 AM

	j := baseJob("Analyze")
	j.addStep(jobStep{
		Name: "Initialize CodeQL",
		Uses: "github/codeql-action/init@v1",
		With: map[string]interface{}{
			"languages": "go",
			"queries":   "security-and-quality",
		},
	})
	j.addStep(jobStep{
		Name: "Perform CodeQL Analysis",
		Uses: "github/codeql-action/analyze@v1",
	})
	w.Jobs = map[string]job{"analyze": j}

	return writeWorkflowToFile(w)
}
