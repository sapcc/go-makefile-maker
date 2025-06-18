// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package dockerfile

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

var (
	//go:embed Dockerfile.tmpl
	dockerfileTemplate string
	//go:embed dockerignore.tmpl
	dockerignoreTemplate string
)

func RenderConfig(cfg core.Configuration) {
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
	var runVersionCommands []string
	for _, binary := range cfg.Binaries {
		if binary.InstallTo != "" {
            # The binaries need to be in PATH which is only the case if they are installed to bin/
			if binary.InstallTo != "bin/" {
				logg.Error("dockerfile: ignoring binary %q with custom install path %q, only 'bin/' is supported at the moment", binary.Name, binary.InstallTo)
				continue
			}

			runVersionCommands = append(runVersionCommands, binary.Name+" --version 2>/dev/null")
		}
	}

	must.Succeed(util.WriteFileFromTemplate("Dockerfile", dockerfileTemplate, map[string]any{
		"Config": cfg,
		"Constants": map[string]any{
			"DefaultGoVersion":   core.DefaultGoVersion,
			"DefaultAlpineImage": core.DefaultAlpineImage,
		},
		"Entrypoint":         entrypoint,
		"RunCommands":        strings.Join(commands, " \\\n  && "),
		"RunVersionCommands": strings.Join(runVersionCommands, " \\\n  && "),
	}))
	must.Succeed(util.WriteFileFromTemplate(".dockerignore", dockerignoreTemplate, map[string]any{
		"ExtraIgnores": cfg.Dockerfile.ExtraIgnores,
	}))
}
