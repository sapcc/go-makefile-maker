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

///////////////////////////////////////////////////////////////////////////////
// Constants and Variables

const (
	defaultPostgresVersion = "12"
	defaultRunnerOS        = "ubuntu-latest"
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
			Uses: "actions/checkout@v3",
		}},
	}
}

func baseJobWithGo(name, goVersion string) job {
	j := baseJob(name)
	j.addStep(jobStep{
		Name: "Set up Go",
		Uses: "actions/setup-go@v2",
		With: map[string]interface{}{"go-version": goVersion},
	})
	return j
}

// Ref: https://github.com/actions/cache/blob/main/examples.md#go---modules\
// enableBuildCache enables additional caching of `go-build` directory.
// runnerOS is only required when enableBuildCache is true.
func cacheGoModules(enableBuildCache bool, runnerOS string) jobStep {
	js := jobStep{
		Name: "Cache Go modules",
		Uses: "actions/cache@v3",
		With: map[string]interface{}{
			"key":          `${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}`,
			"restore-keys": "${{ runner.os }}-go-",
		},
	}

	paths := []string{"~/go/pkg/mod"}
	if strings.HasPrefix(runnerOS, "windows") {
		paths[0] = `~\go\pkg\mod`
	}
	if enableBuildCache {
		switch {
		case strings.HasPrefix(runnerOS, "ubuntu"):
			paths = append(paths, "~/.cache/go-build")
		case strings.HasPrefix(runnerOS, "macos"):
			paths = append(paths, "~/Library/Caches/go-build")
		case strings.HasPrefix(runnerOS, "windows"):
			paths = append(paths, `~\AppData\Local\go-build`)
		}
	}
	js.With["path"] = makeMultilineYAMLString(paths)

	return js
}
