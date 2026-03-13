// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	"slices"
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

func helmWorkflow(cfg core.Configuration) {
	// https://docs.github.com/en/packages/managing-github-packages-using-github-actions-workflows/publishing-and-installing-a-package-with-github-actions#publishing-a-package-using-an-action
	w := newWorkflow("Helm OCI Package GHCR", cfg.GitHubWorkflow.Global.DefaultBranch, nil)

	if w.deleteIf(cfg.GitHubWorkflow.PushHelmChartToGhcr.Path.IsSome() &&
		strings.HasPrefix(cfg.Metadata.URL, "https://github.com")) {
		return
	}

	var helmConfig = cfg.GitHubWorkflow.PushHelmChartToGhcr
	var chartPath = helmConfig.Path.UnwrapOr(".")

	w.Permissions.Contents = tokenScopeRead
	w.Permissions.Packages = tokenScopeWrite

	w.On.WorkflowDispatch.manualTrigger = true

	var helmPackageCmds []string
	strategy := cfg.GitHubWorkflow.PushContainerToGhcr.TagStrategy
	if !helmConfig.DisableVersioning {
		if slices.Contains(strategy, "semver") {
			// try to detect a version from the git tags
			w.On.Push.Tags = []string{"*"}
			helmPackageCmds = append(helmPackageCmds,
				`# try to detect a version from the git tags, only if the tag is exactly on the commit`,
				`APP_VERSION=$(git describe --tags --exact-match ${{ github.sha }} 2>/dev/null || echo "")`,
				`if [ -n "$APP_VERSION" ]; then`,
				`  VERSION=$(echo -n "$APP_VERSION" | sed -E 's/^v//')`,
				`fi`,
				``,
			)
		}
		if slices.Contains(strategy, "sha") {
			w.On.Push.Branches = []string{cfg.GitHubWorkflow.Global.DefaultBranch}
			// use the git sha as version, if no version could be detected from the tags
			helmPackageCmds = append(helmPackageCmds,
				`# use the git sha as app-version, if no version could be detected from the tags`,
				`if [ -z "$APP_VERSION" ]; then`,
				`  APP_VERSION=$(echo -n "sha-${{ github.sha }}")`,
				`fi`,
				`# use the git sha as helm version suffix, version is semver`,
				`if [ -z "$VERSION" ] && [ -n "$APP_VERSION" ]; then`,
				`  VERSION="$(helm show chart `+chartPath+` | grep -E "^version:" | awk '{print $2}' )+${APP_VERSION:0:11}"`,
				`fi`,
				``,
			)
		}
	}
	w.On.PullRequest.Branches = nil

	if !slices.Contains(strategy, "sha") && !slices.Contains(strategy, "semver") {
		// Just upload any chart change
		w.On.Push.Paths = []string{chartPath + "/**"}
	}

	if helmConfig.DependencyUpdate.UnwrapOr(true) {
		helmPackageCmds = append(helmPackageCmds, `HELM_ARGS=--dependency-update`)
	}

	const registry = "ghcr.io"
	j := baseJob("Build and publish Helm Chart OCI", cfg.GitHubWorkflow)
	j.Steps[0].With = map[string]any{
		"fetch-depth": 0,    // we need the full git history to be able to detect versions from tags
		"fetch-tags":  true, // we need the tags to be able to detect versions from tags
	}
	j.addStep(jobStep{
		Name: "Install Helm",
		Uses: core.HelmSetupAction,
	})
	if helmConfig.Lint.UnwrapOr(true) {
		j.addStep(jobStep{
			Name: "Lint Helm Chart",
			Run:  "helm lint " + chartPath,
		})
	}
	j.addStep(jobStep{
		Name: "Package Helm Chart",
		Run: makeMultilineYAMLString(append(
			helmPackageCmds,
			`if [ -n "$APP_VERSION" ]; then`,
			`  HELM_ARGS="$HELM_ARGS --app-version $APP_VERSION"`,
			`fi`,
			`if [ -n "$VERSION" ]; then`,
			`  HELM_ARGS="$HELM_ARGS --version $VERSION"`,
			`fi`,
			`echo "Running helm package with $HELM_ARGS"`,
			`helm package `+chartPath+` --destination ./chart $HELM_ARGS`,
		)),
	})
	j.addStep(jobStep{
		Name: "Log in to the Container registry",
		Uses: core.DockerLoginAction,
		With: map[string]any{ //nolint:gosec // not a hardcoded secret, we are doing templating here
			"registry": registry,
			"username": "${{ github.actor }}",
			"password": "${{ secrets.GITHUB_TOKEN }}",
		},
	})
	j.addStep(jobStep{
		Name: "Push Helm Chart to " + registry,
		Run:  "helm push ./chart/*.tgz oci://" + registry + "/${{ github.repository_owner }}/charts",
	})
	w.Jobs = map[string]job{"build-and-push-helm-package": j}

	writeWorkflowToFile(w)
}
