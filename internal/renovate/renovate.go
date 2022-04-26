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
	"os"
)

type renovateConstraints struct {
	Go string `json:"go"`
}

type renovateConfig struct {
	Extends           []string            `json:"extends"`
	GitHubActions     *githubActions      `json:"github-actions,omitempty"`
	Assignees         []string            `json:"assignees,omitempty"`
	Constraints       renovateConstraints `json:"constraints"`
	PostUpdateOptions []string            `json:"postUpdateOptions"`
	PackageRules      []packageRule       `json:"packageRules,omitempty"`
	PrHourlyLimit     int                 `json:"prHourlyLimit"`
}

type githubActions struct {
	Enabled bool `json:"enabled"`
}

type packageRule struct {
	EnableRenovate       bool     `json:"enabled"`
	MatchPackagePrefixes []string `json:"matchPackagePrefixes,omitempty"`
	AllowedVersions      string   `json:"allowedVersions,omitempty"`
}

func RenderConfig(assignees []string, goVersion string, enableGHActions bool) error {
	config := renovateConfig{
		Extends: []string{
			"config:base",
			"default:pinDigestsDisabled",
			"docker:enableMajor",
			"regexManagers:dockerfileVersions",
		},
		Assignees: assignees,
		Constraints: renovateConstraints{
			Go: goVersion,
		},
		PostUpdateOptions: []string{
			"gomodUpdateImportPaths",
		},
		PackageRules: []packageRule{{
			EnableRenovate:       false,
			MatchPackagePrefixes: []string{"k8s.io/"},
			// Since our clusters use k8s v1.22 therefore we set the allowedVersions to `<0.23`.
			// k8s.io/* deps use v0.x.y instead of v1.x.y therefore we use 0.23 instead of 1.23.
			// Ref: https://docs.renovatebot.com/configuration-options/#allowedversions
			AllowedVersions: "<0.23",
		}},
		PrHourlyLimit: 0,
	}
	if goVersion == "1.17" {
		config.PostUpdateOptions = append([]string{"gomodTidy1.17"}, config.PostUpdateOptions...)
	} else {
		config.PostUpdateOptions = append([]string{"gomodTidy"}, config.PostUpdateOptions...)
	}
	if !enableGHActions {
		// By default, Renovate is enabled for GitHub Actions so we need to disable it
		// here manually in case it is not required.
		config.GitHubActions = &githubActions{Enabled: enableGHActions}
	}

	f, err := os.Create(".github/renovate.json")
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // in order to preserve `<` in allowedVersions field
	err = encoder.Encode(config)
	if err != nil {
		return err
	}

	return f.Close()
}
