// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	. "go.xyrillian.de/gg/option"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

// releasePRBranch is the branch name used by the release-pr workflow when it
// opens or updates the release PR; the goreleaser workflow's tag job uses the
// same value to filter for that PR's merge event.
const releasePRBranch = "chore/release-next"

func releaseWorkflow(cfg core.Configuration) Option[workflow] {
	// https://docs.github.com/en/packages/managing-github-packages-using-github-actions-workflows/publishing-and-installing-a-package-with-github-actions#publishing-a-package-using-an-action
	ghwCfg := cfg.GitHubWorkflow
	w := newWorkflow("goreleaser", ghwCfg.Global.DefaultBranch, nil)

	if w.deleteUnless(ghwCfg.Release.Enabled.UnwrapOr(cfg.GoReleaser.ShouldCreateConfig())) {
		return None[workflow]()
	}

	releasePR := ghwCfg.Release.ReleasePR.UnwrapOr(true)

	w.Permissions.Contents = tokenScopeWrite
	w.Permissions.Packages = tokenScopeWrite

	w.On.Push.Branches = nil
	w.On.PullRequest.Branches = nil
	w.On.Push.Tags = []string{"*"} // goreleaser uses semver to decide if this is a prerelease or not
	if releasePR {
		// Also fire when the release PR is closed; the tag job below filters for
		// merged PRs from chore/release-next and creates the tag in-process so we
		// stay within a single workflow run (the default GITHUB_TOKEN cannot
		// trigger another workflow via tag push).
		w.On.PullRequest.Branches = []string{ghwCfg.Global.DefaultBranch}
		w.On.PullRequest.Types = []string{"closed"}
	}

	j := baseJobWithGo("goreleaser", cfg)
	// This is needed because: https://goreleaser.com/ci/actions/#fetch-depthness
	j.Steps[0].With = map[string]any{
		"fetch-depth": 0,
	}
	if releasePR {
		j.Needs = []string{"tag"}
		// Run on direct tag pushes OR after the tag job successfully created a tag
		// (always() so this job is reached on push events that skip the tag job).
		j.If = "always() && (github.event_name == 'push' || needs.tag.result == 'success')"
		// On the pull_request path, check out the freshly-pushed tag instead of
		// the PR merge commit so goreleaser sees the right ref.
		j.Steps[0].With["ref"] = "${{ github.event_name == 'pull_request' && format('refs/tags/v{0}', needs.tag.outputs.version) || github.ref }}"
	}
	j.addStep(jobStep{
		Name: "Install syft",
		Uses: core.DownloadSyftAction,
	})
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
		Env: map[string]string{ //nolint:gosec // not a hardcoded secret, we are doing templating here
			"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		},
	})

	w.Jobs = map[string]job{"release": j}
	if releasePR {
		w.Jobs["tag"] = tagJob(ghwCfg)
	}
	return Some(w)
}

// tagJob creates the job that runs on the release-PR merge event; it reads the
// version from the CHANGELOG and pushes a tag, after which the goreleaser job
// (depending on it) runs in the same workflow run.
func tagJob(ghwCfg *core.GithubWorkflowConfiguration) job {
	tj := baseJob("tag", ghwCfg)
	tj.If = "github.event_name == 'pull_request' && github.event.pull_request.merged == true && github.event.pull_request.head.ref == '" + releasePRBranch + "'"
	tj.Outputs = map[string]string{
		"version": "${{ steps.version.outputs.version }}",
	}
	// Need full history to push the tag.
	tj.Steps[0].With = map[string]any{
		"fetch-depth": 0,
	}
	tj.addStep(jobStep{
		Name: "Read version from CHANGELOG",
		ID:   "version",
		Uses: core.KeepAChangelogAction,
		With: map[string]any{
			"command": "query",
			"version": "latest",
		},
	})
	tj.addStep(jobStep{
		Name: "Create and push tag",
		Env: map[string]string{
			"VERSION": "v${{ steps.version.outputs.version }}",
		},
		Run: makeMultilineYAMLString([]string{
			`if git rev-parse "${VERSION}" >/dev/null 2>&1; then`,
			`  echo "Tag ${VERSION} already exists, skipping"`,
			`  exit 0`,
			`fi`,
			`git config user.name "github-actions[bot]"`,
			`git config user.email "41898282+github-actions[bot]@users.noreply.github.com"`,
			`git tag -a "${VERSION}" -m "Release ${VERSION}"`,
			`git push origin "${VERSION}"`,
		}),
	})
	return tj
}
