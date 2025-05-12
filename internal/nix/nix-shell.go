// SPDX-FileCopyrightText: 2024 SAP SE
// SPDX-License-Identifier: Apache-2.0

package nix

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/template"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

var (
	//go:embed shell.nix.tmpl
	shellNixTemplate string
	//go:embed envrc.tmpl
	envrcTemplate []byte
)

func RenderShell(cfg core.Configuration, sr golang.ScanResult, renderGoreleaserConfig bool) {
	goVersionSlice := strings.Split(core.DefaultGoVersion, ".")
	goPackage := fmt.Sprintf("go_%s_%s", goVersionSlice[0], goVersionSlice[1])
	packages := []string{
		goPackage,
		"addlicense",
		"go-licence-detector",
		"gotools # goimports",
	}
	if cfg.GolangciLint.CreateConfig {
		packages = append(packages, "golangci-lint")
	}
	if renderGoreleaserConfig {
		packages = append(packages, "goreleaser")
	}
	runControllerGen := sr.KubernetesController
	if cfg.ControllerGen.Enabled != nil {
		runControllerGen = *cfg.ControllerGen.Enabled
	}
	if runControllerGen {
		packages = append(packages, "kubernetes-controller-tools # controller-gen")
		packages = append(packages, "setup-envtest")
	}
	if sr.UseGinkgo {
		packages = append(packages, "ginkgo")
	}
	if sr.UsesPostgres {
		packages = append(packages, "postgresql_"+core.DefaultPostgresVersion)
	}
	packages = append(packages, cfg.Nix.ExtraPackages...)

	slices.Sort(packages)

	t := template.Must(template.New("shell.nix").Parse(shellNixTemplate))
	var buf bytes.Buffer
	must.Succeed(t.Execute(&buf, map[string]any{"Packages": packages, "ExtraLibraries": cfg.Nix.ExtraLibraries}))
	must.Succeed(os.WriteFile("shell.nix", buf.Bytes(), 0666))

	must.Succeed(os.WriteFile(".envrc", envrcTemplate, 0666))
}
