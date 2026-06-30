// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	. "go.xyrillian.de/gg/option"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

// releasePRWorkflow renders .github/workflows/release-pr.yaml when the
// release-PR automation is opted in. See README for details.
func releasePRWorkflow(cfg core.Configuration) Option[workflow] {
	ghwCfg := cfg.GitHubWorkflow
	w := newWorkflow("release-pr", ghwCfg.Global.DefaultBranch, nil)

	enabled := ghwCfg.Release.Enabled.UnwrapOr(cfg.GoReleaser.ShouldCreateConfig()) && ghwCfg.Release.ReleasePR.UnwrapOr(true)
	if w.deleteUnless(enabled) {
		return None[workflow]()
	}

	w.Permissions.Contents = tokenScopeWrite
	w.Permissions.PullRequests = tokenScopeWrite

	w.On.Push.Branches = []string{ghwCfg.Global.DefaultBranch}
	w.On.PullRequest.Branches = nil
	w.On.WorkflowDispatch.Inputs = map[string]workflowDispatchInput{
		"version": {
			Description: "Version bump type",
			Required:    true,
			Default:     "patch",
			Type:        "choice",
			Options:     []string{"patch", "minor", "major"},
		},
	}

	j := baseJob("draft-release", ghwCfg)
	j.Steps[0].With = map[string]any{
		"fetch-depth": 0,
	}

	j.addStep(jobStep{
		Name: "Check for unreleased changes",
		ID:   "changelog",
		Uses: core.KeepAChangelogAction,
		With: map[string]any{
			"command": "query",
			"version": "unreleased",
		},
	})
	j.addStep(jobStep{
		Name: "Check if unreleased section has content",
		ID:   "check",
		Env: map[string]string{
			"RELEASE_NOTES": "${{ steps.changelog.outputs.release-notes }}",
		},
		Run: makeMultilineYAMLString([]string{
			`if [ -n "$RELEASE_NOTES" ]; then`,
			`  echo "has_changes=true" >> "$GITHUB_OUTPUT"`,
			`else`,
			`  echo "has_changes=false" >> "$GITHUB_OUTPUT"`,
			`fi`,
		}),
	})
	hasChanges := "steps.check.outputs.has_changes == 'true'"
	j.addStep(jobStep{
		Name: "Bump changelog",
		ID:   "bump",
		If:   hasChanges,
		Uses: core.KeepAChangelogAction,
		With: map[string]any{
			"command":                 "bump",
			"version":                 "${{ inputs.version || 'patch' }}",
			"keep-unreleased-section": true,
		},
	})
	j.addStep(jobStep{
		Name: "Run release-prepare make target (if defined)",
		If:   hasChanges,
		Env: map[string]string{
			"VERSION": "${{ steps.bump.outputs.version }}",
		},
		// Project-specific extension hook: if the project's Makefile defines a
		// `release-prepare` target, run it. Projects use it for things like
		// regenerating swagger or bumping web/package.json. We probe with
		// `make -n` (dry-run) to avoid failing when the target does not exist.
		Run: makeMultilineYAMLString([]string{
			`if make -n release-prepare >/dev/null 2>&1; then`,
			`  make release-prepare`,
			`else`,
			`  echo "No release-prepare target defined; skipping."`,
			`fi`,
		}),
	})
	j.addStep(jobStep{
		Name: "Create or update pull request",
		If:   hasChanges,
		Uses: core.CreatePullRequestAction,
		With: map[string]any{
			"branch":         releasePRBranch,
			"base":           ghwCfg.Global.DefaultBranch,
			"commit-message": "chore: release v${{ steps.bump.outputs.version }}",
			"title":          "chore: release v${{ steps.bump.outputs.version }}",
			"body":           releasePRBody,
			"labels":         "release",
		},
	})

	w.Jobs = map[string]job{"draft-release": j}

	return Some(w)
}

const releasePRBody = `Automated release PR for v${{ steps.bump.outputs.version }}.

Merging this PR will tag the release and trigger goreleaser.

### Release Notes

${{ steps.bump.outputs.release-notes }}
`
