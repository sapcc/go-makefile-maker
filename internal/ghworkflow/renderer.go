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

	"gopkg.in/yaml.v2"
)

// Render renders GitHub workflows.
func Render(cfg *Configuration) error {
	// Validate config
	cfg.Validate()

	// Render workflows
	err := os.MkdirAll(workflowDir, 0o755)
	if err == nil && cfg.CI.Enabled {
		err = ciWorkflow(cfg)
	}
	if err == nil && cfg.License.Enabled {
		err = licenseWorkflow(cfg)
	}
	if err == nil && cfg.SpellCheck.Enabled {
		err = spellCheckWorkflow(cfg)
	}
	if err != nil {
		return err
	}

	return nil
}

func writeWorkflowToFile(w *workflow) error {
	b, err := yaml.Marshal(w)
	if err != nil {
		return err
	}

	path := filepath.Join(workflowDir, strings.ToLower(w.Name)+".yaml")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, autogenHeader)
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, string(b))

	return nil
}
