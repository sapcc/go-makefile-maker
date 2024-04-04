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
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/dockerfile"
	"github.com/sapcc/go-makefile-maker/internal/ghworkflow"
	"github.com/sapcc/go-makefile-maker/internal/golangcilint"
	"github.com/sapcc/go-makefile-maker/internal/goreleaser"
	"github.com/sapcc/go-makefile-maker/internal/makefile"
	"github.com/sapcc/go-makefile-maker/internal/renovate"
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
		modFileBytes := must.Return(os.ReadFile(core.ModFilename))
		rgx := regexp.MustCompile(`go \d\.\d+(\.\d+)?`)
		goVersionSlice := strings.Split(core.DefaultGoVersion, ".")
		modFileBytesReplaced := rgx.ReplaceAll(modFileBytes, []byte("go "+strings.Join(goVersionSlice[:len(goVersionSlice)-1], ".")))
		must.Succeed(os.WriteFile(core.ModFilename, modFileBytesReplaced, 0o666))
	}

	if fs, err := os.Stat("vendor/modules.txt"); err == nil && fs != nil {
		cfg.Golang.EnableVendoring = true
	}

	// Scan go.mod file for additional context information.
	sr := core.Scan()

	// Render Makefile
	if cfg.Makefile.Enabled == nil || *cfg.Makefile.Enabled {
		makefile.Render(&cfg, sr)
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
	if cfg.GoReleaser.CreateConfig {
		goreleaser.RenderConfig(cfg)
	}

	// Render GitHub workflows
	if cfg.GitHubWorkflow != nil {
		if cfg.GitHubWorkflow.Global.GoVersion == "" {
			if sr.GoVersion == "" {
				logg.Fatal("could not find Go version from go.mod file, consider defining manually by setting 'githubWorkflow.global.goVersion' in config")
			}
			cfg.GitHubWorkflow.Global.GoVersion = sr.GoVersion
		}
		ghworkflow.Render(cfg)
	}

	// Render Renovate config
	if cfg.Renovate.Enabled {
		if cfg.Renovate.GoVersion == "" {
			if sr.GoVersion == "" {
				logg.Fatal("could not find Go version from go.mod file, consider defining manually by setting 'renovate.goVersion' in config")
			}
			cfg.Renovate.GoVersion = sr.GoVersion
		}
		isApplicationRepo := len(cfg.Binaries) > 0
		renovate.RenderConfig(cfg.Renovate, sr, cfg.Metadata.URL, isApplicationRepo)
	}
}
