// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

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

func baseJob(name string, cfg *core.GithubWorkflowConfiguration) job {
	var (
		runsOn   any
		envs     map[string]string
		strategy JobStrategy
	)

	if cfg.IsSelfHostedRunner {
		envs = map[string]string{
			"NODE_EXTRA_CA_CERTS": "/etc/ssl/certs/ca-certificates.crt",
		}
	}

	switch len(cfg.CI.RunsOn) {
	case 0:
		// If no runsOn is specified, we use reasonable defaults
		if cfg.IsSelfHostedRunner {
			if cfg.IsSugarRunner {
				runsOn = core.SugarRunsOn
			} else {
				runsOn = core.DefaultGitHubEnterpriseRunsOn
			}
		} else {
			runsOn = core.DefaultGitHubComRunsOn
		}
	case 1:
		runsOn = cfg.CI.RunsOn[0]
	default: // > 2
		runsOn = "${{ matrix.os }}"
		strategy.Matrix.OS = cfg.CI.RunsOn
	}

	return job{
		Name:   name,
		Env:    envs,
		RunsOn: runsOn,
		Steps: []jobStep{{
			Name: "Check out code",
			Uses: core.CheckoutAction,
		}},
		Strategy: strategy,
	}
}

func baseJobWithGo(name string, cfg core.Configuration) job {
	j := baseJob(name, cfg.GitHubWorkflow)
	j.addStep(jobStep{
		Name: "Set up Go",
		Uses: core.SetupGoAction,
		With: map[string]any{
			"go-version":   cfg.GitHubWorkflow.Global.GoVersion.UnwrapOr(core.DefaultGoVersion),
			"check-latest": true,
		},
	})
	if cfg.GitHubWorkflow.CI.PrepareMakeTarget != "" {
		j.addStep(jobStep{
			Name: "Run prepare make target",
			Run:  "make " + cfg.GitHubWorkflow.CI.PrepareMakeTarget,
		})
	}
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
