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
	"fmt"
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

func licenseWorkflow(cfg *core.GithubWorkflowConfiguration) {
	ignorePaths := cfg.Global.IgnorePaths
	if cfg.License.IgnorePaths != nil {
		ignorePaths = cfg.License.IgnorePaths
	}

	// Default behavior is to check all Go files excluding the vendor directory.
	patterns := []string{"**/*.go"}
	if len(cfg.License.Patterns) > 0 {
		patterns = cfg.License.Patterns
	}

	ignorePatterns := []string{"vendor/**"}
	if len(cfg.License.IgnorePatterns) > 0 {
		ignorePatterns = cfg.License.IgnorePatterns
	}
	// Each ignore pattern is quoted to avoid glob expansion and prefixed with the
	// `-ignore` flag.
	for i, v := range ignorePatterns {
		ignorePatterns[i] = fmt.Sprintf("-ignore %q", v)
	}

	w := newWorkflow("License", cfg.Global.DefaultBranch, ignorePaths)
	j := baseJobWithGo("Check", cfg.Global.GoVersion, false)
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
	w.Jobs = map[string]job{"addlicense": j}

	writeWorkflowToFile(w)
}
