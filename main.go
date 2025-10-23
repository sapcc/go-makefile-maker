// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sapcc/go-api-declarations/bininfo"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
	"github.com/spf13/pflag"
	"go.yaml.in/yaml/v3"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/dockerfile"
	"github.com/sapcc/go-makefile-maker/internal/ghworkflow"
	"github.com/sapcc/go-makefile-maker/internal/golang"
	"github.com/sapcc/go-makefile-maker/internal/golangcilint"
	"github.com/sapcc/go-makefile-maker/internal/goreleaser"
	"github.com/sapcc/go-makefile-maker/internal/makefile"
	"github.com/sapcc/go-makefile-maker/internal/nix"
	"github.com/sapcc/go-makefile-maker/internal/renovate"
	"github.com/sapcc/go-makefile-maker/internal/reuse"
)

func main() {
	bininfo.HandleVersionArgument()

	var flags struct {
		AutoupdateDeps bool
		ShowHelp       bool
	}
	pflag.BoolVar(&flags.AutoupdateDeps, "autoupdate-deps", false, "autoupdate dependencies matching the golang.autoupdateableDeps config option (if any)")
	pflag.BoolVar(&logg.ShowDebug, "debug", false, "print debug logs")
	pflag.BoolVar(&flags.ShowHelp, "help", false, "print this message")
	pflag.Parse()
	if flags.ShowHelp {
		fmt.Print("Usage of go-makefile-maker:\n", pflag.CommandLine.FlagUsages())
		return
	}

	logg.Debug("reading Makefile.maker.yaml")
	file := must.Return(os.Open("Makefile.maker.yaml"))
	var cfg core.Configuration
	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	must.Succeed(dec.Decode(&cfg))
	must.Succeed(file.Close())
	cfg.Validate()

	// The github.com/ prefix is just a safeguard to avoid false positives when the metadata.url is not complete.
	if cfg.GitHubWorkflow != nil && !strings.Contains(cfg.Metadata.URL, "github.com/") {
		cfg.GitHubWorkflow.IsSelfHostedRunner = true
		if strings.Contains(cfg.Metadata.URL, "/sap-cloud-infrastructure/") {
			cfg.GitHubWorkflow.IsSugarRunner = true
		}
	}

	if cfg.Golang.SetGoModVersion {
		logg.Debug("checking Go version in go.mod")
		golang.SetGoVersionInGoMod()
	}

	if fs, err := os.Stat("vendor/modules.txt"); err == nil && fs != nil {
		cfg.Golang.EnableVendoring = true
	}

	// Scan go.mod file for additional context information.
	logg.Debug("reading go.mod")
	sr := golang.Scan()

	if flags.AutoupdateDeps && cfg.Golang.AutoupdateableDepsRx != "" {
		logg.Debug("autoupdating library dependencies")
		golang.AutoupdateDependencies(sr, cfg.Golang)
	}

	logg.Debug("rendering configs for Nix")
	renderGoreleaserConfig := (cfg.GoReleaser.CreateConfig.IsNone() && cfg.GitHubWorkflow != nil && cfg.GitHubWorkflow.Release.Enabled.UnwrapOr(false)) || cfg.GoReleaser.ShouldCreateConfig()
	nix.RenderShell(cfg, sr, renderGoreleaserConfig)

	// Render Makefile
	if cfg.Makefile.Enabled.UnwrapOr(true) {
		logg.Debug("rendering Makefile")
		for _, bin := range cfg.Binaries {
			if !strings.HasPrefix(bin.FromPackage, ".") {
				logg.Fatal("binaries[].fromPackage must begin with a dot, %q is not allowed!", bin.FromPackage)
			}
		}
		makefile.Render(cfg, sr)
	}

	// Render Dockerfile
	if cfg.Dockerfile.Enabled {
		logg.Debug("rendering Dockerfile")
		dockerfile.RenderConfig(cfg)
	}

	// Render golangci-lint config file
	if cfg.GolangciLint.CreateConfig {
		logg.Debug("rendering golangci-lint configuration")
		golangcilint.RenderConfig(cfg, sr)
	}

	// Render Goreleaser config file
	if renderGoreleaserConfig {
		logg.Debug("rendering goreleaser configuration")
		goreleaser.RenderConfig(cfg)
	}

	// Render GitHub workflows
	if cfg.GitHubWorkflow != nil {
		logg.Debug("rendering GitHub Actions workflows")
		ghworkflow.Render(cfg, sr)
	}

	// Render Renovate config
	if cfg.Renovate.Enabled {
		logg.Debug("rendering Renovate configuration")
		if cfg.Renovate.GoVersion == "" {
			cfg.Renovate.GoVersion = sr.GoVersionMajorMinor
		}
		renovate.RenderConfig(cfg, sr)
	}

	// Render REUSE config file
	if cfg.Reuse.IsEnabled() {
		logg.Debug("rendering REUSE configuration")
		reuse.RenderConfig(cfg, sr)
	}
}
