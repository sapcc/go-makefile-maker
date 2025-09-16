// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package nix

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

var (
	//go:embed shell.nix.tmpl
	shellNixTemplate string
	//go:embed envrc.tmpl
	envrcTemplate string
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
		// syft is used by goreleaser to generate an SBOM
		packages = append(packages, "goreleaser", "syft")
	}
	runControllerGen := cfg.ControllerGen.Enabled.UnwrapOr(sr.KubernetesController)
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
	if cfg.Reuse.Enabled.UnwrapOr(true) {
		packages = append(packages, "reuse")
	}
	packages = append(packages, cfg.Nix.ExtraPackages...)

	slices.Sort(packages)

	must.Succeed(util.WriteFileFromTemplate("shell.nix", shellNixTemplate, map[string]any{
		"Packages":       packages,
		"ExtraLibraries": cfg.Nix.ExtraLibraries,
	}))
	must.Succeed(util.WriteFileFromTemplate(".envrc", envrcTemplate, map[string]any{
		"Variables": cfg.VariableValues,
	}))
}
