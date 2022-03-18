// Copyright 2021 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ghworkflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

const workflowDir = ".github/workflows"

// Render renders GitHub workflows.
func Render(cfg *core.Configuration) error {
	ghwCfg := cfg.GitHubWorkflow

	err := os.MkdirAll(workflowDir, 0o755)
	if err == nil && ghwCfg.CI.Enabled {
		err = ciWorkflow(ghwCfg, cfg.Vendoring.Enabled)
	}
	if err == nil && ghwCfg.License.Enabled {
		err = licenseWorkflow(ghwCfg)
	}
	if err == nil && ghwCfg.SpellCheck.Enabled {
		err = spellCheckWorkflow(ghwCfg, cfg.SpellCheck.IgnoreWords)
	}
	if err == nil && ghwCfg.CodeQL.Enabled {
		err = codeQLWorkflow(ghwCfg)
	}
	if err != nil {
		return err
	}

	return nil
}

func writeWorkflowToFile(w *workflow) error {
	path := filepath.Join(workflowDir, strings.ToLower(w.Name)+".yaml")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	defer encoder.Close()
	encoder.SetIndent(2)

	fmt.Fprintln(f, core.AutogeneratedHeader)
	fmt.Fprintln(f, "")
	err = encoder.Encode(w)
	if err != nil {
		return err
	}

	return nil
}