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

	"gopkg.in/yaml.v3"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/dockerfile"
	"github.com/sapcc/go-makefile-maker/internal/ghworkflow"
	"github.com/sapcc/go-makefile-maker/internal/golangcilint"
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

	// Scan go.mod file for additional context information.
	sr := core.Scan()

	// Render Makefile.
	makefile.Render(&cfg, sr)

	// Render Dockerfile
	if cfg.Dockerfile.Enabled {
		if cfg.Metadata.URL == "" {
			logg.Fatal("metadata.url needs to be set when docker.enabled is true")
		}
		dockerfile.RenderConfig(cfg)
	}

	// Render golangci-lint config file.
	if cfg.GolangciLint.CreateConfig {
		golangcilint.RenderConfig(cfg.GolangciLint, cfg.Vendoring.Enabled, sr.MustModulePath(), cfg.SpellCheck.IgnoreWords)
	}

	// Render GitHub workflows.
	if cfg.GitHubWorkflow != nil {
		if cfg.GitHubWorkflow.Global.GoVersion == "" {
			if sr.GoVersion == "" {
				logg.Fatal("could not find Go version from go.mod file, consider defining manually by setting 'githubWorkflow.global.goVersion' in config")
			}
			cfg.GitHubWorkflow.Global.GoVersion = sr.GoVersion
		}
		ghworkflow.Render(&cfg)
	}

	// Render Renovate config.
	if cfg.Renovate.Enabled {
		if cfg.Renovate.GoVersion == "" {
			if sr.GoVersion == "" {
				logg.Fatal("could not find Go version from go.mod file, consider defining manually by setting 'renovate.goVersion' in config")
			}
			cfg.Renovate.GoVersion = sr.GoVersion
		}
		isGoMakefileMakerRepo := sr.MustModulePath() == "github.com/sapcc/go-makefile-maker"
		renovate.RenderConfig(cfg.GitHubWorkflow.Global.Assignees, cfg.Renovate.PackageRules, cfg.Renovate.GoVersion, sr.GoDirectDependencies, isGoMakefileMakerRepo)
	}
}
