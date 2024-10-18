/*******************************************************************************
*
* Copyright 2024 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/

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

let
  # TODO: drop after https://github.com/NixOS/nixpkgs/pull/347304 got merged
  go-licence-detector = buildGoModule rec {
    pname = "go-licence-detector";
    version = "0.7.0";

    src = fetchFromGitHub {
      owner = "elastic";
      repo = "go-licence-detector";
      rev = "v${version}";
      hash = "sha256-43MyzEF7BZ7pcgzDvXx9SjXGHaLozmWkGWUO/yf6K98=";
    };

    vendorHash = "sha256-7vIP5pGFH6CbW/cJp+DiRg2jFcLFEBl8dQzUw1ogTTA=";

    meta = with lib; {
      description = "Detect licences in Go projects and generate documentation";
      homepage = "https://github.com/elastic/go-licence-detector";
      license = licenses.asl20;
      maintainers = with maintainers; [ SuperSandro2000 ];
    };
  };%s
in

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
	overlay := ""
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
		overlay += `

  # TODO: drop after https://github.com/NixOS/nixpkgs/pull/345260 got merged
  postgresql_17 = (import (pkgs.path + /pkgs/servers/sql/postgresql/generic.nix) {
    version = "17.0";
    hash = "sha256-fidhMcD91rYliNutmzuyS4w0mNUAkyjbpZrxboGRCd4=";
  } { self = pkgs; jitSupport = false; }).overrideAttrs ({ nativeBuildInputs, configureFlags , ... }: {
    nativeBuildInputs = nativeBuildInputs ++ (with pkgs; [ bison flex perl docbook_xml_dtd_45 docbook-xsl-nons libxslt ]);
    configureFlags = configureFlags ++ [ "--without-perl" ];
  });`
	}
	packages = append(packages, cfg.Nix.ExtraPackages...)

	slices.Sort(packages)
	packageList := ""
	for _, pkg := range packages {
		packageList += fmt.Sprintf("    %s\n", pkg)
	}

	nixShellFile := fmt.Sprintf(nixShellTemplate, overlay, packageList)
	must.Succeed(os.WriteFile("shell.nix", []byte(nixShellFile), 0666))

	must.Succeed(os.WriteFile(".envrc", []byte(`#!/usr/bin/env bash
if type -P lorri &>/dev/null; then
  eval "$(lorri direnv)"
else
  use nix
fi
`), 0666))
}
