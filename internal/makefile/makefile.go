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
	"path/filepath"
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

// newMakefile defines the structure of the Makefile. Order is important as categories,
// rules, and definitions will appear in the exact order as they are defined.
func newMakefile(cfg *core.Configuration) *makefile {
	hasBinaries := len(cfg.Binaries) > 0

	///////////////////////////////////////////////////////////////////////////
	// General
	general := category{name: "general"}

	general.addDefinition("MAKEFLAGS=--warn-undefined-variables")
	general.addDefinition(strings.TrimSpace(`
# /bin/sh is dash on Debian which does not support all features of ash/bash
# to fix that we use /bin/bash only on Debian to not break Alpine
ifneq (,$(wildcard /etc/os-release)) # check file existence
	ifneq ($(shell grep -c debian /etc/os-release),0)
		SHELL := /bin/bash
	endif
endif
	`))

	if hasBinaries {
		general.addRule(rule{
			target:        "default",
			prerequisites: []string{"build-all"},
		})
	} else {
		general.addRule(rule{
			target: "default",
			phony:  true,
			recipe: []string{"@echo 'There is nothing to build, use `make check` for running the test suite or `make help` for a list of available targets.'"},
		})
	}

	///////////////////////////////////////////////////////////////////////////
	// Build
	build := category{name: "build"}

	defaultBuildFlags := ""
	if cfg.Vendoring.Enabled {
		defaultBuildFlags = "-mod vendor"
	}
	build.addDefinition("GO_BUILDFLAGS =%s", cfg.Variable("GO_BUILDFLAGS", defaultBuildFlags))
	build.addDefinition("GO_LDFLAGS =%s", cfg.Variable("GO_LDFLAGS", ""))
	build.addDefinition("GO_TESTENV =%s", cfg.Variable("GO_TESTENV", ""))

	if hasBinaries {
		build.addRule(buildTargets(cfg.Binaries)...)
		if r, ok := installTarget(cfg.Binaries); ok {
			build.addRule(r)
		}
	}

	///////////////////////////////////////////////////////////////////////////
	// Test
	test := category{name: "test"}

	test.addDefinition(`# which packages to test with "go test"`)
	testPkgGreps := ""
	if cfg.Test.Only != "" {
		testPkgGreps += fmt.Sprintf(" | command grep -E '%s'", cfg.Test.Only)
	}
	if cfg.Test.Except != "" {
		testPkgGreps += fmt.Sprintf(" | command grep -Ev '%s'", cfg.Test.Except)
	}
	test.addDefinition(`GO_TESTPKGS := $(shell go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...%s)`, testPkgGreps)

	test.addDefinition(`# which packages to measure coverage for`)
	coverPkgGreps := ""
	if cfg.Coverage.Only != "" {
		coverPkgGreps += fmt.Sprintf(" | command grep -E '%s'", cfg.Coverage.Only)
	}
	if cfg.Coverage.Except != "" {
		coverPkgGreps += fmt.Sprintf(" | command grep -Ev '%s'", cfg.Coverage.Except)
	}
	test.addDefinition(`GO_COVERPKGS := $(shell go list ./...%s)`, coverPkgGreps)
	test.addDefinition(`# to get around weird Makefile syntax restrictions, we need variables containing a space and comma`)
	test.addDefinition(`space := $(null) $(null)`)
	test.addDefinition(`comma := ,`)

	//add main testing target
	var checkPrerequisites []string
	if hasBinaries {
		checkPrerequisites = []string{"build-all", "static-check", "build/cover.html"}
	} else {
		checkPrerequisites = []string{"static-check", "build/cover.html"}
	}
	test.addRule(rule{
		description:   "Run the test suite (unit tests and golangci-lint).",
		phony:         true,
		target:        "check",
		prerequisites: checkPrerequisites,
		recipe:        []string{`@printf "\e[1;32m>> All checks successful.\e[0m\n"`},
	})

	//add target for installing dependencies for `make check`
	test.addRule(rule{
		description: "Install golangci-lint. This is used in CI, you should probably install golangci-lint using your package manager.",
		phony:       true,
		target:      "prepare-static-check",
		recipe: []string{
			`@if ! hash golangci-lint 2>/dev/null;` +
				` then printf "\e[1;36m>> Installing golangci-lint (this may take a while)...\e[0m\n";` +
				` go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; fi`,
		},
	})

	//add target for static code checks
	test.addRule(rule{
		description:   "Run golangci-lint.",
		phony:         true,
		target:        "static-check",
		prerequisites: []string{"prepare-static-check"},
		recipe: []string{
			`@printf "\e[1;36m>> golangci-lint\e[0m\n"`,
			`@golangci-lint run`,
		},
	})

	//add targets for `go test` incl. coverage report
	test.addRule(rule{
		description: "Run tests and generate coverage report.",
		phony:       true,
		target:      "build/cover.out",
		// We use order only prerequisite because this target is used in CI.
		orderOnlyPrerequisites: []string{"build"},
		recipe: []string{
			`@printf "\e[1;36m>> go test\e[0m\n"`,
			`@env $(GO_TESTENV) go test $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -shuffle=on -p 1 -coverprofile=$@ -covermode=count -coverpkg=$(subst $(space),$(comma),$(GO_COVERPKGS)) $(GO_TESTPKGS)`,
		},
	})

	test.addRule(rule{
		description:   "Generate an HTML file with source code annotations from the coverage report.",
		target:        "build/cover.html",
		prerequisites: []string{"build/cover.out"},
		recipe: []string{
			`@printf "\e[1;36m>> go tool cover > build/cover.html\e[0m\n"`,
			`@go tool cover -html $< -o $@`,
		},
	})

	///////////////////////////////////////////////////////////////////////////
	// Development
	dev := category{name: "development"}

	//ensure that build directory exists
	dev.addRule(rule{
		target: "build",
		recipe: []string{`@mkdir $@`},
	})

	//add tidy-deps or vendor target
	if cfg.Vendoring.Enabled {
		dev.addRule(rule{
			description: "Run go mod tidy, go mod verify, and go mod vendor.",
			target:      "vendor",
			phony:       true,
			recipe: []string{
				"go mod tidy",
				"go mod vendor",
				"go mod verify",
			},
		})
		dev.addRule(rule{
			description: "Same as 'make vendor' but go mod tidy will use '-compat' flag with the Go version from go.mod file as value.",
			target:      "vendor-compat",
			phony:       true,
			recipe: []string{
				`go mod tidy -compat=$(shell awk '$$1 == "go" { print $$2 }' < go.mod)`,
				"go mod vendor",
				"go mod verify",
			},
		})
	} else {
		dev.addRule(rule{
			description: "Run go mod tidy and go mod verify.",
			target:      "tidy-deps",
			phony:       true,
			recipe: []string{
				"go mod tidy",
				"go mod verify",
			},
		})
	}

	dev.addRule(rule{
		description: "Add license headers to all .go files excluding the vendor directory.",
		target:      "license-headers",
		phony:       true,
		recipe: []string{
			`@if ! hash addlicense 2>/dev/null; then printf "\e[1;36m>> Installing addlicense...\e[0m\n"; go install github.com/google/addlicense@latest; fi`,
			`find * \( -name vendor -type d -prune \) -o \( -name \*.go -exec addlicense -c "SAP SE" -- {} + \)`,
		},
	})

	//add cleaning target
	dev.addRule(rule{
		description: "Run git clean.",
		target:      "clean",
		phony:       true,
		recipe:      []string{"git clean -dxf build"},
	})

	return &makefile{
		categories: []category{
			general,
			build,
			test,
			dev,
		},
	}
}

func buildTargets(binaries []core.BinaryConfiguration) []rule {
	result := make([]rule, 0, len(binaries)+1)
	bAllRule := rule{
		description: "Build all binaries.",
		target:      "build-all",
	}
	result = append(result, bAllRule)

	allPrerequisites := make([]string, 0, len(binaries))
	for _, bin := range binaries {
		r := rule{
			description: fmt.Sprintf("Build %s.", bin.Name),
			phony:       true,
			target:      fmt.Sprintf("build/%s", bin.Name),
			recipe: []string{fmt.Sprintf(
				"go build $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -o build/%s %s",
				bin.Name, bin.FromPackage,
			)},
		}

		result = append(result, r)
		allPrerequisites = append(allPrerequisites, r.target)
	}
	result[0].prerequisites = allPrerequisites

	return result
}

// installTarget also returns a bool that tells whether the install target was requested
// in the config.
func installTarget(binaries []core.BinaryConfiguration) (rule, bool) {
	r := rule{
		description: "Install all binaries. " +
			"This option understands the conventional 'DESTDIR' and 'PREFIX' environment variables for choosing install locations.",
		phony:  true,
		target: "install",
	}
	r.addDefinition(strings.TrimSpace(`
DESTDIR =
ifeq ($(shell uname -s),Darwin)
	PREFIX = /usr/local
else
	PREFIX = /usr
endif
	`))

	for _, bin := range binaries {
		if bin.InstallTo != "" {
			r.prerequisites = append(r.prerequisites, fmt.Sprintf("build/%s", bin.Name))
			r.recipe = append(r.recipe, fmt.Sprintf(
				`install -D -m 0755 build/%s "$(DESTDIR)$(PREFIX)/%s/%s"`,
				bin.Name, filepath.Clean(bin.InstallTo), bin.Name,
			))
		}
	}
	if len(r.recipe) == 0 {
		return rule{}, false
	}

	return r, true
}
