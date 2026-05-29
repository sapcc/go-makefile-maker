// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package core

import "github.com/sapcc/go-makefile-maker/internal/util"

const (
	DefaultAlpineImage         = "3.23"
	DefaultGoVersion           = "1.26.3"
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
		return "github/codeql-action/init@7211b7c8077ea37d8641b6271f6a365a22a5fbfa # v4"
	}
}

// GetCodeqlAnalyzeAction returns the right CodeQL analyze action for the chosen Runner.
func GetCodeqlAnalyzeAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/analyze@14e82a807226aece1a9f38735d8c69d48c26627f # v4"
	} else {
		return "github/codeql-action/analyze@7211b7c8077ea37d8641b6271f6a365a22a5fbfa # v4"
	}
}

// GetCodeqlAutobuildAction returns the right CodeQL autobild action for the chosen Runner.
func GetCodeqlAutobuildAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/autobuild@14e82a807226aece1a9f38735d8c69d48c26627f # v4"
	} else {
		return "github/codeql-action/autobuild@7211b7c8077ea37d8641b6271f6a365a22a5fbfa # v4"
	}
}

const (
	CheckoutAction = util.RawString("actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6")
	SetupGoAction  = util.RawString("actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6")

	DockerLoginAction     = util.RawString("docker/login-action@650006c6eb7dba73a995cc03b0b2d7f5ca915bee # v4")
	DockerMetadataAction  = util.RawString("docker/metadata-action@80c7e94dd9b9319bd5eb7a0e0fe9291e23a2a2e9 # v6")
	DockerBuildxAction    = util.RawString("docker/setup-buildx-action@d7f5e7f509e45cec5c76c4d5afdd7de93d0b3df5 # v4")
	DockerQemuAction      = util.RawString("docker/setup-qemu-action@06116385d9baf250c9f4dcb4858b16962ea869c3 # v4")
	DockerBuildPushAction = util.RawString("docker/build-push-action@f9f3042f7e2789586610d6e8b85c8f03e5195baf # v7")

	DownloadSyftAction     = util.RawString("anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610 # v0")
	GoCoverageReportAction = util.RawString("fgrosse/go-coverage-report@cbeb2ab2e32591d690337146ba02a911cc566f3f # v1.3.0")
	GolangCiLintVersion    = "v2.12.2"
	GolangciLintAction     = util.RawString("golangci/golangci-lint-action@82606bf257cbaff209d206a39f5134f0cfbfd2ee # v9")
	GoreleaserAction       = util.RawString("goreleaser/goreleaser-action@5daf1e915a5f0af01ddbcd89a43b8061ff4f1a89 # v7")
	ReuseAction            = util.RawString("fsfe/reuse-action@676e2d560c9a403aa252096d99fcab3e1132b0f5 # v6")
	TyposAction            = util.RawString("crate-ci/typos@f8a58b6b53f2279f71eb605f03a4ae4d10608f45 # v1")
	HelmSetupAction        = util.RawString("azure/setup-helm@dda3372f752e03dde6b3237bc9431cdc2f7a02a2 # v5")
)
