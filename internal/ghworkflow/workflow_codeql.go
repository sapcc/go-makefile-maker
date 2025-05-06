// SPDX-FileCopyrightText: Copyright 2021 SAP SE
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

func codeQLWorkflow(cfg core.Configuration) {
	ghwCfg := cfg.GitHubWorkflow
	w := newWorkflow("CodeQL", ghwCfg.Global.DefaultBranch, nil)
	w.On.WorkflowDispatch.manualTrigger = true

	if w.deleteIf((ghwCfg.SecurityChecks.Enabled == nil || *ghwCfg.SecurityChecks.Enabled)) {
		return
	}

	w.Permissions.Actions = tokenScopeRead         // for github/codeql-action/init to get workflow details
	w.Permissions.SecurityEvents = tokenScopeWrite // for github/codeql-action/analyze to upload SARIF results

	// Overwrite because CodeQL expects the pull_request.branches to be a subset of
	// push.branches.
	w.On.PullRequest.Branches = []string{ghwCfg.Global.DefaultBranch}
	w.On.Schedule = []cronExpr{{Cron: "00 07 * * 1"}} // every Monday at 07:00 AM

	var (
		initAction    = core.CodeqlInitAction
		buildAction   = core.CodeqlAutobuildAction
		analyzeAction = core.CodeqlAnalyzeAction
	)

	if ghwCfg.IsSelfHostedRunner {
		initAction = strings.ReplaceAll(initAction, "github/", "Security-Testing/")
		buildAction = strings.ReplaceAll(buildAction, "github/", "Security-Testing/")
		analyzeAction = strings.ReplaceAll(analyzeAction, "github/", "Security-Testing/")
	}

	j := baseJobWithGo("CodeQL", cfg)
	j.addStep(jobStep{
		Name: "Initialize CodeQL",
		Uses: initAction,
		With: map[string]any{
			"languages": "go",
			"queries":   "security-extended",
		},
	})
	j.addStep(jobStep{
		Name: "Autobuild",
		Uses: buildAction,
	})
	j.addStep(jobStep{
		Name: "Perform CodeQL Analysis",
		Uses: analyzeAction,
	})
	w.Jobs = map[string]job{"analyze": j}

	writeWorkflowToFile(w)
}
