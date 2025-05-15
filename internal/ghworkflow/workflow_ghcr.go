// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

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
	w.On.WorkflowDispatch.manualTrigger = true

	strategy := cfg.PushContainerToGhcr.TagStrategy
	if slices.Contains(strategy, "edge") {
		w.On.Push.Branches = []string{cfg.Global.DefaultBranch}
	} else {
		w.On.Push.Tags = []string{"*"}
	}
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
	if slices.Contains(strategy, "sha") {
		strategy = slices.DeleteFunc(strategy, func(s string) bool {
			return s == "sha"
		})
		tags += `# https://github.com/docker/metadata-action#typesha
type=sha,format=long
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

	platforms := cfg.PushContainerToGhcr.Platforms
	if platforms == "" {
		platforms = "linux/amd64"
	}
	j.addStep(jobStep{
		Name: "Build and push Docker image",
		Uses: core.DockerBuildPushAction,
		With: map[string]any{
			"context":   ".",
			"push":      true,
			"tags":      "${{ steps.meta.outputs.tags }}",
			"labels":    "${{ steps.meta.outputs.labels }}",
			"platforms": platforms,
		},
	})
	w.Jobs = map[string]job{"build-and-push-image": j}

	writeWorkflowToFile(w)
}
