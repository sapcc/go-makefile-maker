// Copyright 2024 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

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
