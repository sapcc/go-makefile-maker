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
	Assignees         []string            `json:"assignees"`
	Constraints       renovateConstraints `json:"constraints"`
	PostUpdateOptions []string            `json:"postUpdateOptions"`
	PrHourlyLimit     int                 `json:"prHourlyLimit"`
}

func RenderConfig(assignees []string, goVersion string) error {
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
		PrHourlyLimit: 0,
	}
	if goVersion == "1.17" {
		config.PostUpdateOptions = append([]string{"gomodTidy1.17"}, config.PostUpdateOptions...)
	} else {
		config.PostUpdateOptions = append([]string{"gomodTidy"}, config.PostUpdateOptions...)
	}

	f, err := os.Create(".github/renovate.json")
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(config)
	if err != nil {
		return err
	}

	return f.Close()
}
