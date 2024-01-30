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
	_ "embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

//go:embed license-scan-rules.json
var licenseRules []byte

//go:embed license-scan-overrides.jsonl
var scanOverrides []byte

// newMakefile defines the structure of the Makefile. Order is important as categories,
// rules, and definitions will appear in the exact order as they are defined.
func newMakefile(cfg *core.Configuration, sr core.ScanResult) *makefile {
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

	var defaultBuildFlags, defaultLdFlags string

	if cfg.Golang.EnableVendoring {
		defaultBuildFlags = "-mod vendor"
	}

	if len(cfg.Golang.LdFlags) > 0 {
		for name, value := range cfg.Golang.LdFlags {
			defaultLdFlags += fmt.Sprintf("%s=$(%s)", name, value)
		}
	}

	build.addDefinition("GO_BUILDFLAGS =%s", cfg.Variable("GO_BUILDFLAGS", defaultBuildFlags))
	build.addDefinition("GO_LDFLAGS =%s", cfg.Variable("GO_LDFLAGS", defaultLdFlags))
	build.addDefinition("GO_TESTENV =%s", cfg.Variable("GO_TESTENV", ""))
	if sr.HasBinInfo {
		build.addDefinition("")
		build.addDefinition("# These definitions are overridable, e.g. to provide fixed version/commit values when")
		build.addDefinition("# no .git directory is present or to provide a fixed build date for reproducability.")
		build.addDefinition(`BININFO_VERSION     ?= $(shell git describe --tags --always --abbrev=7)`)
		build.addDefinition(`BININFO_COMMIT_HASH ?= $(shell git rev-parse --verify HEAD)`)
		build.addDefinition(`BININFO_BUILD_DATE  ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")`)
	}

	if hasBinaries {
		build.addRule(buildTargets(cfg.Binaries, sr)...)
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
		testPkgGreps += fmt.Sprintf(" | grep -E '%s'", cfg.Test.Only)
	}
	if cfg.Test.Except != "" {
		testPkgGreps += fmt.Sprintf(" | grep -Ev '%s'", cfg.Test.Except)
	}
	test.addDefinition(`GO_TESTPKGS := $(shell go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...%s)`, testPkgGreps)

	test.addDefinition(`# which packages to measure coverage for`)
	coverPkgGreps := ""
	if cfg.Coverage.Only != "" {
		coverPkgGreps += fmt.Sprintf(" | grep -E '%s'", cfg.Coverage.Only)
	}
	if cfg.Coverage.Except != "" {
		coverPkgGreps += fmt.Sprintf(" | grep -Ev '%s'", cfg.Coverage.Except)
	}
	test.addDefinition(`GO_COVERPKGS := $(shell go list ./...%s)`, coverPkgGreps)
	test.addDefinition(`# to get around weird Makefile syntax restrictions, we need variables containing nothing, a space and comma`)
	test.addDefinition(`null :=`)
	test.addDefinition(`space := $(null) $(null)`)
	test.addDefinition(`comma := ,`)

	isSAPCC := strings.HasPrefix(sr.ModulePath, "github.com/sapcc") || strings.HasPrefix(sr.ModulePath, "github.wdf.sap.corp") || strings.HasPrefix(sr.ModulePath, "github.tools.sap")

	//add main testing target
	checkPrerequisites := []string{"static-check", "build/cover.html"}
	if hasBinaries {
		checkPrerequisites = append(checkPrerequisites, "build-all")
	}
	test.addRule(rule{
		description:   "Run the test suite (unit tests and golangci-lint).",
		phony:         true,
		target:        "check",
		prerequisites: checkPrerequisites,
		recipe:        []string{`@printf "\e[1;32m>> All checks successful.\e[0m\n"`},
	})

	//add target for installing dependencies for `make check`
	prepareStaticRecipe := []string{
		`@if ! hash golangci-lint 2>/dev/null; then` +
			` printf "\e[1;36m>> Installing golangci-lint (this may take a while)...\e[0m\n";` +
			` go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; fi`,
	}
	if isSAPCC {
		prepareStaticRecipe = append(prepareStaticRecipe, []string{
			`@if ! hash go-licence-detector 2>/dev/null; then` +
				` printf "\e[1;36m>> Installing go-licence-detector...\e[0m\n";` +
				` go install go.elastic.co/go-licence-detector@latest; fi`,
			`@if ! hash addlicense 2>/dev/null; then ` +
				` printf "\e[1;36m>> Installing addlicense...\e[0m\n"; ` +
				` go install github.com/google/addlicense@latest; fi`,
		}...)
	}
	test.addRule(rule{
		description: "Install any tools required by static-check. This is used in CI before dropping privileges, you should probably install all the tools using your package manager",
		phony:       true,
		target:      "prepare-static-check",
		recipe:      prepareStaticRecipe,
	})

	// add target to run golangci-lint
	test.addRule(rule{
		description:   "Install and run golangci-lint. Installing is used in CI, but you should probably install golangci-lint using your package manager.",
		phony:         true,
		target:        "run-golangci-lint",
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
			fmt.Sprintf(
				`@env $(GO_TESTENV) go test $(GO_BUILDFLAGS) -ldflags '%s $(GO_LDFLAGS)' -shuffle=on -p 1 -coverprofile=$@ -covermode=count -coverpkg=$(subst $(space),$(comma),$(GO_COVERPKGS)) $(GO_TESTPKGS)`,
				makeDefaultLinkerFlags(path.Base(sr.MustModulePath()), sr),
			),
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
	if cfg.Golang.EnableVendoring {
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

	if isSAPCC {
		allGoFilesExpr := `$(patsubst $(shell go list .)%,.%/*.go,$(shell go list ./...))`

		ignoreOptions := make([]string, len(cfg.GitHubWorkflow.License.IgnorePatterns))
		for idx, pattern := range cfg.GitHubWorkflow.License.IgnorePatterns {
			//quoting avoids glob expansion
			ignoreOptions[idx] = fmt.Sprintf("-ignore %q", pattern)
		}

		dev.addRule(rule{
			description:   "Add license headers to all non-vendored .go files.",
			target:        "license-headers",
			phony:         true,
			prerequisites: []string{"prepare-static-check"},
			recipe: []string{
				`@printf "\e[1;36m>> addlicense\e[0m\n"`,
				fmt.Sprintf(`@addlicense -c "SAP SE" %s -- %s`,
					strings.Join(ignoreOptions, " "),
					allGoFilesExpr,
				)},
		})

		dev.addRule(rule{
			description:   "Check license headers in all non-vendored .go files.",
			target:        "check-license-headers",
			phony:         true,
			prerequisites: []string{"prepare-static-check"},
			recipe: []string{
				`@printf "\e[1;36m>> addlicense --check\e[0m\n"`,
				fmt.Sprintf(`@addlicense --check %s -- %s`,
					strings.Join(ignoreOptions, " "),
					allGoFilesExpr,
				)},
		})

		licenseRulesFile := ".license-scan-rules.json"
		must.Succeed(os.WriteFile(licenseRulesFile, licenseRules, 0666))

		scanOverridesFile := ".license-scan-overrides.jsonl"
		must.Succeed(os.WriteFile(scanOverridesFile, scanOverrides, 0666))

		dev.addRule(rule{
			description:   "Check all dependency licenses using go-licence-detector.",
			target:        "check-dependency-licenses",
			phony:         true,
			prerequisites: []string{"prepare-static-check"},
			recipe: []string{

				`@printf "\e[1;36m>> go-licence-detector\e[0m\n"`,
				fmt.Sprintf(`@go list -m -mod=readonly -json all | go-licence-detector -includeIndirect -rules %s -overrides %s`,
					licenseRulesFile, scanOverridesFile),
			},
		})
	}

	//add target for static code checks
	staticCheckPrerequisites := []string{"run-golangci-lint"}
	if isSAPCC {
		staticCheckPrerequisites = append(staticCheckPrerequisites, "check-dependency-licenses", "check-license-headers")
	}
	test.addRule(rule{
		description:   "Run static code checks",
		phony:         true,
		target:        "static-check",
		prerequisites: staticCheckPrerequisites,
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

func buildTargets(binaries []core.BinaryConfiguration, sr core.ScanResult) []rule {
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
				"go build $(GO_BUILDFLAGS) -ldflags '%s $(GO_LDFLAGS)' -o build/%s %s",
				makeDefaultLinkerFlags(bin.Name, sr),
				bin.Name, bin.FromPackage,
			)},
		}

		result = append(result, r)
		allPrerequisites = append(allPrerequisites, r.target)
	}
	result[0].prerequisites = allPrerequisites

	return result
}

func makeDefaultLinkerFlags(binaryName string, sr core.ScanResult) string {
	flags := "-s -w"

	if sr.HasBinInfo {
		flags += fmt.Sprintf(
			` -X %[1]s.binName=%[2]s -X %[1]s.version=$(BININFO_VERSION) -X %[1]s.commit=$(BININFO_COMMIT_HASH) -X %[1]s.buildDate=$(BININFO_BUILD_DATE)`,
			"github.com/sapcc/go-api-declarations/bininfo", binaryName,
		)
	}

	return flags
}

// installTarget also returns a bool that tells whether the install target was requested in the config.
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
			// stupid MacOS does not have -D
			r.recipe = append(r.recipe, fmt.Sprintf(
				`install -d -m 0755 "$(DESTDIR)$(PREFIX)/%s"`, filepath.Clean(bin.InstallTo),
			))
			r.recipe = append(r.recipe, fmt.Sprintf(
				`install -m 0755 build/%s "$(DESTDIR)$(PREFIX)/%s/%s"`,
				bin.Name, filepath.Clean(bin.InstallTo), bin.Name,
			))
		}
	}
	if len(r.recipe) == 0 {
		return rule{}, false
	}

	return r, true
}
