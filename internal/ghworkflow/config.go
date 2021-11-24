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
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
)

// Configuration appears in type main.Configuration.
type Configuration struct {
	// These global-level settings are applicable for all workflows. They are
	// superseded by their workflow-level counterpart(s).
	Global struct {
		CommonConfig `yaml:",inline"`

		DefaultBranch string `yaml:"defaultBranch"`
		GoVersion     string `yaml:"goVersion"`
	} `yaml:"global"`

	CI         CIConfig         `yaml:"ci"`
	License    LicenseConfig    `yaml:"license"`
	SpellCheck SpellCheckConfig `yaml:"spellCheck"`

	// These settings are inherited from main.Configuration.
	Vendoring    bool
	GolangciLint bool
}

// CIConfig appears in type Configuration.
type CIConfig struct {
	CommonConfig `yaml:",inline"`

	Enabled      bool     `yaml:"enabled"`
	RunnerOSList []string `yaml:"runOn"`
	Coveralls    bool     `yaml:"coveralls"`
	Postgres     struct {
		Enabled bool   `yaml:"enabled"`
		Version string `yaml:"version"`
	} `yaml:"postgres"`
}

// LicenseConfig appears in type Configuration.
type LicenseConfig struct {
	CommonConfig `yaml:",inline"`

	Enabled  bool     `yaml:"enabled"`
	Patterns []string `yaml:"patterns"`
}

// SpellCheckConfig appears in type Configuration.
type SpellCheckConfig struct {
	CommonConfig `yaml:",inline"`

	Enabled     bool     `yaml:"enabled"`
	IgnoreWords []string `yaml:"ignoreWords"`
}

// CommonConfig holds common configuration options that are applicable for all
// workflows.
type CommonConfig struct {
	IgnorePaths []string `yaml:"ignorePaths"`
}

// Validate validates and sets defaults for global options for Configuration.
func (c *Configuration) Validate() {
	if !(c.CI.Enabled || c.License.Enabled || c.SpellCheck.Enabled) {
		printErrAndExit("no GitHub workflow enabled. See README for workflow configuration")
	}
	if c.CI.Postgres.Enabled && len(c.CI.RunnerOSList) >= 1 {
		if len(c.CI.RunnerOSList) > 1 || !strings.HasPrefix(c.CI.RunnerOSList[0], "ubuntu") {
			printErrAndExit("githubWorkflows.ci.runOn must have a single Ubuntu based runner")
		}
	}

	if c.Global.DefaultBranch == "" {
		found := false
		b, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").CombinedOutput()
		if err == nil {
			branch := strings.TrimPrefix(string(b), "refs/remotes/origin/")
			if branch != string(b) {
				found = true
				c.Global.DefaultBranch = strings.TrimSpace(branch)
			}
		}
		if !found {
			printErrAndExit("could not find default branch using git, consider defining 'githubWorkflows.defaultBranch' in config")
		}
		if err != nil {
			printErrAndExit(err.Error())
		}
	}

	if c.Global.GoVersion == "" {
		filename := "go.mod"
		data, err := os.ReadFile(filename)
		if err != nil {
			printErrAndExit(err.Error())
		}
		f, err := modfile.Parse(filename, data, nil)
		if err != nil {
			printErrAndExit(err.Error())
		}
		c.Global.GoVersion = f.Go.Version
		if c.Global.GoVersion == "" {
			printErrAndExit("could not find Go version from go.mod file, consider defining manually by setting githubWorkflows.global.goVersion")
		}
	}
}

func printErrAndExit(msg string) {
	fmt.Fprintf(os.Stderr, "FATAL: %s\n", msg)
	os.Exit(1)
}
