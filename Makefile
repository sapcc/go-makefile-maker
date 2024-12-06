################################################################################
# This file is AUTOGENERATED with <https://github.com/sapcc/go-makefile-maker> #
# Edit Makefile.maker.yaml instead.                                            #
################################################################################

# Copyright 2024 SAP SE
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0

MAKEFLAGS=--warn-undefined-variables
# /bin/sh is dash on Debian which does not support all features of ash/bash
# to fix that we use /bin/bash only on Debian to not break Alpine
ifneq (,$(wildcard /etc/os-release)) # check file existence
	ifneq ($(shell grep -c debian /etc/os-release),0)
		SHELL := /bin/bash
	endif
endif

default: build-all

run: build/go-makefile-maker
	./build/go-makefile-maker

install-golangci-lint: FORCE
	@if ! hash golangci-lint 2>/dev/null; then printf "\e[1;36m>> Installing golangci-lint (this may take a while)...\e[0m\n"; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; fi

install-go-licence-detector: FORCE
	@if ! hash go-licence-detector 2>/dev/null; then printf "\e[1;36m>> Installing go-licence-detector...\e[0m\n"; go install go.elastic.co/go-licence-detector@latest; fi

install-addlicense: FORCE
	@if ! hash addlicense 2>/dev/null; then  printf "\e[1;36m>> Installing addlicense...\e[0m\n";  go install github.com/google/addlicense@latest; fi

prepare-static-check: FORCE install-golangci-lint install-go-licence-detector install-addlicense

GO_BUILDFLAGS = -mod vendor
GO_LDFLAGS =
GO_TESTENV =
GO_BUILDENV =

build-all: build/go-makefile-maker

build/go-makefile-maker: FORCE
	@env $(GO_BUILDENV) go build $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -o build/go-makefile-maker .

DESTDIR =
ifeq ($(shell uname -s),Darwin)
	PREFIX = /usr/local
else
	PREFIX = /usr
endif

install: FORCE build/go-makefile-maker
	install -d -m 0755 "$(DESTDIR)$(PREFIX)/bin"
	install -m 0755 build/go-makefile-maker "$(DESTDIR)$(PREFIX)/bin/go-makefile-maker"

# which packages to test with test runner
GO_TESTPKGS := $(shell go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...)
ifeq ($(GO_TESTPKGS),)
GO_TESTPKGS := ./...
endif
# which packages to measure coverage for
GO_COVERPKGS := $(shell go list ./...)
# to get around weird Makefile syntax restrictions, we need variables containing nothing, a space and comma
null :=
space := $(null) $(null)
comma := ,

check: FORCE static-check build/cover.html build-all
	@printf "\e[1;32m>> All checks successful.\e[0m\n"

run-golangci-lint: FORCE install-golangci-lint
	@printf "\e[1;36m>> golangci-lint\e[0m\n"
	@golangci-lint run

build/cover.out: FORCE | build
	@printf "\e[1;36m>> Running tests\e[0m\n"
	@env $(GO_TESTENV) go test -shuffle=on -p 1 -coverprofile=$@ $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -covermode=count -coverpkg=$(subst $(space),$(comma),$(GO_COVERPKGS)) $(GO_TESTPKGS)

build/cover.html: build/cover.out
	@printf "\e[1;36m>> go tool cover > build/cover.html\e[0m\n"
	@go tool cover -html $< -o $@

static-check: FORCE run-golangci-lint check-dependency-licenses check-license-headers

build:
	@mkdir $@

vendor: FORCE
	go mod tidy
	go mod vendor
	go mod verify

vendor-compat: FORCE
	go mod tidy -compat=$(shell awk '$$1 == "go" { print $$2 }' < go.mod)
	go mod vendor
	go mod verify

force-license-headers: FORCE install-addlicense
	@printf "\e[1;36m>> addlicense\e[0m\n"
	echo -n $(patsubst $(shell awk '$$1 == "module" {print $$2}' go.mod)%,.%/*.go,$(shell go list ./...)) | xargs -d" " -I{} bash -c 'year="$$(rg -P "Copyright (....) SAP SE" -Nor "\$$1" {})"; awk -i inplace '"'"'{if (display) {print} else {!/^\/\*/ && !/^\*/ && !/^\$$/}}; /^package /{print;display=1}'"'"' {}; addlicense -c "SAP SE" -s=only -y "$$year" -- {}'

license-headers: FORCE install-addlicense
	@printf "\e[1;36m>> addlicense\e[0m\n"
	@addlicense -c "SAP SE" -s -- $(patsubst $(shell awk '$$1 == "module" {print $$2}' go.mod)%,.%/*.go,$(shell go list ./...))

check-license-headers: FORCE install-addlicense
	@printf "\e[1;36m>> addlicense --check\e[0m\n"
	@addlicense --check -- $(patsubst $(shell awk '$$1 == "module" {print $$2}' go.mod)%,.%/*.go,$(shell go list ./...))

check-dependency-licenses: FORCE install-go-licence-detector
	@printf "\e[1;36m>> go-licence-detector\e[0m\n"
	@go list -m -mod=readonly -json all | go-licence-detector -includeIndirect -rules .license-scan-rules.json -overrides .license-scan-overrides.jsonl

clean: FORCE
	git clean -dxf build

vars: FORCE
	@printf "DESTDIR=$(DESTDIR)\n"
	@printf "GO_BUILDENV=$(GO_BUILDENV)\n"
	@printf "GO_BUILDFLAGS=$(GO_BUILDFLAGS)\n"
	@printf "GO_COVERPKGS=$(GO_COVERPKGS)\n"
	@printf "GO_LDFLAGS=$(GO_LDFLAGS)\n"
	@printf "GO_TESTENV=$(GO_TESTENV)\n"
	@printf "GO_TESTPKGS=$(GO_TESTPKGS)\n"
	@printf "PREFIX=$(PREFIX)\n"
help: FORCE
	@printf "\n"
	@printf "\e[1mUsage:\e[0m\n"
	@printf "  make \e[36m<target>\e[0m\n"
	@printf "\n"
	@printf "\e[1mGeneral\e[0m\n"
	@printf "  \e[36mvars\e[0m                         Display values of relevant Makefile variables.\n"
	@printf "  \e[36mhelp\e[0m                         Display this help.\n"
	@printf "\n"
	@printf "\e[1mPrepare\e[0m\n"
	@printf "  \e[36minstall-golangci-lint\e[0m        Install golangci-lint required by run-golangci-lint/static-check\n"
	@printf "  \e[36minstall-go-licence-detector\e[0m  Install-go-licence-detector required by check-dependency-licenses/static-check\n"
	@printf "  \e[36minstall-addlicense\e[0m           Install addlicense required by check-license-headers/license-headers/static-check\n"
	@printf "  \e[36mprepare-static-check\e[0m         Install any tools required by static-check. This is used in CI before dropping privileges, you should probably install all the tools using your package manager\n"
	@printf "\n"
	@printf "\e[1mBuild\e[0m\n"
	@printf "  \e[36mbuild-all\e[0m                    Build all binaries.\n"
	@printf "  \e[36mbuild/go-makefile-maker\e[0m      Build go-makefile-maker.\n"
	@printf "  \e[36minstall\e[0m                      Install all binaries. This option understands the conventional 'DESTDIR' and 'PREFIX' environment variables for choosing install locations.\n"
	@printf "\n"
	@printf "\e[1mTest\e[0m\n"
	@printf "  \e[36mcheck\e[0m                        Run the test suite (unit tests and golangci-lint).\n"
	@printf "  \e[36mrun-golangci-lint\e[0m            Install and run golangci-lint. Installing is used in CI, but you should probably install golangci-lint using your package manager.\n"
	@printf "  \e[36mbuild/cover.out\e[0m              Run tests and generate coverage report.\n"
	@printf "  \e[36mbuild/cover.html\e[0m             Generate an HTML file with source code annotations from the coverage report.\n"
	@printf "  \e[36mstatic-check\e[0m                 Run static code checks\n"
	@printf "\n"
	@printf "\e[1mDevelopment\e[0m\n"
	@printf "  \e[36mvendor\e[0m                       Run go mod tidy, go mod verify, and go mod vendor.\n"
	@printf "  \e[36mvendor-compat\e[0m                Same as 'make vendor' but go mod tidy will use '-compat' flag with the Go version from go.mod file as value.\n"
	@printf "  \e[36mforce-license-headers\e[0m        Remove and re-add all license headers to all non-vendored source code files.\n"
	@printf "  \e[36mlicense-headers\e[0m              Add license headers to all non-vendored source code files.\n"
	@printf "  \e[36mcheck-license-headers\e[0m        Check license headers in all non-vendored .go files.\n"
	@printf "  \e[36mcheck-dependency-licenses\e[0m    Check all dependency licenses using go-licence-detector.\n"
	@printf "  \e[36mclean\e[0m                        Run git clean.\n"

.PHONY: FORCE
