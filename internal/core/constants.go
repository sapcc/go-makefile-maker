// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package core

import "github.com/sapcc/go-makefile-maker/internal/util"

const (
	DefaultAlpineImage         = "3.24"
	DefaultGoVersion           = "1.26.5"
	DefaultPostgresVersion     = "18"
	DefaultLinkerdAwaitVersion = "0.3.3"
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
func GetUploadArtifactAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "actions/upload-artifact@82c141cc518b40d92cc801eee768e7aafc9c2fa2 # v2"
	} else {
		return "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7"
	}
}

// GetCodeqlInitAction returns the right CodeQL init action for the chosen Runner.
// see https://github.com/github/codeql-action/releases
// and https://github.wdf.sap.corp/Security-Testing/codeql-action/releases
func GetCodeqlInitAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/init@14e82a807226aece1a9f38735d8c69d48c26627f # v4"
	} else {
		return "github/codeql-action/init@99df26d4f13ea111d4ec1a7dddef6063f76b97e9 # v4"
	}
}

// GetCodeqlAnalyzeAction returns the right CodeQL analyze action for the chosen Runner.
func GetCodeqlAnalyzeAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/analyze@14e82a807226aece1a9f38735d8c69d48c26627f # v4"
	} else {
		return "github/codeql-action/analyze@99df26d4f13ea111d4ec1a7dddef6063f76b97e9 # v4"
	}
}

// GetCodeqlAutobuildAction returns the right CodeQL autobild action for the chosen Runner.
func GetCodeqlAutobuildAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/autobuild@14e82a807226aece1a9f38735d8c69d48c26627f # v4"
	} else {
		return "github/codeql-action/autobuild@99df26d4f13ea111d4ec1a7dddef6063f76b97e9 # v4"
	}
}

const (
	CheckoutAction = util.RawString("actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7")
	SetupGoAction  = util.RawString("actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6")

	DockerLoginAction     = util.RawString("docker/login-action@af1e73f918a031802d376d3c8bbc3fe56130a9b0 # v4")
	DockerMetadataAction  = util.RawString("docker/metadata-action@dc802804100637a589fabce1cb79ff13a1411302 # v6")
	DockerBuildxAction    = util.RawString("docker/setup-buildx-action@bb05f3f5519dd87d3ba754cc423b652a5edd6d2c # v4")
	DockerQemuAction      = util.RawString("docker/setup-qemu-action@96fe6ef7f33517b61c61be40b68a1882f3264fb8 # v4")
	DockerBuildPushAction = util.RawString("docker/build-push-action@53b7df96c91f9c12dcc8a07bcb9ccacbed38856a # v7")

	CreatePullRequestAction = util.RawString("peter-evans/create-pull-request@5f6978faf089d4d20b00c7766989d076bb2fc7f1 # v8.1.1")
	DownloadSyftAction      = util.RawString("anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610 # v0")
	GHCRCleanupAction       = util.RawString("dataaxiom/ghcr-cleanup-action@d52806a0dc70b430571a37da1fde39733ffd640f # v1")
	GoCoverageReportAction  = util.RawString("fgrosse/go-coverage-report@cbeb2ab2e32591d690337146ba02a911cc566f3f # v1.3.0")
	GolangCiLintVersion     = "v2.12.2"
	GolangciLintAction      = util.RawString("golangci/golangci-lint-action@ba0d7d2ec06a0ea1cb5fa41b2e4a3ab91d21278a # v9")
	GoreleaserAction        = util.RawString("goreleaser/goreleaser-action@f06c13b6b1a9625abc9e6e439d9c05a8f2190e94 # v7")
	HelmSetupAction         = util.RawString("azure/setup-helm@9bc31f4ebc9c6b171d7bfbaa5d006ae7abdb4310 # v5")
	KeepAChangelogAction    = util.RawString("release-flow/keep-a-changelog-action@74931dec7ecdbfc8e38ac9ae7e8dd84c08db2f32 # v3.0.0")
	ReuseAction             = util.RawString("fsfe/reuse-action@676e2d560c9a403aa252096d99fcab3e1132b0f5 # v6")
	TyposAction             = util.RawString("crate-ci/typos@bee27e3a4fd1ea2111cf90ab89cd076c870fce14 # v1")
)
