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
	"strconv"
	"strings"

	_ "embed"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

func mustI(_ int, err error) {
	util.Must(err)
}

//go:embed Dockerfile.template
var template []byte

func RenderConfig(cfg core.Configuration) error {
	var goBuildflags, packages, user, entrypoint string

	if cfg.Vendoring.Enabled {
		goBuildflags = ` GO_BUILDFLAGS='-mod vendor'`
	}

	for _, v := range append([]string{"ca-certificates"}, cfg.Dockerfile.ExtraPackages...) {
		packages += fmt.Sprintf(" %s", v)
	}

	if cfg.Dockerfile.User != "" {
		user = cfg.Dockerfile.User
	} else {
		user = "nobody"
	}

	if len(cfg.Dockerfile.Entrypoint) > 0 {
		entrypoint = strconv.Quote(strings.Join(cfg.Dockerfile.Entrypoint, `", "`))
	} else {
		entrypoint = fmt.Sprintf(`"/usr/bin/%s"`, cfg.Binaries[0].Name)
	}

	dockerfile := fmt.Sprintf(
		`%[1]s
FROM golang:${GOLANG_VERSION}${ALPINE_VERSION} as builder
RUN apk add --no-cache gcc git make musl-dev

COPY . /src
RUN make -C /src install PREFIX=/pkg%[2]s

################################################################################

FROM alpine:${ALPINE_VERSION}

RUN apk add --no-cache%[4]s
COPY --from=builder /pkg/ /usr/

ARG COMMIT_ID=unknown
LABEL source_repository="%[3]s" \
  org.opencontainers.image.url="%[3]s" \
  org.opencontainers.image.revision=$(COMMIT_ID)

USER %[5]s:%[5]s
WORKDIR /var/empty
ENTRYPOINT [ %[6]s ]
`, template, goBuildflags, cfg.Metadata.URL, packages, user, entrypoint)

	f, err := os.Create("Dockerfile")
	util.Must(err)

	mustI(f.WriteString(dockerfile))
	util.Must(f.Close())

	f, err = os.Create(".dockerignore")
	util.Must(err)

	mustI(f.WriteString(
		`.dockerignore
# TODO: uncomment when applications no longer use git to get version information
#.git/
.github/
.gitignore
.goreleaser.yml
/*.env*
.golangci.yaml
build/
CONTRIBUTING.md
Dockerfile
docs/
LICENSE*
Makefile.maker.yaml
README.md
report.html
shell.nix
/testing/
`))
	mustI(f.WriteString(strings.Join(cfg.Dockerfile.ExtraIgnores, "\n") + "\n"))
	return f.Close()
}
