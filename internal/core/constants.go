// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package core

const (
	DefaultAlpineImage         = "3.23"
	DefaultGoVersion           = "1.26.2"
	DefaultPostgresVersion     = "18"
	DefaultLinkerdAwaitVersion = "0.2.7"
	DefaultGitHubComRunsOn     = "ubuntu-latest"
)

// DefaultGitHubEnterpriseRunsOn is a map of group names to runner labels for GitHub Enterprise.
var DefaultGitHubEnterpriseRunsOn = map[string]string{
	"group": "organization/Default",
}

// SugarRunsOn is an array of Sugar runners.
var SugarRunsOn = []string{"self-hosted"}

// GetUploadArtifactAction works around GitHub not supporting their own stuff
// https://github.com/actions/upload-artifact/issues/537
// NOTE: When removing this, also remove the corresponding renovate rule
func GetUploadArtifactAction(isSelfHostedRunner bool) string {
	if isSelfHostedRunner {
		return "actions/upload-artifact@v2"
	} else {
		return "actions/upload-artifact@v7"
	}
}

// GetCodeqlInitAction returns the right CodeQL init action for the chosen Runner.
// see https://github.com/github/codeql-action/releases
// and https://github.wdf.sap.corp/Security-Testing/codeql-action/releases
func GetCodeqlInitAction(isSelfHostedRunner bool) string {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/init@v3"
	} else {
		return "github/codeql-action/init@v4"
	}
}

// GetCodeqlAnalyzeAction returns the right CodeQL analyze action for the chosen Runner.
func GetCodeqlAnalyzeAction(isSelfHostedRunner bool) string {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/analyze@v3"
	} else {
		return "github/codeql-action/analyze@v4"
	}
}

// GetCodeqlAutobuildAction returns the right CodeQL autobild action for the chosen Runner.
func GetCodeqlAutobuildAction(isSelfHostedRunner bool) string {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/autobuild@v3"
	} else {
		return "github/codeql-action/autobuild@v4"
	}
}

const (
	CheckoutAction = "actions/checkout@v6"
	SetupGoAction  = "actions/setup-go@v6"

	DockerLoginAction     = "docker/login-action@v4"
	DockerMetadataAction  = "docker/metadata-action@v6"
	DockerBuildxAction    = "docker/setup-buildx-action@v4"
	DockerQemuAction      = "docker/setup-qemu-action@v4"
	DockerBuildPushAction = "docker/build-push-action@v7"

	DownloadSyftAction     = "anchore/sbom-action/download-syft@v0"
	GoCoverageReportAction = "fgrosse/go-coverage-report@v1.3.0"
	GolangciLintAction     = "golangci/golangci-lint-action@v9"
	GoreleaserAction       = "goreleaser/goreleaser-action@v7"
	ReuseAction            = "fsfe/reuse-action@v6"
	TyposAction            = "crate-ci/typos@v1"
	HelmSetupAction        = "azure/setup-helm@v5"
)
