// SPDX-FileCopyrightText: 2023 SAP SE
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import "github.com/sapcc/go-makefile-maker/internal/core"

func releaseWorkflow(cfg core.Configuration) {
	// https://docs.github.com/en/packages/managing-github-packages-using-github-actions-workflows/publishing-and-installing-a-package-with-github-actions#publishing-a-package-using-an-action
	ghwCfg := cfg.GitHubWorkflow
	w := newWorkflow("goreleaser", ghwCfg.Global.DefaultBranch, nil)

	if w.deleteIf(ghwCfg.Release.Enabled) {
		return
	}

	w.Permissions.Contents = tokenScopeWrite
	w.Permissions.Packages = tokenScopeWrite

	w.On.Push.Branches = nil
	w.On.PullRequest.Branches = nil
	w.On.Push.Tags = []string{"*"} // goreleaser uses semver to decide if this is a prerelease or not

	j := baseJobWithGo("goreleaser", cfg)
	// This is needed because: https://goreleaser.com/ci/actions/#fetch-depthness
	j.Steps[0].With = map[string]any{
		"fetch-depth": 0,
	}
	j.addStep(jobStep{
		Name: "Generate release info",
		Run: makeMultilineYAMLString([]string{
			"go install github.com/sapcc/go-bits/tools/release-info@latest",
			"mkdir -p build",
			`release-info CHANGELOG.md "$(git describe --tags --abbrev=0)" > build/release-info`,
		}),
	})
	j.addStep(jobStep{
		Name: "Run GoReleaser",
		Uses: core.GoreleaserAction,
		With: map[string]any{
			"version": "latest",
			"args":    "release --clean --release-notes=./build/release-info",
		},
		Env: map[string]string{
			"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		},
	})
	w.Jobs = map[string]job{"release": j}

	writeWorkflowToFile(w)
}
