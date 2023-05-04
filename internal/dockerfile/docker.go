// Copyright 2022 SAP SE
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

package dockerfile

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	_ "embed"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

//go:embed Dockerfile.template
var template []byte

var argStatementRx = regexp.MustCompile(`^ARG\s*(\w+)\s*=\s*(.+?)\s*$`)

func RenderConfig(cfg core.Configuration) {
	if cfg.Dockerfile.User != "" {
		if cfg.Dockerfile.User == "root" {
			logg.Fatal("the `dockerfile.user` config option has been removed; set `dockerfile.runAsRoot` if you need to run as root")
		} else {
			logg.Fatal("the `dockerfile.user` config option has been removed; commands now run as user `appuser` (ID 4200) in group `appgroup` (ID 4200)")
		}
	}

	//read "ARG" statements from `Dockerfile.template`
	buildArgs := make(map[string]string)
	for _, line := range strings.Split(string(template), "\n") {
		match := argStatementRx.FindStringSubmatch(line)
		if match != nil {
			buildArgs[match[1]] = match[2]
		}
	}

	var goBuildflags, packages, userCommand, entrypoint, workingDir string

	if cfg.Vendoring.Enabled {
		goBuildflags = ` GO_BUILDFLAGS='-mod vendor'`
	}

	for _, v := range append([]string{"ca-certificates"}, cfg.Dockerfile.ExtraPackages...) {
		packages += fmt.Sprintf(" %s", v)
	}

	if cfg.Dockerfile.RunAsRoot {
		userCommand = ""
		workingDir = "/"
	} else {
		// this is the same as `USER appuser:appgroup`, but using numeric IDs
		// should allow Kubernetes to validate the `runAsNonRoot` rule without
		// requiring an explicit `runAsUser: 4200` setting in the container spec
		userCommand = "USER 4200:4200\n"
		workingDir = "/home/appuser"
	}

	if len(cfg.Dockerfile.Entrypoint) > 0 {
		entrypoint = fmt.Sprintf(`"%s"`, strings.Join(cfg.Dockerfile.Entrypoint, `", "`))
	} else {
		entrypoint = fmt.Sprintf(`"/usr/bin/%s"`, cfg.Binaries[0].Name)
	}

	extraDirectives := strings.Join(cfg.Dockerfile.ExtraDirectives, "\n")
	if extraDirectives != "" {
		extraDirectives += "\n"
	}

	dockerfile := fmt.Sprintf(
		`FROM golang:%[1]s%[2]s as builder

RUN apk add --no-cache --no-progress gcc git make musl-dev

COPY . /src
ARG BININFO_BUILD_DATE BININFO_COMMIT_HASH BININFO_VERSION # provided to 'make install'
RUN make -C /src install PREFIX=/pkg%[3]s

################################################################################

FROM alpine:%[2]s

RUN addgroup -g 4200 appgroup
RUN adduser -h /home/appuser -s /sbin/nologin -G appgroup -D -u 4200 appuser
# upgrade all installed packages to fix potential CVEs in advance
RUN apk upgrade --no-cache --no-progress \
  && apk add --no-cache --no-progress%[5]s
COPY --from=builder /pkg/ /usr/

ARG BININFO_BUILD_DATE BININFO_COMMIT_HASH BININFO_VERSION
LABEL source_repository="%[4]s" \
  org.opencontainers.image.url="%[4]s" \
  org.opencontainers.image.created=${BININFO_BUILD_DATE} \
  org.opencontainers.image.revision=${BININFO_COMMIT_HASH} \
  org.opencontainers.image.version=${BININFO_VERSION}

%[6]s%[7]sWORKDIR %[8]s
ENTRYPOINT [ %[9]s ]
`, buildArgs["GOLANG_VERSION"], buildArgs["ALPINE_VERSION"], goBuildflags, cfg.Metadata.URL, packages, extraDirectives, userCommand, workingDir, entrypoint)

	must.Succeed(os.WriteFile("Dockerfile", []byte(dockerfile), 0666))

	dockerignoreLines := append([]string{
		`.dockerignore`,
		`# TODO: uncomment when applications no longer use git to get version information`,
		`#.git/`,
		`.github/`,
		`.gitignore`,
		`.goreleaser.yml`,
		`/*.env*`,
		`.golangci.yaml`,
		`build/`,
		`CONTRIBUTING.md`,
		`Dockerfile`,
		`docs/`,
		`LICENSE*`,
		`Makefile.maker.yaml`,
		`README.md`,
		`report.html`,
		`shell.nix`,
		`/testing/`,
	}, cfg.Dockerfile.ExtraIgnores...)
	dockerignore := strings.Join(dockerignoreLines, "\n") + "\n"

	must.Succeed(os.WriteFile(".dockerignore", []byte(dockerignore), 0666))
}
