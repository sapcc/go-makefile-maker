package nix

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/sapcc/go-bits/must"
	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

func RenderShell(cfg core.Configuration, sr golang.ScanResult, renderGoreleaserConfig bool) {
	nixShellTemplate := `{ pkgs ? import <nixpkgs> { } }:

with pkgs;

mkShell {
  nativeBuildInputs = [
%s
    # keep this line if you use bash
    bashInteractive
  ];
}
`

	goVersionSlice := strings.Split(core.DefaultGoVersion, ".")
	goPackage := fmt.Sprintf("go_%s_%s", goVersionSlice[0], goVersionSlice[1])
	packages := []string{
		goPackage,
		"go-licence-detector",
		"gotools # goimports",
	}
	if cfg.GolangciLint.CreateConfig {
		packages = append(packages, "golangci-lint")
	}
	if renderGoreleaserConfig {
		packages = append(packages, "goreleaser")
	}
	if sr.KubernetesController {
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
	packageList := ""
	for _, pkg := range packages {
		packageList += fmt.Sprintf("    %s\n", pkg)
	}

	nixShellFile := fmt.Sprintf(nixShellTemplate, packageList)
	must.Succeed(os.WriteFile("shell.nix", []byte(nixShellFile), 0666))

	must.Succeed(os.WriteFile(".envrc", []byte(`#!/usr/bin/env bash
if type -P lorri &>/dev/null; then
  eval "$(lorri direnv)"
else
  use nix
fi
`), 0666))
}
