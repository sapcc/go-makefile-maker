// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package makefile

import (
	_ "embed"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
	"github.com/sapcc/go-makefile-maker/internal/util"
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
	runControllerGen := cfg.ControllerGen.Enabled.UnwrapOr(sr.KubernetesController)
	// TODO: checking on GoVersion is only an aid until we can properly detect rust applications
	isGolang := sr.GoVersion != ""
	isSAPCC := cfg.Metadata.IsSAPProject()

	///////////////////////////////////////////////////////////////////////////
	// General
	general := category{name: "general"}

	general.addDefinition(`# macOS ships with make 3.81 from 2006, which does not support all the features that we want (e.g. --warn-undefined-variables)
ifeq ($(MAKE_VERSION),3.81)
  ifeq (,$(shell which gmake 2>/dev/null))
    $(error We do not support this "make" version ($(MAKE_VERSION)) which is two decades old. Please install a newer version, e.g. using "brew install make")
  else
    $(error We do not support this "make" version ($(MAKE_VERSION)) which is two decades old. You have a newer GNU make installed, so please run "gmake" instead)
  endif
endif
`)

	// WARNING: Do not remove this just because it may be inconvenient to you. Learn to work with it.
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
			description: "Install modernize required by run-modernize/static-check",
			phony:       true,
			target:      "install-modernize",
			recipe:      installTool("modernize", "golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest"),
		})
		prepareStaticRecipe = append(prepareStaticRecipe, "install-golangci-lint", "install-modernize")
	}

	if cfg.ShellCheck.Enabled.UnwrapOr(true) {
		prepare.addRule(rule{
			description: "Install shellcheck required by run-shellcheck/static-check",
			phony:       true,
			target:      "install-shellcheck",
			recipe: []string{
				`@if ! hash shellcheck 2>/dev/null; then` +
					` printf "\e[1;36m>> Installing shellcheck...\e[0m\n";` +
					` SHELLCHECK_ARCH=$(shell uname -m);` +
					` if [[ "$$SHELLCHECK_ARCH" == "arm64" ]]; then SHELLCHECK_ARCH=aarch64; fi;` +
					` SHELLCHECK_OS=$(shell uname -s | tr '[:upper:]' '[:lower:]');` +
					` SHELLCHECK_VERSION="stable";` +
					` if command -v curl >/dev/null 2>&1; then GET="curl -sLo-"; elif command -v wget >/dev/null 2>&1; then GET="wget -O-"; else echo "Didn't find curl or wget to download shellcheck"; exit 2; fi;` +
					` $$GET "https://github.com/koalaman/shellcheck/releases/download/$$SHELLCHECK_VERSION/shellcheck-$$SHELLCHECK_VERSION.$$SHELLCHECK_OS.$$SHELLCHECK_ARCH.tar.xz" | tar -Jxf -;` +
					// hardcoding go here is not nice but since we mainly target go it should be acceptable
					` BIN=$$(go env GOBIN); if [[ -z $$BIN ]]; then BIN=$$(go env GOPATH)/bin; fi;` +
					` install -Dm755 shellcheck-$$SHELLCHECK_VERSION/shellcheck -t "$$BIN";` +
					` rm -rf shellcheck-$$SHELLCHECK_VERSION; fi`,
			},
		})
		prepareStaticRecipe = append(prepareStaticRecipe, "install-shellcheck")
	}

	if isGolang && (cfg.License.AddHeaders.UnwrapOr(isSAPCC) || cfg.License.CheckDependencies.UnwrapOr(isSAPCC)) {
		prepare.addRule(rule{
			description: "Install-go-licence-detector required by check-dependency-licenses/static-check",
			phony:       true,
			target:      "install-go-licence-detector",
			recipe:      installTool("go-licence-detector", "go.elastic.co/go-licence-detector@latest"),
		})
		prepareStaticRecipe = append(prepareStaticRecipe, "install-go-licence-detector")
	}
	if cfg.License.AddHeaders.UnwrapOr(isSAPCC) {
		prepare.addRule(rule{
			description: "Install addlicense required by check-license-headers/license-headers/static-check",
			phony:       true,
			target:      "install-addlicense",
			recipe:      installTool("addlicense", "github.com/google/addlicense@latest"),
		})
		prepareStaticRecipe = append(prepareStaticRecipe, "install-addlicense")

		prepare.addRule(rule{
			description: "Install reuse required by license-headers/check-reuse",
			phony:       true,
			target:      "install-reuse",
			recipe: []string{
				`@if ! hash reuse 2>/dev/null; then` +
					` if ! hash pipx 2>/dev/null; then` +
					` printf "\e[1;31m>> You are required to manually intervene to install reuse as go-makefile-maker cannot automatically resolve installing reuse on all setups.\e[0m\n";` +
					` printf "\e[1;31m>> The preferred way for go-makefile-maker to install python tools after nix-shell is pipx which could not be found. Either install pipx using your package manager or install reuse using your package manager if at least version 6 is available.\e[0m\n";` +
					` printf "\e[1;31m>> As your Python was likely installed by your package manager, just doing pip install --user sadly does no longer work as pip issues a warning about breaking your system. Generally running --break-system-packages with --user is safe to do but you should only run this command if you can resolve issues with it yourself: pip3 install --user --break-system-packages reuse\e[0m\n";` +
					` else` +
					` printf "\e[1;36m>> Installing reuse...\e[0m\n";` +
					` pipx install reuse;` +
					` fi;` +
					` fi`,
			},
		})
		prepareStaticRecipe = append(prepareStaticRecipe, "install-reuse")
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
		// WARNING: DO NOT CHANGE THIS WITHOUT FIRST UNDERSTANDING THE CONSEQUENCES.
		// All changes have to be okay'd by Stefan Majewsky to avoid breaking setups.
		build.addDefinition("# To add additional flags or values (before the default ones), specify the variable in the environment, e.g. `GO_BUILDFLAGS='-tags experimental' make`.")
		build.addDefinition("# To override the default flags or values, specify the variable on the command line, e.g. `make GO_BUILDFLAGS='-tags experimental'`.")
		build.addDefinition("GO_BUILDFLAGS +=%s", cfg.Variable("GO_BUILDFLAGS", defaultBuildFlags))
		build.addDefinition("GO_LDFLAGS +=%s", cfg.Variable("GO_LDFLAGS", strings.TrimSpace(defaultLdFlags)))
		build.addDefinition("GO_TESTFLAGS +=%s", cfg.Variable("GO_TESTFLAGS", ""))
		build.addDefinition("GO_TESTENV +=%s", cfg.Variable("GO_TESTENV", ""))
		build.addDefinition("GO_BUILDENV +=%s", cfg.Variable("GO_BUILDENV", ""))
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
			applyconfigurationParams := ""
			if cfg.ControllerGen.ApplyconfigurationHeaderFile != "" {
				applyconfigurationParams = fmt.Sprintf(`:headerFile="%s"`, cfg.ControllerGen.ApplyconfigurationHeaderFile)
			}
			test.addRule(rule{
				description: "Generate code for Kubernetes CRDs and deepcopy.",
				target:      "generate",
				recipe: []string{
					`@printf "\e[1;36m>> controller-gen\e[0m\n"`,
					fmt.Sprintf(`@controller-gen crd rbac:roleName=%s webhook paths="./..." output:crd:artifacts:config=%s`, roleName, crdOutputPath),
					fmt.Sprintf(`@controller-gen object%s paths="./..."`, objectParams),
					fmt.Sprintf(`@controller-gen applyconfiguration%s paths="./..."`, applyconfigurationParams),
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

		if cfg.ShellCheck.Enabled.UnwrapOr(true) {
			// add target to run shellcheck
			ignorePathArgs := ""
			for _, path := range cfg.ShellCheck.AllIgnorePaths(cfg.Golang) {
				if strings.HasPrefix(path, "/") {
					logg.Fatal("ShellCheck ignore paths must not start with a slash, got: %s", path)
				}

				// https://github.com/ludeeus/action-shellcheck/blob/master/action.yaml#L120-L124
				if !strings.HasPrefix(path, "./") {
					ignorePathArgs += fmt.Sprintf(" \\( -path '*/%s/*' -prune \\) -o", path)
				}
				ignorePathArgs += fmt.Sprintf(" \\( -path '%s' -prune \\) -o", path)
			}
			// partly taken from https://github.com/ludeeus/action-shellcheck/blob/master/action.yaml#L164-L196
			test.addRule(rule{
				description:   "Install and run shellcheck. Installing is used in CI, but you should probably install shellcheck using your package manager.",
				phony:         true,
				target:        "run-shellcheck",
				prerequisites: []string{"install-shellcheck"},
				recipe: []string{
					`@printf "\e[1;36m>> shellcheck\e[0m\n"`,
					fmt.Sprintf(`@find . %s -type f \( -name '*.bash' -o -name '*.ksh' -o -name '*.zsh' -o -name '*.sh' -o -name '*.shlib' \) -exec shellcheck %s {} +`, strings.TrimSpace(ignorePathArgs), cfg.ShellCheck.Opts),
				},
			})
		}

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

		// The current design of `easypg.WithTestDB()` came about because we wanted to get rid of the `./testing/with-postgres-db.sh` wrapper.
		// Since wrappers outside of go test are not desirable, we can only hook into the TestMain level,
		// and then there is no good way to deal with multiple test binaries running in parallel.
		// We could use file locking to make them wait for each other, but that would just reverse this change with extra steps.
		//
		// usesPostgres reflects whether `github.com/lib/pq` is loaded. `github.com/sapcc/go-bits/easypg` hard depends on `github.com/lib/pq`.
		singleThreaded := ""
		if sr.UsesPostgres {
			singleThreaded = "-p 1 "
		}

		// NOTE: Ginkgo will always write the coverage profile as "coverprofile.out", so we will choose the same path for non-Ginkgo tests, too.
		// The actual final path is build/cover.out, which will be filled by a post-processing step below.
		testRunner := fmt.Sprintf("go test -shuffle=on %s-coverprofile=build/coverprofile.out", singleThreaded)
		if sr.UseGinkgo {
			testRunner = "go run github.com/onsi/ginkgo/v2/ginkgo run --randomize-all -output-dir=build"
		}
		goTest := fmt.Sprintf(`%s $(GO_BUILDFLAGS) -ldflags '%s $(GO_LDFLAGS)' -covermode=count -coverpkg=$(subst $(space),$(comma),$(GO_COVERPKGS)) $(GO_TESTFLAGS) $(GO_TESTPKGS)`,
			testRunner, makeDefaultLinkerFlags(path.Base(sr.ModulePath), sr))
		if runControllerGen {
			testRule.prerequisites = append(testRule.prerequisites, "generate", "install-setup-envtest")
			testRule.recipe = append(testRule.recipe, fmt.Sprintf(`KUBEBUILDER_ASSETS=$$(setup-envtest use %s -p path) %s`, sr.KubernetesVersion, goTest))
		} else {
			testRule.recipe = append(testRule.recipe, `@env $(GO_TESTENV) `+goTest)
		}
		// workaround for <https://github.com/fgrosse/go-coverage-report/issues/61>: merge block coverage manually
		testRule.recipe = append(testRule.recipe, `@awk < build/coverprofile.out '$$1 != "mode:" { is_filename[$$1] = true; counts1[$$1]+=$$2; counts2[$$1]+=$$3 } END { for (filename in is_filename) { printf "%s %d %d\n", filename, counts1[filename], counts2[filename]; } }' | sort | $(SED) '1s/^/mode: count\n/' > $@`)

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

	if cfg.License.AddHeaders.UnwrapOr(isSAPCC) {
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
		copyright := cfg.License.Copyright.UnwrapOr("SAP SE or an SAP affiliate company")

		dev.addRule(rule{
			description:   "Add (or overwrite) license headers on all non-vendored source code files.",
			target:        "license-headers",
			phony:         true,
			prerequisites: []string{"install-addlicense", "install-reuse"},
			recipe: []string{
				`@printf "\e[1;36m>> addlicense (for license headers on source code files)\e[0m\n"`,
				// We must use gawk to use gnu awk on Darwin
				fmt.Sprintf(`@printf "%%s\0" %s | $(XARGS) -0 -I{} bash -c '`+
					// Try to extract the copyright year
					`year="$$(grep 'Copyright' {} | head -n1 | grep -E -o '"'"'[0-9]{4}(-[0-9]{4})?'"'"')"; `+
					// If year is empty, set it to the current year
					`if [[ -z "$$year" ]]; then year=$$(date +%%Y); fi; `+
					// clean up old license headers
					`gawk -i inplace '"'"'{if (display) {print} else {!/^\/\*/ && !/^\*/}}; {if (!display && $$0 ~ /^(package |$$)/) {display=1} else { }}'"'"' {}; `+
					// Run addlicense tool, will be a no-op if the license header is already present
					`addlicense -c "%s" -s=only -y "$$year" %s {}; `+
					// Replace "// Copyright" with "// SPDX-FileCopyrightText:" to fulfill reuse
					`$(SED) -i '"'"'1s+// Copyright +// SPDX-FileCopyrightText: +'"'"' {}; `+
					`'`, allSourceFilesExpr, copyright, ignoreOptionsStr),
				`@printf "\e[1;36m>> reuse annotate (for license headers on other files)\e[0m\n"`,
				fmt.Sprintf(`@reuse lint -j | jq -r '.non_compliant.missing_licensing_info[]' | grep -vw vendor | $(XARGS) reuse annotate -c '%s' -l Apache-2.0 --skip-unrecognised`, copyright),
				`@printf "\e[1;36m>> reuse download --all\e[0m\n"`,
				`@reuse download --all`,
				`@printf "\e[1;35mPlease review the changes. If *.license files were generated, consider instructing go-makefile-maker to add overrides to REUSE.toml instead.\e[0m\n"`,
			},
		})

		test.addRule(rule{
			description:   "Check license headers in all non-vendored .go files with addlicense.",
			target:        "check-addlicense",
			phony:         true,
			prerequisites: []string{"install-addlicense"},
			recipe: []string{
				`@printf "\e[1;36m>> addlicense --check\e[0m\n"`,
				fmt.Sprintf(`@addlicense --check %s %s`, ignoreOptionsStr, allSourceFilesExpr),
			},
		})
		test.addRule(rule{
			description:   "Check reuse compliance",
			target:        "check-reuse",
			phony:         true,
			prerequisites: []string{"install-reuse"},
			recipe: []string{
				`@printf "\e[1;36m>> reuse lint\e[0m\n"`,
				// reuse is very verbose, so we only show the output if there are problems
				`@if ! reuse lint -q; then reuse lint; fi`,
			},
		})
		test.addRule(rule{
			description:   "Run static code checks",
			phony:         true,
			target:        "check-license-headers",
			prerequisites: []string{"check-addlicense", "check-reuse"},
		})

		if isGolang {
			must.Succeed(util.WriteFile(".editorconfig", editorconfig))

			licenseRulesFile := ".license-scan-rules.json"
			must.Succeed(util.WriteFile(licenseRulesFile, licenseRules))

			scanOverridesFile := ".license-scan-overrides.jsonl"
			must.Succeed(util.WriteFile(scanOverridesFile, scanOverrides))

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

	staticCheckPrerequisites := []string{"run-shellcheck"}
	if isGolang {
		// add target for static code checks
		staticCheckPrerequisites = append(staticCheckPrerequisites, "run-golangci-lint", "run-modernize")
		if cfg.License.CheckDependencies.UnwrapOr(isSAPCC) {
			staticCheckPrerequisites = append(staticCheckPrerequisites, "check-dependency-licenses")
		}

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

	if cfg.License.AddHeaders.UnwrapOr(isSAPCC) {
		staticCheckPrerequisites = append(staticCheckPrerequisites, "check-license-headers")
	}

	test.addRule(rule{
		description:   "Run static code checks (internal option to enforce --keep-going)",
		phony:         true,
		target:        "__static-check",
		hideTarget:    true,
		prerequisites: staticCheckPrerequisites,
	})
	test.addRule(rule{
		description: "Run static code checks",
		phony:       true,
		target:      "static-check",
		recipe:      []string{`@$(MAKE) --keep-going --no-print-directory __static-check`},
	})

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
