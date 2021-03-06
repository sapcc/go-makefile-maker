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
	"fmt"
	"os"

	"github.com/sapcc/go-makefile-maker/internal/ghworkflow"
	"gopkg.in/yaml.v2"
)

func main() {
	file, err := os.Open("Makefile.maker.yaml")
	must(err)

	var cfg Configuration
	must(yaml.NewDecoder(file).Decode(&cfg))
	must(file.Close())

	if len(cfg.Binaries) == 0 {
		must(fmt.Errorf("Makefile.maker.yaml does not reference any binaries"))
	}

	file, err = os.Create("Makefile")
	must(err)
	r := Renderer{out: file}
	r.Render(cfg)
	must(file.Close())

	if cfg.GitHubWorkflows != nil {
		cfg.GitHubWorkflows.Vendoring = cfg.Vendoring.Enabled
		cfg.GitHubWorkflows.GolangciLint = cfg.StaticCheck.GolangciLint
		err := ghworkflow.Render(cfg.GitHubWorkflows)
		must(err)
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
