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
// limitations under the License.

package ghworkflow

import (
	"slices"
	"strings"

	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

func ghcrWorkflow(cfg *core.GithubWorkflowConfiguration) {
	// https://docs.github.com/en/packages/managing-github-packages-using-github-actions-workflows/publishing-and-installing-a-package-with-github-actions#publishing-a-package-using-an-action
	w := newWorkflow("Container Registry GHCR", cfg.Global.DefaultBranch, nil)

	if w.deleteIf(cfg.PushContainerToGhcr.Enabled) {
		return
	}

	w.Permissions.Contents = tokenScopeRead
	w.Permissions.Packages = tokenScopeWrite

	w.On.Push.Branches = nil
	w.On.Push.Tags = []string{"*"}
	w.On.PullRequest.Branches = nil

	registry := "ghcr.io"

	j := baseJob("Push container to ghcr.io", cfg.IsSelfHostedRunner)
	j.addStep(jobStep{
		Name: "Log in to the Container registry",
		Uses: core.DockerLoginAction,
		With: map[string]any{
			"registry": registry,
			"username": "${{ github.actor }}",
			"password": "${{ secrets.GITHUB_TOKEN }}",
		},
	})

	var tags string
	strategy := cfg.PushContainerToGhcr.TagStrategy
	if slices.Contains(strategy, "edge") {
		strategy = slices.DeleteFunc(strategy, func(s string) bool {
			return s == "edge"
		})
		tags += `# https://github.com/docker/metadata-action#typeedge
type=edge
`
	}
	if slices.Contains(strategy, "latest") {
		strategy = slices.DeleteFunc(strategy, func(s string) bool {
			return s == "latest"
		})
		tags += `# https://github.com/docker/metadata-action#latest-tag
type=raw,value=latest,enable={{is_default_branch}}
`
	}
	if slices.Contains(strategy, "semver") {
		strategy = slices.DeleteFunc(strategy, func(s string) bool {
			return s == "semver"
		})
		tags += `# https://github.com/docker/metadata-action#typesemver
type=semver,pattern={{raw}}
type=semver,pattern=v{{major}}.{{minor}}
type=semver,pattern=v{{major}}
`
	}

	if len(strategy) != 0 {
		logg.Fatal("unknown tagStrategy: %s", strings.Join(strategy, ", "))
	}

	j.addStep(jobStep{
		Name: "Extract metadata (tags, labels) for Docker",
		ID:   "meta",
		Uses: core.DockerMetadataAction,
		With: map[string]any{
			"images": registry + "/${{ github.repository }}",
			"tags":   tags,
		},
	})
	j.addStep(jobStep{
		Name: "Set up QEMU",
		Uses: core.DockerQemuAction,
	})
	j.addStep(jobStep{
		Name: "Set up Docker Buildx",
		Uses: core.DockerBuildxAction,
	})
	j.addStep(jobStep{
		Name: "Build and push Docker image",
		Uses: core.DockerBuildPushAction,
		With: map[string]any{
			"context":   ".",
			"push":      true,
			"tags":      "${{ steps.meta.outputs.tags }}",
			"labels":    "${{ steps.meta.outputs.labels }}",
			"platforms": "linux/amd64,linux/arm64",
		},
	})
	w.Jobs = map[string]job{"build-and-push-image": j}

	writeWorkflowToFile(w)
}
