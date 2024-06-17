// Copyright 2023 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

const (
	DefaultAlpineImage         = "3.20"
	DefaultGoVersion           = "1.22.4"
	DefaultPostgresVersion     = "16"
	DefaultLinkerdAwaitVersion = "0.2.7"
	DefaultGitHubComRunnerType = "ubuntu-latest"
)

var DefaultGitHubEnterpriseRunnerType = [...]string{"self-hosted"}

const (
	CheckoutAction = "actions/checkout@v4"
	SetupGoAction  = "actions/setup-go@v5"

	DockerLoginAction     = "docker/login-action@v3"
	DockerMetadataAction  = "docker/metadata-action@v5"
	DockerBuildxAction    = "docker/setup-buildx-action@v3"
	DockerQemuAction      = "docker/setup-qemu-action@v3"
	DockerBuildPushAction = "docker/build-push-action@v6"

	CodeqlInitAction      = "github/codeql-action/init@v3"
	CodeqlAnalyzeAction   = "github/codeql-action/analyze@v3"
	CodeqlAutobuildAction = "github/codeql-action/autobuild@v3"

	GolangciLintAction = "golangci/golangci-lint-action@v6"
	GoreleaserAction   = "goreleaser/goreleaser-action@v6"
	GovulncheckAction  = "golang/govulncheck-action@v1"
	MisspellAction     = "reviewdog/action-misspell@v1"
)
