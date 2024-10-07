package nix

import (
	"fmt"
	"os"

	"github.com/sapcc/go-bits/must"
)

func RenderShell() {
	nixShellTemplate := `{ pkgs ? import <nixpkgs> { } }:

with pkgs;

mkShell {
  nativeBuildInputs = [
    go_1_23

    addlicense
    ginkgo # conditional
    go-licence-detector
    golangci-lint
    goreleaser
    gotools # goimports
    kubernetes-controller-tools # controller-gen
    postgresql_17
    setup-envtest # k8s controller

    openssl # conditional, list of options?

    # keep this line if you use bash
    bashInteractive
  ];
}
`

  packageList := ""
  for _, package := range packages {
    packageList += "    " + package
  }

	nixShellFile := fmt.Sprintf(nixShellTemplate, packageList)
	must.Succeed(os.WriteFile(".goreleaser.yaml", []byte(nixShellFile), 0666))

	must.Succeed(os.WriteFile(".goreleaser.yaml", []byte(`#!/usr/bin/env bash
if type -P lorri &>/dev/null; then
  eval "$(lorri direnv)"
else
  use nix
fi
`), 0666))
}
