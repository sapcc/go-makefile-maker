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
	"fmt"
	"os"

	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v3"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/ghworkflow"
	"github.com/sapcc/go-makefile-maker/internal/golangcilint"
	"github.com/sapcc/go-makefile-maker/internal/makefile"
)

func main() {
	file, err := os.Open("Makefile.maker.yaml")
	must(err)

	var cfg core.Configuration
	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	must(dec.Decode(&cfg))
	must(file.Close())
	must(cfg.Validate())

	// Render Makefile.
	must(makefile.Render(&cfg))

	// Read go.mod file for module path and Go version.
	modFilename := "go.mod"
	modFileBytes, err := os.ReadFile(modFilename)
	must(err)
	modFile, err := modfile.Parse(modFilename, modFileBytes, nil)
	must(err)

	// Render golangci-lint config file.
	if cfg.GolangciLint.CreateConfig {
		if modFile.Module.Mod.Path == "" {
			must(errors.New("could not find module path from go.mod file, make sure it is defined"))
		}
		must(golangcilint.RenderConfig(cfg.GolangciLint, cfg.Vendoring.Enabled, modFile.Module.Mod.Path, cfg.SpellCheck.IgnoreWords))
	}

	// Render GitHub workflows.
	if cfg.GitHubWorkflow != nil {
		if cfg.GitHubWorkflow.Global.GoVersion == "" {
			if modFile.Go.Version == "" {
				must(errors.New("could not find Go version from go.mod file, consider defining manually by setting 'githubWorkflow.global.goVersion' in config"))
			}
			cfg.GitHubWorkflow.Global.GoVersion = modFile.Go.Version
		}
		must(ghworkflow.Render(&cfg))
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL:", err.Error())
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "HINT: Did you run this in the root directory of a suitable Git repository?")
		}
		os.Exit(1)
	}
}
