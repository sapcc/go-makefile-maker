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
	"sort"
	"strings"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

//go:embed license-scan-rules.json
var licenseRules []byte

//go:embed license-scan-overrides.jsonl
var scanOverrides []byte

// newMakefile defines the structure of the Makefile. Order is important as categories,
// rules, and definitions will appear in the exact order as they are defined.
func newMakefile(cfg *core.Configuration, sr golang.ScanResult) *makefile {
	hasBinaries := len(cfg.Binaries) > 0
	// TODO: checking on GoVersion is only an aid until we can properly detect rust applications
	isGolang := sr.GoVersion != ""
	isSAPCC := strings.HasPrefix(cfg.Metadata.URL, "https://github.com/sapcc") ||
		strings.HasPrefix(cfg.Metadata.URL, "https://github.wdf.sap.corp") ||
		strings.HasPrefix(cfg.Metadata.URL, "https://github.tools.sap")

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
	// Prepare
	prepare := category{name: "prepare"}

	// add target for installing dependencies for `make static-check`
	var prepareStaticRecipe []string
	if isGolang {
		prepareStaticRecipe = append(prepareStaticRecipe, []string{
			`@if ! hash golangci-lint 2>/dev/null; then` +
				` printf "\e[1;36m>> Installing golangci-lint (this may take a while)...\e[0m\n";` +
				` go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; fi`,
		}...)
	}
	if isSAPCC {
		if isGolang {
			prepareStaticRecipe = append(prepareStaticRecipe, []string{
				`@if ! hash go-licence-detector 2>/dev/null; then` +
					` printf "\e[1;36m>> Installing go-licence-detector...\e[0m\n";` +
					` go install go.elastic.co/go-licence-detector@latest; fi`,
			}...)
		}
		prepareStaticRecipe = append(prepareStaticRecipe, []string{
			`@if ! hash addlicense 2>/dev/null; then ` +
				` printf "\e[1;36m>> Installing addlicense...\e[0m\n"; ` +
				` go install github.com/google/addlicense@latest; fi`,
		}...)
	}
	prepare.addRule(rule{
		description:   "Install any tools required by static-check. This is used in CI before dropping privileges, you should probably install all the tools using your package manager",
		phony:         true,
		target:        "prepare-static-check",
		recipe:        prepareStaticRecipe,
		prerequisites: []string{},
	})

	if sr.KubernetesController {
		prepare.addRule(rule{
			description: "Install controller-gen required by static-check and build-all. This is used in CI before dropping privileges, you should probably install all the tools using your package manager",
			phony:       true,
			recipe: []string{
				`@if ! hash controller-gen 2>/dev/null; then` +
					` printf "\e[1;36m>> Installing controller-gen...\e[0m\n";` +
					` go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest; fi`,
			},
			target: "install-controller-gen",
		})

		prepare.addRule(rule{
			description: "Install setup-envtest required by check. This is used in CI before dropping privileges, you should probably install all the tools using your package manager",
			phony:       true,
			recipe: []string{
				`@if ! hash setup-envtest 2>/dev/null; then` +
					` printf "\e[1;36m>> Installing setup-envtest...\e[0m\n";` +
					` go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest; fi`,
			},
			target: "install-setup-envtest",
		})
	}

	if sr.UseGinkgo {
		prepare.addRule(rule{
			description: "Install ginkgo required when using it as test runner. This is used in CI before dropping privileges, you should probably install all the tools using your package manager",
			phony:       true,
			recipe: []string{
				`@if ! hash ginkgo 2>/dev/null; then` +
					` printf "\e[1;36m>> Installing ginkgo...\e[0m\n";` +
					` go install github.com/onsi/ginkgo/v2/ginkgo@latest; fi`,
			},
			target: "install-ginkgo",
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
		var names []string
		for key := range cfg.Golang.LdFlags {
			names = append(names, key)
		}
		sort.Strings(names)

		for _, name := range names {
			value := cfg.Golang.LdFlags[name]
			defaultLdFlags += fmt.Sprintf("%s=$(%s) ", name, value)
		}
	}

	if isGolang {
		build.addDefinition("GO_BUILDFLAGS =%s", cfg.Variable("GO_BUILDFLAGS", defaultBuildFlags))
		build.addDefinition("GO_LDFLAGS =%s", cfg.Variable("GO_LDFLAGS", strings.TrimSpace(defaultLdFlags)))
		build.addDefinition("GO_TESTENV =%s", cfg.Variable("GO_TESTENV", ""))
		build.addDefinition("GO_BUILDENV =%s", cfg.Variable("GO_BUILDENV", ""))
	}
	if sr.KubernetesController {
		build.addDefinition("TESTBIN=$(shell pwd)/testbin")
	}
	if sr.HasBinInfo {
		build.addDefinition("")
		build.addDefinition("# These definitions are overridable, e.g. to provide fixed version/commit values when")
		build.addDefinition("# no .git directory is present or to provide a fixed build date for reproducibility.")
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

	if isGolang {
		test.addDefinition(`# which packages to test with test runner`)
		testPkgGreps := ""
		if cfg.Test.Only != "" {
			testPkgGreps += fmt.Sprintf(" | grep -E '%s'", cfg.Test.Only)
		}
		if cfg.Test.Except != "" {
			testPkgGreps += fmt.Sprintf(" | grep -Ev '%s'", cfg.Test.Except)
		}
		pathVar := "ImportPath"
		// ginkgo only understands relative names eg ./config or config but not example.com/package/config
		if sr.UseGinkgo {
			pathVar = "Dir"
		}
		test.addDefinition(`GO_TESTPKGS := $(shell go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.%s}}{{end}}' ./...%s)`, pathVar, testPkgGreps)
		test.addDefinition(strings.TrimSpace(`
ifeq ($(GO_TESTPKGS),)
GO_TESTPKGS := ./...
endif
  `))

		test.addDefinition(`# which packages to measure coverage for`)
		coverPkgGreps := ""
		if cfg.Coverage.Only != "" {
			coverPkgGreps += fmt.Sprintf(" | grep -E '%s'", cfg.Coverage.Only)
		}
		if cfg.Coverage.Except != "" {
			coverPkgGreps += fmt.Sprintf(" | grep -Ev '%s'", cfg.Coverage.Except)
		}
		test.addDefinition(`GO_COVERPKGS := $(shell go list ./...%s)`, coverPkgGreps)
	}

	test.addDefinition(`# to get around weird Makefile syntax restrictions, we need variables containing nothing, a space and comma`)
	test.addDefinition(`null :=`)
	test.addDefinition(`space := $(null) $(null)`)
	test.addDefinition(`comma := ,`)

	if isGolang {
		// add main testing target
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

		if sr.KubernetesController {
			components := strings.Split(sr.ModulePath, "/")
			roleName := components[len(components)-1]
			test.addRule(rule{
				description: "Generate code for Kubernetes CRDs and deepcopy.",
				target:      "generate",
				recipe: []string{
					`@printf "\e[1;36m>> controller-gen\e[0m\n"`,
					fmt.Sprintf(`@controller-gen crd rbac:roleName=%s paths="./..." output:crd:artifacts:config=crd`, roleName),
					`@controller-gen object paths=./...`,
				},
				prerequisites: []string{"install-controller-gen"},
			})
		}

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

		// add targets for test runner incl. coverage report
		testRule := rule{
			description: "Run tests and generate coverage report.",
			phony:       true,
			target:      "build/cover.out",
			// We use order only prerequisite because this target is used in CI.
			orderOnlyPrerequisites: []string{"build"},
			recipe: []string{
				`@printf "\e[1;36m>> Running tests\e[0m\n"`,
			},
		}

		testRunner := "go test -shuffle=on -p 1 -coverprofile=$@"
		if sr.UseGinkgo {
			testRunner = "ginkgo run --randomize-all -output-dir=build"
			testRule.prerequisites = append(testRule.prerequisites, "install-ginkgo")
		}
		goTest := fmt.Sprintf(`%s $(GO_BUILDFLAGS) -ldflags '%s $(GO_LDFLAGS)' -covermode=count -coverpkg=$(subst $(space),$(comma),$(GO_COVERPKGS)) $(GO_TESTPKGS)`,
			testRunner, makeDefaultLinkerFlags(path.Base(sr.ModulePath), sr))
		if sr.KubernetesController {
			testRule.prerequisites = append(testRule.prerequisites, "generate", "install-setup-envtest")
			testRule.recipe = append(testRule.recipe, fmt.Sprintf(`KUBEBUILDER_ASSETS="$(shell setup-envtest use %s --bin-dir $(TESTBIN) -p path)" %s`, sr.KubernetesVersion, goTest))
		} else {
			testRule.recipe = append(testRule.recipe, `@env $(GO_TESTENV) `+goTest)
		}
		if sr.UseGinkgo {
			testRule.recipe = append(testRule.recipe, `@mv build/coverprofile.out build/cover.out`)
		}

		test.addRule(testRule)

		test.addRule(rule{
			description:   "Generate an HTML file with source code annotations from the coverage report.",
			target:        "build/cover.html",
			prerequisites: []string{"build/cover.out"},
			recipe: []string{
				`@printf "\e[1;36m>> go tool cover > build/cover.html\e[0m\n"`,
				`@go tool cover -html $< -o $@`,
			},
		})
	}

	///////////////////////////////////////////////////////////////////////////
	// Development
	dev := category{name: "development"}

	if isGolang {
		// ensure that build directory exists
		dev.addRule(rule{
			target: "build",
			recipe: []string{`@mkdir $@`},
		})

		// add tidy-deps or vendor target
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
	}

	if isSAPCC {
		// `go list .` does not work to get the package name because it requires a go file in the current directory
		// but some packages like concourse-swift-resource or gatekeeper-addons only have subpackages
		allSourceFilesExpr := `$(patsubst $(shell awk '$$1 == "module" {print $$2}' go.mod)%,.%/*.go,$(shell go list ./...))`
		if !isGolang {
			allSourceFilesExpr = `$(shell find -name *.rs)`
		}

		var ignoreOptions []string
		if cfg.GitHubWorkflow != nil {
			for _, pattern := range cfg.GitHubWorkflow.License.IgnorePatterns {
				// quoting avoids glob expansion
				ignoreOptions = append(ignoreOptions, fmt.Sprintf("-ignore %q", pattern))
			}
		}
		ignoreOptionsStr := strings.Join(append(ignoreOptions, "--"), " ")

		dev.addRule(rule{
			description:   "Add license headers to all non-vendored source code files.",
			target:        "license-headers",
			phony:         true,
			prerequisites: []string{"prepare-static-check"},
			recipe: []string{
				`@printf "\e[1;36m>> addlicense\e[0m\n"`,
				fmt.Sprintf(`@addlicense -c "SAP SE" %s %s`, ignoreOptionsStr, allSourceFilesExpr)},
		})

		dev.addRule(rule{
			description:   "Check license headers in all non-vendored .go files.",
			target:        "check-license-headers",
			phony:         true,
			prerequisites: []string{"prepare-static-check"},
			recipe: []string{
				`@printf "\e[1;36m>> addlicense --check\e[0m\n"`,
				fmt.Sprintf(`@addlicense --check %s %s`, ignoreOptionsStr, allSourceFilesExpr)},
		})

		if isGolang {
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
	}

	if isGolang {
		// add target for static code checks
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
	}

	// add cleaning target
	dev.addRule(rule{
		description: "Run git clean.",
		target:      "clean",
		phony:       true,
		recipe:      []string{"git clean -dxf build"},
	})

	return &makefile{
		categories: []category{
			general,
			prepare,
			build,
			test,
			dev,
		},
	}
}

func buildTargets(binaries []core.BinaryConfiguration, sr golang.ScanResult) []rule {
	result := make([]rule, 0, len(binaries)+1)
	buildAllRule := rule{
		description: "Build all binaries.",
		target:      "build-all",
	}
	if sr.KubernetesController {
		buildAllRule.prerequisites = []string{"install-controller-gen"}
	}
	result = append(result, buildAllRule)

	allPrerequisites := make([]string, 0, len(binaries))
	for _, bin := range binaries {
		r := rule{
			description: fmt.Sprintf("Build %s.", bin.Name),
			phony:       true,
			target:      "build/" + bin.Name,
			recipe: []string{fmt.Sprintf(
				"@env $(GO_BUILDENV) go build $(GO_BUILDFLAGS) -ldflags '%s $(GO_LDFLAGS)' -o build/%s %s",
				makeDefaultLinkerFlags(bin.Name, sr),
				bin.Name, bin.FromPackage,
			)},
		}

		if sr.KubernetesController {
			r.prerequisites = append(r.prerequisites, "generate")
		}

		result = append(result, r)
		allPrerequisites = append(allPrerequisites, r.target)
	}
	result[0].prerequisites = allPrerequisites

	return result
}

func makeDefaultLinkerFlags(binaryName string, sr golang.ScanResult) string {
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
			r.prerequisites = append(r.prerequisites, "build/"+bin.Name)
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
