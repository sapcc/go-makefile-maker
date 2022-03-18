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

package makefile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

//Renderer is used to render the Makefile.
type Renderer struct {
	out          io.Writer
	currentBlock string
}

func New(w io.Writer) (*Renderer, error) {
	f, err := os.Create("Makefile")
	if err != nil {
		return nil, err
	}
	return &Renderer{out: f}, nil
}

//Render renders the Makefile (except for the part in `cfg.Verbatim`).
func (r *Renderer) Render(cfg *core.Configuration) {
	fmt.Fprintln(r.out, core.AutogeneratedHeader)

	r.addDefinition("MAKEFLAGS=--warn-undefined-variables")
	fmt.Fprintln(r.out)

	r.addDefinition("# /bin/sh is dash on Debian which does not support all features of ash/bash")
	r.addDefinition("# to fix that we use /bin/bash only on Debian to not break Alpine")
	r.addDefinition("ifneq (,$(wildcard /etc/os-release)) # check file existence")
	r.addDefinition("\tifneq ($(shell grep -c debian /etc/os-release),0)")
	r.addDefinition("\t\tSHELL := /bin/bash")
	r.addDefinition("\tendif")
	r.addDefinition("endif")

	r.currentBlock = "definition"
	r.addRule("default: build-all")

	if cfg.Verbatim != "" {
		fmt.Fprintln(r.out)
		r.currentBlock = "none"
		fmt.Fprintf(r.out, "%s", FixRuleIndentation(cfg.Verbatim))
		r.currentBlock = "rule"
	}

	r.addBuildAllTarget(cfg.Binaries)

	//add definitions for common variables
	defaultBuildFlags := ""
	if cfg.Vendoring.Enabled {
		defaultBuildFlags = "-mod vendor"
	}
	r.addDefinition("GO_BUILDFLAGS =%s", cfg.Variable("GO_BUILDFLAGS", defaultBuildFlags))
	r.addDefinition("GO_LDFLAGS =%s", cfg.Variable("GO_LDFLAGS", ""))
	r.addDefinition("GO_TESTENV =%s", cfg.Variable("GO_TESTENV", ""))

	//add build targets for each binary
	for _, bin := range cfg.Binaries {
		r.addRule("build/%s: FORCE", bin.Name)
		r.addRecipe("go build $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -o build/%s %s", bin.Name, bin.FromPackage)
	}

	r.addInstallTargetIfDesired(cfg.Binaries)

	//add definitions for testing targets
	r.addDefinition(`# which packages to test with "go test"`)
	r.addDefinition(`GO_TESTPKGS := $(shell go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...)`)
	r.addDefinition(`# which packages to measure coverage for`)
	coverPkgGreps := ""
	if cfg.Coverage.Only != "" {
		coverPkgGreps += fmt.Sprintf(" | command grep -E '%s'", cfg.Coverage.Only)
	}
	if cfg.Coverage.Except != "" {
		coverPkgGreps += fmt.Sprintf(" | command grep -Ev '%s'", cfg.Coverage.Except)
	}
	r.addDefinition(`GO_COVERPKGS := $(shell go list ./...%s)`, coverPkgGreps)
	r.addDefinition(`# to get around weird Makefile syntax restrictions, we need variables containing a space and comma`)
	r.addDefinition(`space := $(null) $(null)`)
	r.addDefinition(`comma := ,`)

	//add main testing target
	r.addRule("check: build-all static-check build/cover.html FORCE")
	r.addRecipe(`@printf "\e[1;32m>> All checks successful.\e[0m\n"`)

	//add target for installing dependencies for `make check`
	r.addRule("prepare-static-check: FORCE")
	r.addRecipe(`@if ! hash golangci-lint 2>/dev/null; then printf "\e[1;36m>> Installing golangci-lint (this may take a while)...\e[0m\n"; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; fi`)

	//add target for static code checks
	r.addRule("static-check: prepare-static-check FORCE")
	r.addRecipe(`@printf "\e[1;36m>> golangci-lint\e[0m\n"`)
	r.addRecipe(`@golangci-lint run`)

	//add targets for `go test` incl. coverage report
	r.addRule(`build/cover.out: build FORCE`)
	r.addRecipe(`@printf "\e[1;36m>> go test\e[0m\n"`)
	r.addRecipe(`@env $(GO_TESTENV) go test $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -shuffle=on -p 1 -coverprofile=$@ -covermode=count -coverpkg=$(subst $(space),$(comma),$(GO_COVERPKGS)) $(GO_TESTPKGS)`)
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
		r.addRule("vendor-compat: FORCE")
		r.addRecipe(`go mod tidy -compat=$(shell awk '$$1 == "go" { print $$2 }' < go.mod)`)
		r.addRecipe("go mod vendor")
		r.addRecipe("go mod verify")
	} else {
		r.addRule("tidy-deps: FORCE")
		r.addRecipe("go mod tidy")
		r.addRecipe("go mod verify")
	}

	r.addRule(`license-headers: FORCE`)
	r.addRecipe(`@if ! hash addlicense 2>/dev/null; then printf "\e[1;36m>> Installing addlicense...\e[0m\n"; go install github.com/google/addlicense@latest; fi`)
	r.addRecipe(`find * \( -name vendor -type d -prune \) -o \( -name \*.go -exec addlicense -c "SAP SE" -- {} + \)`)

	//add cleaning target
	r.addRule("clean: FORCE")
	r.addRecipe("git clean -dxf build")

	r.addRule(".PHONY: FORCE")
}

func (r *Renderer) addBuildAllTarget(binaries []core.BinaryConfiguration) {
	rule := "build-all:"
	for _, bin := range binaries {
		rule += fmt.Sprintf(" build/%s", bin.Name)
	}
	r.addRule(rule)
}

func (r *Renderer) addInstallTargetIfDesired(binaries []core.BinaryConfiguration) {
	installTargetDeps := ""
	for _, bin := range binaries {
		if bin.InstallTo != "" {
			installTargetDeps += fmt.Sprintf(" build/%s", bin.Name)
		}
	}
	if installTargetDeps == "" {
		return
	}

	r.addDefinition("DESTDIR =")
	r.addDefinition("ifeq ($(shell uname -s),Darwin)")
	r.addDefinition("\tPREFIX = /usr/local")
	r.addDefinition("else")
	r.addDefinition("\tPREFIX = /usr")
	r.addDefinition("endif")
	r.addRule("install: FORCE%s", installTargetDeps)

	for _, bin := range binaries {
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
		fmt.Fprintln(r.out)
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
		fmt.Fprintln(r.out)
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