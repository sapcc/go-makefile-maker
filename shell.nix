# SPDX-License-Identifier: Apache-2.0

{ pkgs ? import <nixpkgs> { } }:

with pkgs;

mkShell {
  nativeBuildInputs = [
    go-licence-detector
    go_1_23
    golangci-lint
    gotools # goimports

    # keep this line if you use bash
    bashInteractive
  ];
}
