// SPDX-FileCopyrightText: Copyright 2020 SAP SE
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

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
	file := must.Return(os.Open("Makefile.maker.yaml"))

	var cfg core.Configuration
	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	must.Succeed(dec.Decode(&cfg))
	must.Succeed(file.Close())
	cfg.Validate()

	if cfg.GitHubWorkflow != nil && !strings.HasPrefix(cfg.Metadata.URL, "https://github.com") {
		cfg.GitHubWorkflow.IsSelfHostedRunner = true
	}

	if cfg.Golang.SetGoModVersion {
		golang.SetGoVersionInGoMod()
	}

	if fs, err := os.Stat("vendor/modules.txt"); err == nil && fs != nil {
		cfg.Golang.EnableVendoring = true
	}

	// Scan go.mod file for additional context information.
	sr := golang.Scan()

	renderGoreleaserConfig := (cfg.GoReleaser.CreateConfig == nil && cfg.GitHubWorkflow != nil && cfg.GitHubWorkflow.Release.Enabled) || (cfg.GoReleaser.CreateConfig != nil && *cfg.GoReleaser.CreateConfig)

	nix.RenderShell(cfg, sr, renderGoreleaserConfig)

	// Render Makefile
	if cfg.Makefile.Enabled == nil || *cfg.Makefile.Enabled {
		for _, bin := range cfg.Binaries {
			if !strings.HasPrefix(bin.FromPackage, ".") {
				logg.Fatal("binaries[].fromPackage must begin with a dot, %q is not allowed!", bin.FromPackage)
			}
		}
		makefile.Render(cfg, sr)
	}

	// Render Dockerfile
	if cfg.Dockerfile.Enabled {
		dockerfile.RenderConfig(cfg)
	}

	// Render golangci-lint config file
	if cfg.GolangciLint.CreateConfig {
		golangcilint.RenderConfig(cfg, sr)
	}

	// Render Goreleaser config file
	if renderGoreleaserConfig {
		goreleaser.RenderConfig(cfg)
	}

	// Render GitHub workflows
	if cfg.GitHubWorkflow != nil {
		// consider different fallbacks when no explicit go version is set
		if cfg.GitHubWorkflow.Global.GoVersion == "" {
			// default to the version in go.mod
			goVersion := sr.GoVersion

			// overwrite it, we want to use the latest go version
			if cfg.Golang.SetGoModVersion {
				goVersion = core.DefaultGoVersion
			}

			cfg.GitHubWorkflow.Global.GoVersion = goVersion
		}
		ghworkflow.Render(cfg, sr)
	}

	// Render Renovate config
	if cfg.Renovate.Enabled {
		if cfg.Renovate.GoVersion == "" {
			cfg.Renovate.GoVersion = sr.GoVersionMajorMinor
		}
		// TODO: checking on GoVersion is only an aid until we can properly detect rust applications
		isApplicationRepo := sr.GoVersion == "" || len(cfg.Binaries) > 0
		renovate.RenderConfig(cfg.Renovate, sr, cfg.Metadata.URL, isApplicationRepo)
	}

	// Render REUSE config file
	if cfg.Reuse.IsEnabled() {
		reuse.RenderConfig(cfg, sr)
	}
}
