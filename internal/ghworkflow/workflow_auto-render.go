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
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

func autoRenderWorkflow(cfg *core.GithubWorkflowConfiguration) error {
	w := newWorkflow("Auto render", cfg.Global.DefaultBranch, nil)
	w.Permissions.Contents = tokenScopeWrite // for peter-evans/create-pull-request to create commit & PR

	w.On.PullRequest = pushAndPRTriggerOpts{} // will be run when merged
	w.On.Push = pushAndPRTriggerOpts{Branches: []string{cfg.Global.DefaultBranch}, Paths: []string{"Makefile.maker.yaml"}}
	w.On.Schedule = []cronExpr{{Cron: "00 07 * * 1"}} // every Monday at 07:00 AM
	w.On.WorkflowDispatch.manualTrigger = true

	j := baseJob("go-makefile-maker")
	j.addStep(jobStep{
		Name: "Install go-makefile-maker",
		Run:  "go install github.com/sapcc/go-makefile-maker@latest",
	})
	j.addStep(jobStep{
		Name: "Run go-makefile-maker",
		Run: `
export PATH=$PATH:$(go env GOPATH)/bin
go-makefile-maker`,
	})

	with := map[string]interface{}{
		"author": "${{ github.actor }} <${{ github.actor }}@users.noreply.github.com>",
		"body": `Update report
- Updated with *today's* date
- Auto-generated by [create-pull-request][1]

[1]: https://github.com/peter-evans/create-pull-request`,
		"branch":         "auto-update/go-makefile-maker",
		"commit-message": "Run go-makefile-maker",
		"committer":      "GitHub <noreply@github.com>",
		"delete-branch":  true,
		"title":          "[Cron] Run go-makefile-maker",
	}
	if len(cfg.Global.Assignees) > 0 {
		with["assignees"] = strings.Join(cfg.Global.Assignees, ",")
	}
	j.addStep(jobStep{
		Name: "Create Pull Request",
		Uses: "peter-evans/create-pull-request@v4",
		With: with,
	})

	w.Jobs = map[string]job{"go-makefile-maker": j}
	return writeWorkflowToFile(w)
}
