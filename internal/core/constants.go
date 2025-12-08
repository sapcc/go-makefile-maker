// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package core

const (
	DefaultAlpineImage         = "3.23"
	DefaultGoVersion           = "1.25.5"
	DefaultPostgresVersion     = "17"
	DefaultLinkerdAwaitVersion = "0.2.7"
	DefaultGitHubComRunsOn     = "ubuntu-latest"
)

var DefaultGitHubEnterpriseRunsOn = map[string]string{
	"group": "organization/Default",
}
var SugarRunsOn = []string{"self-hosted"}

// GetUploadArtifactAction works around GitHub not supporting their own stuff
// https://github.com/actions/upload-artifact/issues/537
// NOTE: When removing this, also remove the corresponding renovate rule
func GetUploadArtifactAction(isSelfHostedRunner bool) string {
	if isSelfHostedRunner {
		return "actions/upload-artifact@v2"
	} else {
		return "actions/upload-artifact@v5"
	}
}

// see https://github.com/github/codeql-action/releases
// and https://github.wdf.sap.corp/Security-Testing/codeql-action/releases
func GetCodeqlInitAction(isSelfHostedRunner bool) string {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/init@v3"
	} else {
		return "github/codeql-action/init@v4"
	}
}
func GetCodeqlAnalyzeAction(isSelfHostedRunner bool) string {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/analyze@v3"
	} else {
		return "github/codeql-action/analyze@v4"
	}
}
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

	DockerLoginAction     = "docker/login-action@v3"
	DockerMetadataAction  = "docker/metadata-action@v5"
	DockerBuildxAction    = "docker/setup-buildx-action@v3"
	DockerQemuAction      = "docker/setup-qemu-action@v3"
	DockerBuildPushAction = "docker/build-push-action@v6"

	DownloadSyftAction     = "anchore/sbom-action/download-syft@v0"
	GoCoverageReportAction = "fgrosse/go-coverage-report@v1.2.0"
	GolangciLintAction     = "golangci/golangci-lint-action@v9"
	GoreleaserAction       = "goreleaser/goreleaser-action@v6"
	ReuseAction            = "fsfe/reuse-action@v6"
	TyposAction            = "crate-ci/typos@v1"
)
