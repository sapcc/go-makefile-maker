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

import "github.com/sapcc/go-makefile-maker/internal/core"

func dependencyReviewWorkflow(cfg *core.GithubWorkflowConfiguration) {
	w := newWorkflow("Dependency Review", cfg.Global.DefaultBranch, nil)
	w.On.Push.Branches = []string{} // trigger only on pull requests

	j := baseJob("Review")
	j.addStep(jobStep{
		Name: "Dependency Review",
		Uses: dependencyReviewAction,
		With: map[string]interface{}{
			"fail-on-severity": "high",
			"deny-licenses":    "AGPL-1.0, AGPL-3.0, GPL-1.0, GPL-2.0, GPL-3.0, LGPL-2.0, LGPL-2.1, LGPL-3.0",
		},
	})
	w.Jobs = map[string]job{"review": j}
	writeWorkflowToFile(w)
}
