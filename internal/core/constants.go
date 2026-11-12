// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package core

import "github.com/sapcc/go-makefile-maker/internal/util"

const (
	DefaultAlpineImage         = "3.24"
	DefaultGoVersion           = "1.27.4"
	DefaultPostgresVersion     = "18"
	DefaultLinkerdAwaitVersion = "0.3.3"
	DefaultGitHubComRunsOn     = "ubuntu-latest"
)

// DefaultGitHubEnterpriseRunsOn is a map of group names to runner labels for GitHub Enterprise.
var DefaultGitHubEnterpriseRunsOn = map[string]string{
	"group": "organization/Default",
}

// SciRunsOn selects SALT runners for the SCI organization.
var SciRunsOn = []string{"sci"}

// GetUploadArtifactAction works around GitHub not supporting their own stuff
// https://github.com/actions/upload-artifact/issues/537
// NOTE: When removing this, also remove the corresponding renovate rule
func GetUploadArtifactAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "actions/upload-artifact@82c141cc518b40d92cc801eee768e7aafc9c2fa2 # v2"
	} else {
		return "actions/upload-artifact@cf430e030ddbb5b0abf93d22962f4752f3646cd9 # v7"
	}
}

// GetCodeqlInitAction returns the right CodeQL init action for the chosen Runner.
// see https://github.com/github/codeql-action/releases
// and https://github.wdf.sap.corp/Security-Testing/codeql-action/releases
func GetCodeqlInitAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/init@6551ad85cdc2ebf529cfd121b0f4879b2c3d91e6 # v4"
	} else {
		return "github/codeql-action/init@33d7b85da1a04171e211befe2c739303b9a4c066 # v4"
	}
}

// GetCodeqlAnalyzeAction returns the right CodeQL analyze action for the chosen Runner.
func GetCodeqlAnalyzeAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/analyze@6551ad85cdc2ebf529cfd121b0f4879b2c3d91e6 # v4"
	} else {
		return "github/codeql-action/analyze@33d7b85da1a04171e211befe2c739303b9a4c066 # v4"
	}
}

// GetCodeqlAutobuildAction returns the right CodeQL autobild action for the chosen Runner.
func GetCodeqlAutobuildAction(isSelfHostedRunner bool) util.RawString {
	if isSelfHostedRunner {
		return "Security-Testing/codeql-action/autobuild@6551ad85cdc2ebf529cfd121b0f4879b2c3d91e6 # v4"
	} else {
		return "github/codeql-action/autobuild@33d7b85da1a04171e211befe2c739303b9a4c066 # v4"
	}
}

const (
	CheckoutAction     = util.RawString("actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7")
	SetupGoAction      = util.RawString("actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7")
	SetupNodeAction    = util.RawString("actions/setup-node@949feb2413d6458794dcd2491c4babbbce0c15c1 # v7")
	GitHubScriptAction = util.RawString("actions/github-script@3a2844b7e9c422d3c10d287c895573f7108da1b3 # v9")

	DockerLoginAction     = util.RawString("docker/login-action@8aa5831fb8d48d8f76594c7ddde61ae58019f550 # v4")
	DockerMetadataAction  = util.RawString("docker/metadata-action@dc802804100637a589fabce1cb79ff13a1411302 # v6")
	DockerBuildxAction    = util.RawString("docker/setup-buildx-action@f87e5991a6d7451dcb8d9637bfbc97413f497069 # v4")
	DockerQemuAction      = util.RawString("docker/setup-qemu-action@99012661954931238ded8c8b007157a8430204e1 # v4")
	DockerBuildPushAction = util.RawString("docker/build-push-action@c3c9e263c25d99ce0380d002d59b67737d91b0dc # v7")

	CreatePullRequestAction = util.RawString("peter-evans/create-pull-request@3ffccacd46053766916f087ce41d627c0b8cf2d1 # v8")
	DownloadSyftAction      = util.RawString("anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610 # v0")
	GHCRCleanupAction       = util.RawString("dataaxiom/ghcr-cleanup-action@d52806a0dc70b430571a37da1fde39733ffd640f # v1")
	GoCoverageReportAction  = util.RawString("fgrosse/go-coverage-report@c4bf7e78d1379325814b4dd42d80e526082e788b # v1.5.0")
	GolangCiLintVersion     = "v2.15.0"
	GolangciLintAction      = util.RawString("golangci/golangci-lint-action@ba0d7d2ec06a0ea1cb5fa41b2e4a3ab91d21278a # v9")
	GoreleaserAction        = util.RawString("goreleaser/goreleaser-action@a4f02ab5355a589de60d917a659ce617ed19b3de # v7")
	HelmSetupAction         = util.RawString("azure/setup-helm@2f5e7bc67f1f9972ab16657bb795f36d8f04de0e # v5")
	KeepAChangelogAction    = util.RawString("release-flow/keep-a-changelog-action@74931dec7ecdbfc8e38ac9ae7e8dd84c08db2f32 # v3")
	ReuseAction             = util.RawString("fsfe/reuse-action@676e2d560c9a403aa252096d99fcab3e1132b0f5 # v6")
	TyposAction             = util.RawString("crate-ci/typos@2611abd15acb0c2e317fb9e45855aabb8b35a414 # v1")
)
