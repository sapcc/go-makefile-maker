// SPDX-FileCopyrightText: Copyright 2022 SAP SE
// SPDX-License-Identifier: Apache-2.0

package dockerfile

import (
	"fmt"
	"os"
	"strings"

	_ "embed"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

func RenderConfig(cfg core.Configuration) {
	var (
		goBuildflags, packages, userCommand, entrypoint, workingDir, addUserGroup string
		extraCommands                                                             []string
	)

	if cfg.Golang.EnableVendoring {
		goBuildflags = ` GO_BUILDFLAGS='-mod vendor'`
	}

	if cfg.Dockerfile.RunAsRoot {
		userCommand = ""
		workingDir = "/"
		addUserGroup = ""
	} else {
		// this is the same as `USER appuser:appgroup`, but using numeric IDs
		// should allow Kubernetes to validate the `runAsNonRoot` rule without
		// requiring an explicit `runAsUser: 4200` setting in the container spec
		userCommand = "USER 4200:4200\n"
		workingDir = "/home/appuser"
		addUserGroup = strings.ReplaceAll(`RUN addgroup -g 4200 appgroup \
	&& adduser -h /home/appuser -s /sbin/nologin -G appgroup -D -u 4200 appuser

`, "\t", "  ")
	}

	// if there is an entrypoint configured use that otherwise fallback to the binary name
	if len(cfg.Dockerfile.Entrypoint) > 0 {
		entrypoint = fmt.Sprintf(`"%s"`, strings.Join(cfg.Dockerfile.Entrypoint, `", "`))
	} else {
		entrypoint = fmt.Sprintf(`"/usr/bin/%s"`, cfg.Binaries[0].Name)
	}

	if cfg.Dockerfile.WithLinkerdAwait {
		extraCommands = []string{
			fmt.Sprintf(
				"wget -qO /usr/bin/linkerd-await https://github.com/linkerd/linkerd-await/releases/download/release%%2Fv%[1]s/linkerd-await-v%[1]s-amd64",
				core.DefaultLinkerdAwaitVersion,
			),
			"chmod 755 /usr/bin/linkerd-await",
		}
		// add linkrd-await after the fallback for entrypoint has been set
		entrypoint = `"/usr/bin/linkerd-await", "--shutdown", "--", ` + entrypoint
	}

	var runVersionArg string
	firstBinary := true
	if len(cfg.Binaries) > 0 {
		runVersionArg += "RUN "
	}
	for _, binary := range cfg.Binaries {
		if binary.InstallTo == "" {
			continue
		}
		if !firstBinary {
			runVersionArg += strings.ReplaceAll(` \
	&& `, "\t", "  ")
		}
		runVersionArg += binary.Name + " --version 2>/dev/null"
		firstBinary = false
	}

	extraBuildStages := ""
	for _, stage := range cfg.Dockerfile.ExtraBuildStages {
		extraBuildStages += fmt.Sprintf("%s\n\n%s\n\n", strings.TrimSpace(stage), strings.Repeat("#", 80))
	}

	extraDirectives := strings.Join(cfg.Dockerfile.ExtraDirectives, "\n")
	if extraDirectives != "" {
		extraDirectives += "\n"
	}

	for _, v := range cfg.Dockerfile.ExtraPackages {
		packages += " " + v
	}

	commands := []string{
		"apk upgrade --no-cache --no-progress",
	}
	if packages != "" {
		commands = append(commands, "apk add --no-cache --no-progress"+packages)
	}
	commands = append(commands, extraCommands...)
	commands = append(commands, "apk del --no-cache --no-progress apk-tools alpine-keys alpine-release libc-utils")

	runCommands := strings.Join(commands, " \\\n  && ")

	dockerfile := fmt.Sprintf(strings.ReplaceAll(`%[1]sFROM golang:%[2]s-alpine%[3]s AS builder

RUN apk add --no-cache --no-progress ca-certificates gcc git make musl-dev

COPY . /src
ARG BININFO_BUILD_DATE BININFO_COMMIT_HASH BININFO_VERSION # provided to 'make install'
RUN make -C /src install PREFIX=/pkg GOTOOLCHAIN=local%[4]s

################################################################################

FROM alpine:%[3]s

%[5]s# upgrade all installed packages to fix potential CVEs in advance
# also remove apk package manager to hopefully remove dependency on OpenSSL 🤞
RUN %[6]s

COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=builder /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY --from=builder /pkg/ /usr/
# make sure all binaries can be executed
%[7]s

ARG BININFO_BUILD_DATE BININFO_COMMIT_HASH BININFO_VERSION
LABEL source_repository="%[8]s" \
	org.opencontainers.image.url="%[8]s" \
	org.opencontainers.image.created=${BININFO_BUILD_DATE} \
	org.opencontainers.image.revision=${BININFO_COMMIT_HASH} \
	org.opencontainers.image.version=${BININFO_VERSION}

%[9]s%[10]sWORKDIR %[11]s
ENTRYPOINT [ %[12]s ]
`, "\t", "  "), extraBuildStages, core.DefaultGoVersion, core.DefaultAlpineImage, goBuildflags, addUserGroup, runCommands, runVersionArg, cfg.Metadata.URL, extraDirectives, userCommand, workingDir, entrypoint)

	must.Succeed(os.WriteFile("Dockerfile", []byte(dockerfile), 0666))

	dockerignoreLines := append([]string{
		`/.dockerignore`,
		`.DS_Store`,
		`# TODO: uncomment when applications no longer use git to get version information`,
		`#.git/`,
		`/.github/`,
		`/.gitignore`,
		`/.goreleaser.yml`,
		`/*.env*`,
		`/.golangci.yaml`,
		`/.vscode/`,
		`/build/`,
		`/CONTRIBUTING.md`,
		`/Dockerfile`,
		`/docs/`,
		`/LICENSE*`,
		`/Makefile.maker.yaml`,
		`/README.md`,
		`/report.html`,
		`/shell.nix`,
		`/testing/`,
	}, cfg.Dockerfile.ExtraIgnores...)
	dockerignore := strings.Join(dockerignoreLines, "\n") + "\n"

	must.Succeed(os.WriteFile(".dockerignore", []byte(dockerignore), 0666))
}
