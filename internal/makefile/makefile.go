// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

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

//go:embed editorconfig
var editorconfig []byte

//go:embed license-scan-rules.json
var licenseRules []byte

//go:embed license-scan-overrides.jsonl
var scanOverrides []byte

// newMakefile defines the structure of the Makefile. Order is important as categories,
// rules, and definitions will appear in the exact order as they are defined.
func newMakefile(cfg core.Configuration, sr golang.ScanResult) *makefile {
	hasBinaries := len(cfg.Binaries) > 0
	runControllerGen := sr.KubernetesController
	if cfg.ControllerGen.Enabled != nil {
		runControllerGen = *cfg.ControllerGen.Enabled
	}
	// TODO: checking on GoVersion is only an aid until we can properly detect rust applications
	isGolang := sr.GoVersion != ""
	isSAPCC := strings.HasPrefix(cfg.Metadata.URL, "https://github.com/sapcc/") ||
		strings.HasPrefix(cfg.Metadata.URL, "https://github.com/SAP-cloud-infrastructure/") ||
		strings.HasPrefix(cfg.Metadata.URL, "https://github.com/cobaltcore-dev/") ||
		strings.HasPrefix(cfg.Metadata.URL, "https://github.com/ironcore-dev/") ||
		strings.HasPrefix(cfg.Metadata.URL, "https://github.wdf.sap.corp/") ||
		strings.HasPrefix(cfg.Metadata.URL, "https://github.tools.sap/")

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

	installTool := func(name, modulePath string) []string {
		return []string{
			fmt.Sprintf(`@if ! hash %s 2>/dev/null; then`, name) +
				fmt.Sprintf(` printf "\e[1;36m>> Installing %s (this may take a while)...\e[0m\n";`, name) +
				fmt.Sprintf(` go install %s; fi`, modulePath),
		}
	}

	var prepareStaticRecipe []string
	if isGolang {
		prepare.addRule(rule{
			description: "Install goimports required by goimports/static-check",
			phony:       true,
			target:      "install-goimports",
			recipe:      installTool("goimports", "golang.org/x/tools/cmd/goimports@latest"),
		})
		prepare.addRule(rule{
			description: "Install golangci-lint required by run-golangci-lint/static-check",
			phony:       true,
			target:      "install-golangci-lint",
			recipe:      installTool("golangci-lint", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"),
		})
		prepare.addRule(rule{
			description: "Install modernize required by modernize/static-check",
			phony:       true,
			target:      "install-modernize",
			recipe:      installTool("modernize", "golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest"),
		})
		prepareStaticRecipe = append(prepareStaticRecipe, "install-golangci-lint", "install-modernize")
	}
	if sr.UseGinkgo {
		prepare.addRule(rule{
			description: "Install ginkgo required when using it as test runner. This is used in CI before dropping privileges, you should probably install all the tools using your package manager",
			phony:       true,
			target:      "install-ginkgo",
			recipe:      installTool("ginkgo", "github.com/onsi/ginkgo/v2/ginkgo@latest"),
		})
		prepareStaticRecipe = append(prepareStaticRecipe, "install-ginkgo")
	}

	if isSAPCC {
		if isGolang {
			prepare.addRule(rule{
				description: "Install-go-licence-detector required by check-dependency-licenses/static-check",
				phony:       true,
				target:      "install-go-licence-detector",
				recipe:      installTool("go-licence-detector", "go.elastic.co/go-licence-detector@latest"),
			})
			prepareStaticRecipe = append(prepareStaticRecipe, "install-go-licence-detector")
		}
		prepare.addRule(rule{
			description: "Install addlicense required by check-license-headers/license-headers/static-check",
			phony:       true,
			target:      "install-addlicense",
			recipe:      installTool("addlicense", "github.com/google/addlicense@latest"),
		})
		prepareStaticRecipe = append(prepareStaticRecipe, "install-addlicense")
	}
	// add target for installing dependencies for `make static-check`
	prepare.addRule(rule{
		description:   "Install any tools required by static-check. This is used in CI before dropping privileges, you should probably install all the tools using your package manager",
		phony:         true,
		target:        "prepare-static-check",
		prerequisites: prepareStaticRecipe,
	})

	if runControllerGen {
		prepare.addRule(rule{
			description: "Install controller-gen required by static-check and build-all. This is used in CI before dropping privileges, you should probably install all the tools using your package manager",
			phony:       true,
			target:      "install-controller-gen",
			recipe:      installTool("controller-gen", "sigs.k8s.io/controller-tools/cmd/controller-gen@latest"),
		})

		prepare.addRule(rule{
			description: "Install setup-envtest required by check. This is used in CI before dropping privileges, you should probably install all the tools using your package manager",
			phony:       true,
			target:      "install-setup-envtest",
			recipe:      installTool("setup-envtest", "sigs.k8s.io/controller-runtime/tools/setup-envtest@latest"),
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
	if sr.HasBinInfo {
		build.addDefinition("")
		build.addDefinition("# These definitions are overridable, e.g. to provide fixed version/commit values when")
		build.addDefinition("# no .git directory is present or to provide a fixed build date for reproducibility.")
		build.addDefinition(`BININFO_VERSION     ?= $(shell git describe --tags --always --abbrev=7)`)
		build.addDefinition(`BININFO_COMMIT_HASH ?= $(shell git rev-parse --verify HEAD)`)
		build.addDefinition(`BININFO_BUILD_DATE  ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")`)
	}

	if hasBinaries {
		build.addRule(buildTargets(cfg.Binaries, sr, runControllerGen)...)
		if r, ok := installTarget(cfg.Binaries, &cfg); ok {
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

		if runControllerGen {
			crdOutputPath := "crd"
			if cfg.ControllerGen.CrdOutputPath != "" {
				crdOutputPath = cfg.ControllerGen.CrdOutputPath
			}
			components := strings.Split(sr.ModulePath, "/")
			roleName := components[len(components)-1]
			if cfg.ControllerGen.RBACRoleName != "" {
				roleName = cfg.ControllerGen.RBACRoleName
			}
			objectParams := ""
			if cfg.ControllerGen.ObjectHeaderFile != "" {
				objectParams = fmt.Sprintf(`:headerFile="%s"`, cfg.ControllerGen.ObjectHeaderFile)
			}
			test.addRule(rule{
				description: "Generate code for Kubernetes CRDs and deepcopy.",
				target:      "generate",
				recipe: []string{
					`@printf "\e[1;36m>> controller-gen\e[0m\n"`,
					fmt.Sprintf(`@controller-gen crd rbac:roleName=%s webhook paths="./..." output:crd:artifacts:config=%s`, roleName, crdOutputPath),
					fmt.Sprintf(`@controller-gen object%s paths="./..."`, objectParams),
				},
				prerequisites: []string{"install-controller-gen"},
			})
		}

		// add target to run golangci-lint
		test.addRule(rule{
			description:   "Install and run golangci-lint. Installing is used in CI, but you should probably install golangci-lint using your package manager.",
			phony:         true,
			target:        "run-golangci-lint",
			prerequisites: []string{"install-golangci-lint"},
			recipe: []string{
				`@printf "\e[1;36m>> golangci-lint\e[0m\n"`,
				`@golangci-lint config verify`,
				`@golangci-lint run`,
			},
		})

		// add target to run modernize
		test.addRule(rule{
			description:   "Install and run modernize. Installing is used in CI, but you should probably install modernize using your package manager.",
			phony:         true,
			target:        "run-modernize",
			prerequisites: []string{"install-modernize"},
			recipe: []string{
				`@printf "\e[1;36m>> modernize\e[0m\n"`,
				`@modernize $(GO_TESTPKGS)`,
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
		if runControllerGen {
			testRule.prerequisites = append(testRule.prerequisites, "generate", "install-setup-envtest")
			testRule.recipe = append(testRule.recipe, fmt.Sprintf(`KUBEBUILDER_ASSETS=$$(setup-envtest use %s -p path) %s`, sr.KubernetesVersion, goTest))
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

	// `go list .` does not work to get the package name because it requires a go file in the current directory
	// but some packages like concourse-swift-resource or gatekeeper-addons only have subpackages
	allSourceFilesExpr := `$(patsubst $(shell awk '$$1 == "module" {print $$2}' go.mod)%,.%/*.go,$(shell go list ./...))`
	if !isGolang {
		allSourceFilesExpr = `$(shell find -name *.rs)`
	}

	if isSAPCC {
		var ignoreOptions []string
		if cfg.GitHubWorkflow != nil {
			for _, pattern := range cfg.GitHubWorkflow.License.IgnorePatterns {
				// quoting avoids glob expansion
				ignoreOptions = append(ignoreOptions, fmt.Sprintf("-ignore %q", pattern))
			}
		}
		ignoreOptionsStr := strings.Join(append(ignoreOptions, "--"), " ")

		// Darwin's sed does not support sed -i but sed -i ""
		// xargs fails: command line cannot be assembled, too long
		general.addDefinition(strings.TrimSpace(`
UNAME_S := $(shell uname -s)
SED = sed
XARGS = xargs
ifeq ($(UNAME_S),Darwin)
	SED = gsed
	XARGS = gxargs
endif
`))
		dev.addRule(rule{
			description:   "Add (or overwrite) license headers on all non-vendored source code files.",
			target:        "license-headers",
			phony:         true,
			prerequisites: []string{"install-addlicense"},
			recipe: []string{
				`@printf "\e[1;36m>> addlicense (for license headers on source code files)\e[0m\n"`,
				// We must use gawk to use gnu awk on Darwin
				fmt.Sprintf(`@printf "%%s\0" %s | $(XARGS) -0 -I{} bash -c 'year="$$(grep 'Copyright' {} | head -n1 | grep -E -o '"'"'[0-9]{4}(-[0-9]{4})?'"'"')"; gawk -i inplace '"'"'{if (display) {print} else {!/^\/\*/ && !/^\*/}}; {if (!display && $$0 ~ /^(package |$$)/) {display=1} else { }}'"'"' {}; addlicense -c "SAP SE or an SAP affiliate company" -s=only -y "$$year" %s {}; $(SED) -i '"'"'1s+// Copyright +// SPDX-FileCopyrightText: +'"'"' {}'`, allSourceFilesExpr, ignoreOptionsStr),
				`@printf "\e[1;36m>> reuse annotate (for license headers on other files)\e[0m\n"`,
				`@reuse lint -j | jq -r '.non_compliant.missing_licensing_info[]' | grep -vw vendor | $(XARGS) reuse annotate -c 'SAP SE or an SAP affiliate company' -l Apache-2.0 --skip-unrecognised`,
				`@printf "\e[1;36m>> reuse download --all\e[0m\n"`,
				`@reuse download --all`,
				`@printf "\e[1;35mPlease review the changes. If *.license files were generated, consider instructing go-makefile-maker to add overrides to REUSE.toml instead.\e[0m\n"`,
			},
		})

		tidyTarget := "tidy-deps"
		if cfg.Golang.EnableVendoring {
			tidyTarget = "vendor"
		}
		dev.addRule(rule{
			description:   "Check license headers in all non-vendored .go files.",
			target:        "check-license-headers",
			phony:         true,
			prerequisites: []string{"install-addlicense", tidyTarget},
			recipe: []string{
				`@printf "\e[1;36m>> addlicense --check\e[0m\n"`,
				fmt.Sprintf(`@addlicense --check %s %s`, ignoreOptionsStr, allSourceFilesExpr),
			},
		})

		if isGolang {
			must.Succeed(os.WriteFile(".editorconfig", editorconfig, 0o666))

			licenseRulesFile := ".license-scan-rules.json"
			must.Succeed(os.WriteFile(licenseRulesFile, licenseRules, 0o666))

			scanOverridesFile := ".license-scan-overrides.jsonl"
			must.Succeed(os.WriteFile(scanOverridesFile, scanOverrides, 0o666))

			dev.addRule(rule{
				description:   "Check all dependency licenses using go-licence-detector.",
				target:        "check-dependency-licenses",
				phony:         true,
				prerequisites: []string{"install-go-licence-detector"},
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
		staticCheckPrerequisites := []string{"run-golangci-lint", "run-modernize"}
		if isSAPCC {
			staticCheckPrerequisites = append(staticCheckPrerequisites, "check-dependency-licenses", "check-license-headers")
		}
		test.addRule(rule{
			description:   "Run static code checks",
			phony:         true,
			target:        "static-check",
			prerequisites: staticCheckPrerequisites,
		})

		dev.addRule(rule{
			description:   "Run goimports on all non-vendored .go files",
			phony:         true,
			target:        "goimports",
			prerequisites: []string{"install-goimports"},
			recipe: []string{
				fmt.Sprintf(`@printf "\e[1;36m>> goimports -w -local %s\e[0m\n"`, cfg.Metadata.URL),
				fmt.Sprintf(`@goimports -w -local %s %s`, sr.ModulePath, allSourceFilesExpr),
			},
		})

		dev.addRule(rule{
			description:   "Run modernize on all non-vendored .go files",
			phony:         true,
			target:        "modernize",
			prerequisites: []string{"install-modernize"},
			recipe: []string{
				`@printf "\e[1;36m>> modernize -fix ./...\e[0m\n"`,
				`@modernize -fix ./...`,
			},
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

func buildTargets(binaries []core.BinaryConfiguration, sr golang.ScanResult, runControllerGen bool) []rule {
	result := make([]rule, 0, len(binaries)+1)
	buildAllRule := rule{
		description: "Build all binaries.",
		target:      "build-all",
	}
	if runControllerGen {
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
				"env $(GO_BUILDENV) go build $(GO_BUILDFLAGS) -ldflags '%s $(GO_LDFLAGS)' -o build/%s %s",
				makeDefaultLinkerFlags(bin.Name, sr),
				bin.Name, bin.FromPackage,
			)},
		}

		if runControllerGen {
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
func installTarget(binaries []core.BinaryConfiguration, cfg *core.Configuration) (rule, bool) {
	r := rule{
		description: "Install all binaries. " +
			"This option understands the conventional 'DESTDIR' and 'PREFIX' environment variables for choosing install locations.",
		phony:  true,
		target: "install",
	}
	r.addDefinition(strings.TrimSpace(`
DESTDIR =%s
ifeq ($(shell uname -s),Darwin)
	PREFIX = /usr/local
else
	PREFIX = /usr
endif
`), cfg.Variable("DESTDIR", ""))

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
