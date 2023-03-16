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

///////////////////////////////////////////////////////////////////////////////
// Constants

const (
	defaultPostgresVersion   = "12"
	defaultK8sEnvtestVersion = "1.25.x!"
	defaultRunnerOS          = "ubuntu-latest"
)

const (
	cacheAction            = "actions/cache@v3"
	checkoutAction         = "actions/checkout@v3"
	setupGoAction          = "actions/setup-go@v4"
	dependencyReviewAction = "actions/dependency-review-action@v3"

	codeqlInitAction    = "github/codeql-action/init@v2"
	codeqlAnalyzeAction = "github/codeql-action/analyze@v2"

	golangciLintAction = "golangci/golangci-lint-action@v3"

	misspellAction = "reviewdog/action-misspell@v1"
)

///////////////////////////////////////////////////////////////////////////////
// Helper functions

func pushAndPRTriggers(defaultBranch string, ignorePaths []string) eventTrigger {
	return eventTrigger{
		Push: pushAndPRTriggerOpts{
			Branches:    []string{defaultBranch},
			PathsIgnore: ignorePaths,
		},
		PullRequest: pushAndPRTriggerOpts{
			Branches:    []string{"*"},
			PathsIgnore: ignorePaths,
		},
	}
}

func baseJob(name string) job {
	return job{
		Name:   name,
		RunsOn: defaultRunnerOS,
		Steps: []jobStep{{
			Name: "Check out code",
			Uses: checkoutAction,
		}},
	}
}

func baseJobWithGo(name, goVersion string) job {
	j := baseJob(name)
	step := jobStep{
		Name: "Set up Go",
		Uses: setupGoAction,
		With: map[string]interface{}{"go-version": goVersion},
	}
	j.addStep(step)
	return j
}
