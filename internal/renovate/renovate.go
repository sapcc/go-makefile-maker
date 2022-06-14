// Copyright 2022 SAP SE
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

package renovate

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/module"
)

type constraints struct {
	Go string `json:"go"`
}

type config struct {
	Extends           []string      `json:"extends"`
	Assignees         []string      `json:"assignees,omitempty"`
	Constraints       constraints   `json:"constraints"`
	PostUpdateOptions []string      `json:"postUpdateOptions"`
	PackageRules      []PackageRule `json:"packageRules,omitempty"`
	PrHourlyLimit     int           `json:"prHourlyLimit"`
	SemanticCommits   string        `json:"semanticCommits,omitempty"`
}

type PackageRule struct {
	MatchPackageNames    []string `yaml:"matchPackageNames" json:"matchPackageNames,omitempty"`
	MatchPackagePrefixes []string `yaml:"matchPackagePrefixes" json:"matchPackagePrefixes,omitempty"`
	MatchUpdateTypes     []string `yaml:"matchUpdateTypes" json:"matchUpdateTypes,omitempty"`
	MatchDepTypes        []string `yaml:"matchDepTypes" json:"matchDepTypes,omitempty"`
	MatchFiles           []string `yaml:"matchFiles" json:"matchFiles,omitempty"`
	AllowedVersions      string   `yaml:"allowedVersions" json:"allowedVersions,omitempty"`
	AutoMerge            bool     `yaml:"automerge" json:"automerge,omitempty"`
	EnableRenovate       *bool    `yaml:"enabled" json:"enabled,omitempty"`
}

func (c *config) addPackageRule(rule PackageRule) {
	c.PackageRules = append(c.PackageRules, rule)
}

func RenderConfig(
	assignees []string, customPackageRules []PackageRule,
	goVersion string, goDeps []module.Version,
	isGoMakefileMakerRepo bool) error {

	cfg := config{
		Extends: []string{
			"config:base",
			"default:pinDigestsDisabled",
		},
		Assignees: assignees,
		Constraints: constraints{
			Go: goVersion,
		},
		PostUpdateOptions: []string{
			"gomodUpdateImportPaths",
		},
		PrHourlyLimit:   0,
		SemanticCommits: "disabled",
	}
	if goVersion == "1.17" {
		cfg.PostUpdateOptions = append([]string{"gomodTidy1.17"}, cfg.PostUpdateOptions...)
	} else {
		cfg.PostUpdateOptions = append([]string{"gomodTidy"}, cfg.PostUpdateOptions...)
	}

	// Only enable Dockerfile and github-actions updates for go-makefile-maker itself.
	if isGoMakefileMakerRepo {
		cfg.Extends = append(cfg.Extends, "docker:enableMajor", "regexManagers:dockerfileVersions")
	} else {
		cfg.addPackageRule(PackageRule{
			MatchDepTypes:  []string{"action"},
			EnableRenovate: &isGoMakefileMakerRepo,
		})
		cfg.addPackageRule(PackageRule{
			MatchDepTypes:  []string{"dockerfile"},
			EnableRenovate: &isGoMakefileMakerRepo,
		})
	}

	// Default package rules.
	// NOTE: When changing this list, please also adjust the documentation for
	// default package rules in the README.
	cfg.addPackageRule(PackageRule{
		MatchPackageNames: []string{"golang"},
		AllowedVersions:   fmt.Sprintf("%s.x", goVersion),
	})
	hasK8sIOPkgs := false
	var autoMergePkgs []string
	for _, v := range goDeps {
		switch dep := v.Path; {
		case strings.HasPrefix(dep, "k8s.io/"):
			hasK8sIOPkgs = true
		case dep == "github.com/sapcc/go-api-declarations":
			autoMergePkgs = append(autoMergePkgs, "github.com/sapcc/go-api-declarations")
		case dep == "github.com/sapcc/gophercloud-sapcc":
			autoMergePkgs = append(autoMergePkgs, "github.com/sapcc/gophercloud-sapcc")
		case dep == "github.com/sapcc/go-bits":
			autoMergePkgs = append(autoMergePkgs, "github.com/sapcc/go-bits")
		}
	}
	if hasK8sIOPkgs {
		cfg.addPackageRule(PackageRule{
			MatchPackagePrefixes: []string{"k8s.io/"},
			// Since our clusters use k8s v1.22 therefore we set the allowedVersions to `0.22.x`.
			// k8s.io/* deps use v0.x.y instead of v1.x.y therefore we use 0.22 instead of 1.22.
			// Ref: https://docs.renovatebot.com/configuration-options/#allowedversions
			AllowedVersions: "0.22.x",
		})
	}
	if len(autoMergePkgs) > 0 {
		cfg.addPackageRule(PackageRule{
			MatchPackagePrefixes: autoMergePkgs,
			AutoMerge:            true,
		})
	}

	// Renovate will evaluate all packageRules and not stop once it gets a first match
	// therefore the packageRules should be in the order of importance so that user
	// defined rules can override settings from earlier rules.
	for _, rule := range customPackageRules {
		cfg.addPackageRule(rule)
	}

	f, err := os.Create(".github/renovate.json")
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // in order to preserve `<` in allowedVersions field
	err = encoder.Encode(cfg)
	if err != nil {
		return err
	}

	return f.Close()
}
