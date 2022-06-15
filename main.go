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
	"errors"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/dockerfile"
	"github.com/sapcc/go-makefile-maker/internal/ghworkflow"
	"github.com/sapcc/go-makefile-maker/internal/golangcilint"
	"github.com/sapcc/go-makefile-maker/internal/makefile"
	"github.com/sapcc/go-makefile-maker/internal/renovate"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

func main() {
	file, err := os.Open("Makefile.maker.yaml")
	util.Must(err)

	var cfg core.Configuration
	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	util.Must(dec.Decode(&cfg))
	util.Must(file.Close())
	util.Must(cfg.Validate())

	// Scan go.mod file for additional context information.
	sr, err := core.Scan()
	util.Must(err)

	// Render Makefile.
	util.Must(makefile.Render(&cfg, sr))

	// Render Dockerfile
	if cfg.Dockerfile.Enabled {
		if cfg.Metadata.URL == "" {
			util.Must(errors.New("metadata.url needs to be set when docker.enabled is true"))
		}
		util.Must(dockerfile.RenderConfig(cfg))
	}

	// Render golangci-lint config file.
	if cfg.GolangciLint.CreateConfig {
		util.Must(golangcilint.RenderConfig(cfg.GolangciLint, cfg.Vendoring.Enabled, sr.MustModulePath(), cfg.SpellCheck.IgnoreWords))
	}

	// Render GitHub workflows.
	if cfg.GitHubWorkflow != nil {
		if cfg.GitHubWorkflow.Global.GoVersion == "" {
			if sr.GoVersion == "" {
				util.Must(errors.New("could not find Go version from go.mod file, consider defining manually by setting 'githubWorkflow.global.goVersion' in config"))
			}
			cfg.GitHubWorkflow.Global.GoVersion = sr.GoVersion
		}
		util.Must(ghworkflow.Render(&cfg))
	}

	// Render Renovate config.
	if cfg.Renovate.Enabled {
		if cfg.Renovate.GoVersion == "" {
			if sr.GoVersion == "" {
				util.Must(errors.New("could not find Go version from go.mod file, consider defining manually by setting 'renovate.goVersion' in config"))
			}
			cfg.Renovate.GoVersion = sr.GoVersion
		}
		isGoMakefileMakerRepo := sr.MustModulePath() == "github.com/sapcc/go-makefile-maker"
		util.Must(renovate.RenderConfig(cfg.GitHubWorkflow.Global.Assignees, cfg.Renovate.PackageRules, cfg.Renovate.GoVersion, sr.GoDirectDependencies, isGoMakefileMakerRepo))
	}
}
