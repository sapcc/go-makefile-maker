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
	"fmt"
	"os"
	"strings"
	"text/template"
)

var configTmpl = template.Must(template.New("renovate").Parse(strings.TrimSpace(strings.ReplaceAll(`
{
	"extends": [
		"config:base",
		"default:pinDigestsDisabled",
		"docker:enableMajor",
		"regexManagers:dockerfileVersions"
	],
	"constraints": {
		"go": "{{ .GoVersion }}"
	},
	"postUpdateOptions": [
		"gomodTidy{{- if eq .GoVersion "1.17" }}1.17{{- end }}",
		"gomodUpdateImportPaths"
	],
	"prHourlyLimit": 0
}
`, "\t", "  "))))

type configTmplData struct {
	GoVersion string
}

func RenderConfig(goVersion string) error {
	f, err := os.Create(".github/renovate.json")
	if err != nil {
		return err
	}
	err = configTmpl.Execute(f, configTmplData{
		GoVersion: goVersion,
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(f) // empty line at end

	return f.Close()
}
