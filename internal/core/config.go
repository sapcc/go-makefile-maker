// SPDX-FileCopyrightText: Copyright 2020 SAP SE
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"os/exec"
	"strings"
	"time"

	"github.com/sapcc/go-bits/logg"
)

var AutogeneratedHeader = strings.TrimSpace(`
################################################################################
# This file is AUTOGENERATED with <https://github.com/sapcc/go-makefile-maker> #
# Edit Makefile.maker.yaml instead.                                            #
################################################################################

# SPDX-FileCopyrightText: Copyright 2024 SAP SE
# SPDX-License-Identifier: Apache-2.0
`)

///////////////////////////////////////////////////////////////////////////////
// Core configuration

// Configuration is the data structure that we read from the input file.
type Configuration struct {
	Binaries       []BinaryConfiguration        `yaml:"binaries"`
	Coverage       CoverageConfiguration        `yaml:"coverageTest"`
	ControllerGen  ControllerGen                `yaml:"controllerGen"`
	Dockerfile     DockerfileConfig             `yaml:"dockerfile"`
	GitHubWorkflow *GithubWorkflowConfiguration `yaml:"githubWorkflow"`
	Golang         GolangConfiguration          `yaml:"golang"`
	GolangciLint   GolangciLintConfiguration    `yaml:"golangciLint"`
	GoReleaser     GoReleaserConfiguration      `yaml:"goReleaser"`
	Makefile       MakefileConfig               `yaml:"makefile"`
	Metadata       Metadata                     `yaml:"metadata"`
	Nix            NixConfig                    `yaml:"nix"`
	Renovate       RenovateConfig               `yaml:"renovate"`
	SpellCheck     SpellCheckConfiguration      `yaml:"spellCheck"`
	Test           TestConfiguration            `yaml:"testPackages"`
	Reuse          ReuseConfiguration           `yaml:"reuse"`
	Verbatim       string                       `yaml:"verbatim"`
	VariableValues map[string]string            `yaml:"variables"`
}

// Variable returns the value of this variable if it's overridden in the config,
// or the default value otherwise.
func (c Configuration) Variable(name, defaultValue string) string {
	value, exists := c.VariableValues[name]
	if exists {
		return " " + value
	}
	if defaultValue == "" {
		return ""
	}
	return " " + defaultValue
}

// BinaryConfiguration appears in type Configuration.
type BinaryConfiguration struct {
	Name        string `yaml:"name"`
	FromPackage string `yaml:"fromPackage"`
	InstallTo   string `yaml:"installTo"`
}

// TestConfiguration appears in type Configuration.
type TestConfiguration struct {
	Only   string `yaml:"only"`
	Except string `yaml:"except"`
}

// ReuseConfiguration appears in type Configuration.
type ReuseConfiguration struct {
	Enabled *bool `yaml:"enabled"`
}

// CoverageConfiguration appears in type Configuration.
type CoverageConfiguration struct {
	Only   string `yaml:"only"`
	Except string `yaml:"except"`
}

// GolangConfiguration appears in type Configuration.
type GolangConfiguration struct {
	EnableVendoring bool              `yaml:"enableVendoring"`
	LdFlags         map[string]string `yaml:"ldflags"`
	SetGoModVersion bool              `yaml:"setGoModVersion"`
}

// GolangciLintConfiguration appears in type Configuration.
type GolangciLintConfiguration struct {
	CreateConfig     bool          `yaml:"createConfig"`
	ErrcheckExcludes []string      `yaml:"errcheckExcludes"`
	SkipDirs         []string      `yaml:"skipDirs"`
	Timeout          time.Duration `yaml:"timeout"`
}

type GoReleaserConfiguration struct {
	CreateConfig *bool             `yaml:"createConfig"`
	BinaryName   string            `yaml:"binaryName"`
	Files        *[]string         `yaml:"files"`
	Format       string            `yaml:"format"`
	Ldflags      map[string]string `yaml:"ldflags"`
	NameTemplate string            `yaml:"nameTemplate"`
}

// SpellCheckConfiguration appears in type Configuration.
type SpellCheckConfiguration struct {
	IgnoreWords []string `yaml:"ignoreWords"`
}

///////////////////////////////////////////////////////////////////////////////
// GitHub workflow configuration

// GithubWorkflowConfiguration appears in type Configuration.
type GithubWorkflowConfiguration struct {
	// These global-level settings are applicable for all workflows. They are
	// superseded by their workflow-level counterpart(s).
	Global struct {
		DefaultBranch string `yaml:"defaultBranch"`
		GoVersion     string `yaml:"goVersion"`
	} `yaml:"global"`

	CI                  CIWorkflowConfig             `yaml:"ci"`
	IsSelfHostedRunner  bool                         `yaml:"omit"`
	License             LicenseWorkflowConfig        `yaml:"license"`
	PushContainerToGhcr PushContainerToGhcrConfig    `yaml:"pushContainerToGhcr"`
	Release             ReleaseWorkflowConfig        `yaml:"release"`
	SecurityChecks      SecurityChecksWorkflowConfig `yaml:"securityChecks"`
}

// CIWorkflowConfig appears in type Configuration.
type CIWorkflowConfig struct {
	Enabled           bool     `yaml:"enabled"`
	Coveralls         bool     `yaml:"coveralls"`
	PrepareMakeTarget string   `yaml:"prepareMakeTarget"`
	IgnorePaths       []string `yaml:"ignorePaths"`
	RunnerType        []string `yaml:"runOn"`
}

// LicenseWorkflowConfig appears in type Configuration.
type LicenseWorkflowConfig struct {
	Enabled        *bool    `yaml:"enabled"`
	IgnorePatterns []string `yaml:"ignorePatterns"`
}

type PushContainerToGhcrConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Platforms   string   `yaml:"platforms"`
	TagStrategy []string `yaml:"tagStrategy"`
}

type ReleaseWorkflowConfig struct {
	Enabled bool `yaml:"enabled"`
}

// SecurityChecksWorkflowConfig appears in type Configuration.
type SecurityChecksWorkflowConfig struct {
	Enabled *bool `yaml:"enabled"`
}

type PackageRule struct {
	MatchPackageNames []string `yaml:"matchPackageNames" json:"matchPackageNames,omitempty"`
	MatchUpdateTypes  []string `yaml:"matchUpdateTypes" json:"matchUpdateTypes,omitempty"`
	MatchDepTypes     []string `yaml:"matchDepTypes" json:"matchDepTypes,omitempty"`
	MatchFileNames    []string `yaml:"matchFileNames" json:"matchFileNames,omitempty"`
	Extends           []string `yaml:"extends" json:"extends,omitempty"`
	AllowedVersions   string   `yaml:"allowedVersions" json:"allowedVersions,omitempty"`
	AutoMerge         bool     `yaml:"automerge" json:"automerge,omitempty"`
	EnableRenovate    *bool    `yaml:"enabled" json:"enabled,omitempty"`
	GroupName         string   `yaml:"groupName" json:"groupName,omitempty"`
	MinimumReleaseAge string   `yaml:"minimumReleaseAge" json:"minimumReleaseAge,omitempty"`
}

// RenovateConfig appears in type Configuration.
type RenovateConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Assignees      []string      `yaml:"assignees"`
	GoVersion      string        `yaml:"goVersion"`
	PackageRules   []PackageRule `yaml:"packageRules"`
	CustomManagers []interface{} `yaml:"customManagers"`
}

// DockerfileConfig appears in type Configuration.
type DockerfileConfig struct {
	Enabled          bool     `yaml:"enabled"`
	Entrypoint       []string `yaml:"entrypoint"`
	ExtraBuildStages []string `yaml:"extraBuildStages"`
	ExtraDirectives  []string `yaml:"extraDirectives"`
	ExtraIgnores     []string `yaml:"extraIgnores"`
	ExtraPackages    []string `yaml:"extraPackages"`
	RunAsRoot        bool     `yaml:"runAsRoot"`
	WithLinkerdAwait bool     `yaml:"withLinkerdAwait"`
}

type ControllerGen struct {
	Enabled          *bool  `yaml:"enabled"`
	CrdOutputPath    string `yaml:"crdOutputPath"`
	ObjectHeaderFile string `yaml:"objectHeaderFile"`
	RBACRoleName     string `yaml:"rbacRoleName"`
}

type MakefileConfig struct {
	Enabled *bool `yaml:"enabled"` // this is a pointer to bool to treat an absence as true for backwards compatibility
}

type Metadata struct {
	URL string `yaml:"url"`
}

type NixConfig struct {
	ExtraLibraries []string `yaml:"extraLibraries"`
	ExtraPackages  []string `yaml:"extraPackages"`
}

///////////////////////////////////////////////////////////////////////////////
// Helper functions

func (c *Configuration) Validate() {
	if c.Dockerfile.Enabled {
		if c.Metadata.URL == "" {
			logg.Fatal("metadata.url must be set when docker.enabled is true")
		}
	}

	// Validate GolangciLintConfiguration.
	if len(c.GolangciLint.ErrcheckExcludes) > 0 && !c.GolangciLint.CreateConfig {
		logg.Fatal("golangciLint.createConfig must be set to 'true' if golangciLint.errcheckExcludes is defined")
	}

	// Validate GithubWorkflowConfiguration.
	ghwCfg := c.GitHubWorkflow
	if ghwCfg != nil {
		if c.Metadata.URL == "" {
			logg.Fatal("metadata.url must be set when any github workflow is configured otherwise it cannot be determined which github runner type should be used")
		}

		// Validate global options.
		if ghwCfg.Global.DefaultBranch == "" {
			errMsg := "could not find default branch using git, you can define it manually by setting 'githubWorkflow.global.defaultBranch' in config"
			b, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").CombinedOutput()
			if err != nil {
				logg.Fatal("%s: %s", errMsg, err.Error())
			}

			branch := strings.TrimPrefix(string(b), "refs/remotes/origin/")
			if branch == string(b) {
				logg.Fatal(errMsg)
			} else {
				c.GitHubWorkflow.Global.DefaultBranch = strings.TrimSpace(branch)
			}
		}

		// Validate CI workflow configuration.
		if ghwCfg.CI.Enabled {
			if len(ghwCfg.CI.RunnerType) > 1 && !strings.HasPrefix(ghwCfg.CI.RunnerType[0], "ubuntu") {
				logg.Fatal("githubWorkflow.ci.runOn must only define a single Ubuntu based runner when githubWorkflow.ci.enabled is true")
			}
		}
	}
}
