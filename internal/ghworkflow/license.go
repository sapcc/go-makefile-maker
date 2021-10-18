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

import "strings"

func licenseWorkflow(cfg *Configuration) error {
	ignorePaths := cfg.Global.IgnorePaths
	if cfg.License.IgnorePaths != nil {
		ignorePaths = cfg.License.IgnorePaths
	}

	cmd := "addlicense --check -- " + strings.Join(cfg.License.Patterns, " ")
	if len(cfg.License.Patterns) == 0 {
		// Check all .go files excluding the vendor directory.
		cmd = `find * \( -name vendor -type d -prune \) -o \( -name \*.go -exec addlicense --check -- {} + \)`
	}

	w := &workflow{
		Name: "License",
		On:   eventTriggers(cfg.Global.DefaultBranch, ignorePaths),
	}
	j := baseJobWithGo("Check", defaultGoVersion)
	j.Steps = append(j.Steps, jobStep{
		Name: "Check if source code files have license header",
		Run: makeMultilineYAMLString([]string{
			"go install github.com/google/addlicense@latest",
			cmd,
		}),
	},
	)
	w.Jobs = map[string]job{"addlicense": j}

	return writeWorkflowToFile(w)
}
