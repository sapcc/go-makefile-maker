// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package dockerfile

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

var (
	//go:embed Dockerfile.tmpl
	dockerfileTemplate string
	//go:embed dockerignore.tmpl
	dockerignoreTemplate string
)

func RenderConfig(cfg core.Configuration, sr golang.ScanResult) {
	// if there is an entrypoint configured use that otherwise fallback to the first binary name
	var entrypoint string
	if len(cfg.Dockerfile.Entrypoint) > 0 {
		entrypoint = fmt.Sprintf(`"%s"`, strings.Join(cfg.Dockerfile.Entrypoint, `", "`))
	} else {
		entrypoint = fmt.Sprintf(`"/usr/bin/%s"`, cfg.Binaries[0].Name)
	}

	// these commands will be run early on to install dependencies
	commands := []string{
		"apk upgrade --no-cache --no-progress",
	}
	if pkgs := cfg.Dockerfile.ExtraPackages; len(pkgs) > 0 {
		commands = append(commands, "apk add --no-cache --no-progress "+strings.Join(pkgs, " "))
	}
	if cfg.Dockerfile.WithLinkerdAwait {
		commands = append(commands,
			fmt.Sprintf(
				"wget -qO /usr/bin/linkerd-await https://github.com/linkerd/linkerd-await/releases/download/release%%2Fv%[1]s/linkerd-await-v%[1]s-amd64",
				core.DefaultLinkerdAwaitVersion,
			),
			"chmod 755 /usr/bin/linkerd-await",
		)
		entrypoint = `"/usr/bin/linkerd-await", "--shutdown", "--", ` + entrypoint
	}
	commands = append(commands, "apk del --no-cache --no-progress apk-tools alpine-keys alpine-release libc-utils")

	// these commands will be run after `make install` to see that all installed commands can be executed
	// (e.g. that all required shared libraries can be loaded correctly)
	var (
		runVersionCommands []string
		pathsToCopy        = make(map[string]string)
	)
	for _, binary := range cfg.Binaries {
		switch {
		case binary.InstallTo == "":
			continue
		case binary.InstallTo == "bin/":
			// The binaries need to be in PATH which is only the case if they are installed to bin/
			pathsToCopy["/pkg"] = "/usr"
			runVersionCommands = append(runVersionCommands, binary.Name+" --version 2>/dev/null")
		case filepath.Clean(binary.InstallTo) == "/opt/resource":
			// special case: Concourse resource type binaries are accessed through symlinks in /opt/resource that `make install` generates
			pathsToCopy["/opt/resource"] = "/opt/resource"
			runVersionCommands = append(runVersionCommands,
				"/opt/resource/check --version 2>/dev/null",
				"/opt/resource/in --version 2>/dev/null",
				"/opt/resource/out --version 2>/dev/null",
			)
		default:
			logg.Error("dockerfile: ignoring binary %q with custom install path %q, only 'bin/' is supported at the moment", binary.Name, binary.InstallTo)
		}
	}

	var dockerHubMirror string
	if strings.HasPrefix(cfg.Metadata.URL, "https://github.wdf.sap.corp") {
		dockerHubMirror = "keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/"
	}

	var extraTestPackages []string
	reuseEnabled := cfg.Reuse.Enabled.UnwrapOr(true)
	if reuseEnabled {
		extraTestPackages = append(extraTestPackages, "py3-pip")
	}
	if sr.UsesPostgres {
		extraTestPackages = append(extraTestPackages, "postgresql")
	}

	must.Succeed(util.WriteFileFromTemplate("Dockerfile", dockerfileTemplate, map[string]any{
		"Config": cfg,
		"Constants": map[string]any{
			"DefaultGoVersion":   core.DefaultGoVersion,
			"DefaultAlpineImage": core.DefaultAlpineImage,
		},
		"DockerHubMirror":    dockerHubMirror,
		"CheckEnv":           cfg.Dockerfile.CheckEnv,
		"ExtraTestPackages":  extraTestPackages,
		"Entrypoint":         entrypoint,
		"PathsToCopy":        pathsToCopy,
		"ReuseEnabled":       reuseEnabled,
		"RunCommands":        strings.Join(commands, " \\\n  && "),
		"RunVersionCommands": strings.Join(runVersionCommands, " \\\n  && "),
		"UseBuildKit":        cfg.Dockerfile.UseBuildKit,
	}))

	ignores := cfg.Dockerfile.ExtraIgnores
	if sr.UsesPostgres {
		ignores = append(ignores, "/.testdb/")
	}
	must.Succeed(util.WriteFileFromTemplate(".dockerignore", dockerignoreTemplate, map[string]any{
		"ExtraIgnores": ignores,
	}))
}
