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

	"gopkg.in/yaml.v3"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

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

func baseJob(name string, isSelfHostedRunner bool) job {
	var runnerType any

	if isSelfHostedRunner {
		runnerType = core.DefaultGitHubEnterpriseRunnerType
	} else {
		runnerType = core.DefaultGitHubComRunnerType
	}

	return job{
		Name:   name,
		RunsOn: runnerType,
		Steps: []jobStep{{
			Name: "Check out code",
			Uses: core.CheckoutAction,
		}},
	}
}

func baseJobWithGo(name string, isSelfHostedRunner bool, goVersion string) job {
	j := baseJob(name, isSelfHostedRunner)
	step := jobStep{
		Name: "Set up Go",
		Uses: core.SetupGoAction,
		With: map[string]any{
			"go-version":   goVersion,
			"check-latest": true,
		},
	}
	j.addStep(step)
	return j
}

// makeMultilineYAMLString adds \n to the strings and joins them.
// yaml.Marshal() takes care of the rest.
func makeMultilineYAMLString(in []string) string {
	out := strings.Join(in, "\n")
	// We need the \n at the end to ensure that yaml.Marshal() puts the right
	// chomping indicator; i.e. `|` instead of `|-`.
	if len(in) > 1 {
		out += "\n"
	}
	return out
}

// quotedString is used to force single quotes around a string during Marshal.
type quotedString string

// MarshalYAML implements the yaml.Marshaler interface.
func (qs quotedString) MarshalYAML() (any, error) {
	return yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.SingleQuotedStyle,
		Value: string(qs),
	}, nil
}
