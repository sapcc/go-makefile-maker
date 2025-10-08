// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package core

const (
	DefaultAlpineImage         = "3.22"
	DefaultGoVersion           = "1.25.2"
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
func GetUploadArtifactAction(isSelfHostedRunner bool) string {
	if isSelfHostedRunner {
		return "actions/upload-artifact@v2"
	} else {
		return "actions/upload-artifact@v4"
	}
}

const (
	CheckoutAction = "actions/checkout@v5"
	SetupGoAction  = "actions/setup-go@v6"

	DockerLoginAction     = "docker/login-action@v3"
	DockerMetadataAction  = "docker/metadata-action@v5"
	DockerBuildxAction    = "docker/setup-buildx-action@v3"
	DockerQemuAction      = "docker/setup-qemu-action@v3"
	DockerBuildPushAction = "docker/build-push-action@v6"

	CodeqlInitAction      = "github/codeql-action/init@v3"
	CodeqlAnalyzeAction   = "github/codeql-action/analyze@v3"
	CodeqlAutobuildAction = "github/codeql-action/autobuild@v3"

	DownloadSyftAction     = "anchore/sbom-action/download-syft@v0"
	GoCoverageReportAction = "fgrosse/go-coverage-report@v1.2.0"
	GolangciLintAction     = "golangci/golangci-lint-action@v8"
	GoreleaserAction       = "goreleaser/goreleaser-action@v6"
	MisspellAction         = "reviewdog/action-misspell@v1"
	ReuseAction            = "fsfe/reuse-action@v5"
)
