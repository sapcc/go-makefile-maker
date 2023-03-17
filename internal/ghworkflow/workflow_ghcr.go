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

import "github.com/sapcc/go-makefile-maker/internal/core"

func ghcrWorkflow(cfg *core.GithubWorkflowConfiguration) {
	// https://docs.github.com/en/packages/managing-github-packages-using-github-actions-workflows/publishing-and-installing-a-package-with-github-actions#publishing-a-package-using-an-action
	w := newWorkflow("Container Registry GHCR", cfg.Global.DefaultBranch, nil)
	w.Permissions.Contents = tokenScopeRead
	w.Permissions.Packages = tokenScopeWrite

	w.On.Push.Branches = []string{cfg.Global.DefaultBranch}
	w.On.Push.Tags = []string{"*"}
	w.On.PullRequest.Branches = nil

	registry := "ghcr.io"

	j := baseJob("Push container to ghcr.io")
	j.addStep(jobStep{
		Name: "Log in to the Container registry",
		Uses: dockerLoginAction,
		With: map[string]interface{}{
			"registry": registry,
			"username": "${{ github.actor }}",
			"password": "${{ secrets.GITHUB_TOKEN }}",
		},
	})
	j.addStep(jobStep{
		Name: "Extract metadata (tags, labels) for Docker",
		ID:   "meta",
		Uses: dockerMetadataAction,
		With: map[string]interface{}{
			"images": registry + "/${{ github.repository }}",
			// https://github.com/docker/metadata-action#latest-tag
			"tags": "type=raw,value=latest,enable={{is_default_branch}}",
		},
	})
	j.addStep(jobStep{
		Name: "Build and push Docker image",
		Uses: dockerBuildPushAction,
		With: map[string]interface{}{
			"context": ".",
			"push":    true,
			"tags":    "${{ steps.meta.outputs.tags }}",
			"labels":  "${{ steps.meta.outputs.labels }}",
		},
	})
	w.Jobs = map[string]job{"build-and-push-image": j}

	writeWorkflowToFile(w)
}
