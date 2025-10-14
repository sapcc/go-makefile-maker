// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	"github.com/sapcc/go-makefile-maker/internal/core"
)

func codeQLWorkflow(cfg core.Configuration) {
	ghwCfg := cfg.GitHubWorkflow
	w := newWorkflow("CodeQL", ghwCfg.Global.DefaultBranch, nil)
	w.On.WorkflowDispatch.manualTrigger = true

	if w.deleteIf(ghwCfg.SecurityChecks.IsEnabled()) {
		return
	}

	w.Permissions.Actions = tokenScopeRead         // for github/codeql-action/init to get workflow details
	w.Permissions.SecurityEvents = tokenScopeWrite // for github/codeql-action/analyze to upload SARIF results

	// Overwrite because CodeQL expects the pull_request.branches to be a subset of
	// push.branches.
	w.On.PullRequest.Branches = []string{ghwCfg.Global.DefaultBranch}
	w.On.Schedule = []cronExpr{{Cron: "00 07 * * 1"}} // every Monday at 07:00 AM

	j := baseJobWithGo("CodeQL", cfg)
	j.addStep(jobStep{
		Name: "Initialize CodeQL",
		Uses: core.GetCodeqlInitAction(ghwCfg.IsSelfHostedRunner),
		With: map[string]any{
			"languages": "go",
			"queries":   cfg.GitHubWorkflow.SecurityChecks.Queries.UnwrapOr("security-extended"),
		},
	})
	j.addStep(jobStep{
		Name: "Autobuild",
		Uses: core.GetCodeqlAutobuildAction(ghwCfg.IsSelfHostedRunner),
	})
	j.addStep(jobStep{
		Name: "Perform CodeQL Analysis",
		Uses: core.GetCodeqlAnalyzeAction(ghwCfg.IsSelfHostedRunner),
	})
	w.Jobs = map[string]job{"analyze": j}

	writeWorkflowToFile(w)
}
