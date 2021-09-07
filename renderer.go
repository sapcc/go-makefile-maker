/******************************************************************************
*
*  Copyright 2020 SAP SE
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
******************************************************************************/

package main

import (
	"fmt"
	"io"
	"path/filepath"
)

//Renderer is used to render the Makefile.
type Renderer struct {
	out          io.Writer
	currentBlock string
}

//Render renders the Makefile (except for the part in `cfg.Verbatim`).
func (r *Renderer) Render(cfg Configuration) {
	fmt.Fprintln(r.out, "################################################################################")
	fmt.Fprintln(r.out, "# This file is AUTOGENERATED with <https://github.com/sapcc/go-makefile-maker> #")
	fmt.Fprintln(r.out, "# Edit Makefile.maker.yaml instead.                                            #")
	fmt.Fprintln(r.out, "################################################################################")

	r.addDefinition("MAKEFLAGS=--warn-undefined-variables")
	r.addDefinition("SHELL := /bin/bash")

	r.currentBlock = "definition"
	r.addRule("default: build-all")

	if cfg.Verbatim != "" {
		r.out.Write([]byte("\n"))
		r.currentBlock = "none"
		r.out.Write([]byte(FixRuleIndentation(cfg.Verbatim)))
		r.currentBlock = "rule"
	}

	r.addBuildAllTarget(cfg)

	//add definitions for common variables
	defaultBuildFlags := ""
	if cfg.Vendoring.Enabled {
		defaultBuildFlags = "-mod vendor"
	}
	r.addDefinition("GO_BUILDFLAGS = %s", cfg.Variable("GO_BUILDFLAGS", defaultBuildFlags))
	r.addDefinition("GO_LDFLAGS = %s", cfg.Variable("GO_LDFLAGS", ""))
	r.addDefinition("GO_TESTENV = %s", cfg.Variable("GO_TESTENV", ""))

	//add build targets for each binary
	for _, bin := range cfg.Binaries {
		r.addRule("build/%s: FORCE", bin.Name)
		r.addRecipe("go build $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -o build/%s %s", bin.Name, bin.FromPackage)
	}

	r.addInstallTargetIfDesired(cfg)

	//add definitions for testing targets
	if !cfg.StaticCheck.GolangciLint {
		r.addDefinition(`# which packages to test with static checkers`)
		r.addDefinition(`GO_ALLPKGS := $(shell go list ./...)`)
		r.addDefinition(`# which files to test with static checkers (this contains a list of globs)`)
		r.addDefinition(`GO_ALLFILES := $(addsuffix /*.go,$(patsubst $(shell go list .)%,.%,$(shell go list ./...)))`)
	}
	r.addDefinition(`# which packages to test with "go test"`)
	r.addDefinition(`GO_TESTPKGS := $(shell go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./...)`)
	r.addDefinition(`# which packages to measure coverage for`)
	coverPkgGreps := ""
	if cfg.Coverage.Only != "" {
		coverPkgGreps += fmt.Sprintf(" | grep -E '%s'", cfg.Coverage.Only)
	}
	if cfg.Coverage.Except != "" {
		coverPkgGreps += fmt.Sprintf(" | grep -Ev '%s'", cfg.Coverage.Except)
	}
	r.addDefinition(`GO_COVERPKGS := $(shell go list ./...%s)`, coverPkgGreps)
	r.addDefinition(`# to get around weird Makefile syntax restrictions, we need variables containing a space and comma`)
	r.addDefinition(`space := $(null) $(null)`)
	r.addDefinition(`comma := ,`)

	//add main testing target
	r.addRule("check: build-all static-check build/cover.html FORCE")
	r.addRecipe(`@printf "\e[1;32m>> All checks successful.\e[0m\n"`)

	//add target for static code checks
	r.addRule("static-check: FORCE")
	if cfg.StaticCheck.GolangciLint {
		r.addRecipe(`@command -v golangci-lint >/dev/null 2>&1 || { echo >&2 "Error: golangci-lint is not installed. See: https://golangci-lint.run/usage/install/"; exit 1; }`)
		r.addRecipe(`@printf "\e[1;36m>> golangci-lint\e[0m\n"`)
		r.addRecipe(`@golangci-lint run`)
	} else {
		r.addRecipe(`@if ! hash golint 2>/dev/null; then printf "\e[1;36m>> Installing golint...\e[0m\n"; GO111MODULE=off go get -u golang.org/x/lint/golint; fi`)
		r.addRecipe(`@printf "\e[1;36m>> gofmt\e[0m\n"`)
		r.addRecipe(`@if s="$$(gofmt -s -d $(GO_ALLFILES) 2>/dev/null)" && test -n "$$s"; then echo "$$s"; false; fi`)
		r.addRecipe(`@printf "\e[1;36m>> golint\e[0m\n"`)
		r.addRecipe(`@if s="$$(golint $(GO_ALLPKGS) 2>/dev/null)" && test -n "$$s"; then echo "$$s"; false; fi`)
		r.addRecipe(`@printf "\e[1;36m>> go vet\e[0m\n"`)
		r.addRecipe(`@go vet $(GO_BUILDFLAGS) $(GO_ALLPKGS)`)
	}

	//add targets for `go test` incl. coverage report
	r.addRule(`build/cover.out: build FORCE`)
	r.addRecipe(`@printf "\e[1;36m>> go test\e[0m\n"`)
	r.addRecipe(`@env $(GO_TESTENV) go test $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -p 1 -coverprofile=$@ -covermode=count -coverpkg=$(subst $(space),$(comma),$(GO_COVERPKGS)) $(GO_TESTPKGS)`)
	r.addRule(`build/cover.html: build/cover.out`)
	r.addRecipe(`@printf "\e[1;36m>> go tool cover > build/cover.html\e[0m\n"`)
	r.addRecipe(`@go tool cover -html $< -o $@`)

	//ensure that build directory exists
	r.addRule(`build:`)
	r.addRecipe(`@mkdir $@`)

	//add tidy-deps or vendor target
	if cfg.Vendoring.Enabled {
		r.addRule("vendor: FORCE")
		r.addRecipe("go mod tidy")
		r.addRecipe("go mod vendor")
		r.addRecipe("go mod verify")
	} else {
		r.addRule("tidy-deps: FORCE")
		r.addRecipe("go mod tidy")
		r.addRecipe("go mod verify")
	}

	r.addRule(`license-headers: FORCE`)
	r.addRecipe(`@if ! hash addlicense 2>/dev/null; then printf "\e[1;36m>> Installing addlicense...\e[0m\n"; GO111MODULE=off go get -u github.com/google/addlicense; fi`)
	r.addRecipe(`find * \( -name vendor -type d -prune \) -o \( -name \*.go -exec addlicense -c "SAP SE" -- {} + \)`)

	//add cleaning target
	r.addRule("clean: FORCE")
	r.addRecipe("git clean -dxf build")

	r.addRule(".PHONY: FORCE")
}

func (r *Renderer) addBuildAllTarget(cfg Configuration) {
	rule := "build-all:"
	for _, bin := range cfg.Binaries {
		rule += fmt.Sprintf(" build/%s", bin.Name)
	}
	r.addRule(rule)
}

func (r *Renderer) addInstallTargetIfDesired(cfg Configuration) {
	installTargetDeps := ""
	for _, bin := range cfg.Binaries {
		if bin.InstallTo != "" {
			installTargetDeps += fmt.Sprintf(" build/%s", bin.Name)
		}
	}
	if installTargetDeps == "" {
		return
	}

	r.addDefinition("DESTDIR =")
	r.addDefinition("ifeq ($(shell uname -s),Darwin)")
	r.addDefinition("  PREFIX = /usr/local")
	r.addDefinition("else")
	r.addDefinition("  PREFIX = /usr")
	r.addDefinition("endif")
	r.addRule("install: FORCE%s", installTargetDeps)

	for _, bin := range cfg.Binaries {
		if bin.InstallTo != "" {
			r.addRecipe(`install -D -m 0755 build/%s "$(DESTDIR)$(PREFIX)/%s/%s"`, bin.Name, filepath.Clean(bin.InstallTo), bin.Name)
		}
	}
}

func (r *Renderer) addDefinition(def string, args ...interface{}) {
	if len(args) > 0 {
		def = fmt.Sprintf(def, args...)
	}
	if r.currentBlock != "definition" {
		//put an empty line between rules
		r.out.Write([]byte("\n"))
	}
	fmt.Fprintf(r.out, "%s\n", def)
	r.currentBlock = "definition"
}

func (r *Renderer) addRule(rule string, args ...interface{}) {
	if len(args) > 0 {
		rule = fmt.Sprintf(rule, args...)
	}
	if r.currentBlock != "none" {
		//put an empty line between rules
		r.out.Write([]byte("\n"))
	}
	fmt.Fprintf(r.out, "%s\n", rule)
	r.currentBlock = "rule"
}

func (r *Renderer) addRecipe(recipe string, args ...interface{}) {
	if len(args) > 0 {
		recipe = fmt.Sprintf(recipe, args...)
	}
	fmt.Fprintf(r.out, "\t%s\n", recipe)
}
