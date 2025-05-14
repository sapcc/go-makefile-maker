// SPDX-FileCopyrightText: 2025 SAP SE
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
)

// WriteFileFromTemplate generates and writes the contents of `fileName` by
// loading `templateCode` using text/template and executing it with `data`.
//
// The `templateCode` usually lives in a *.tmpl file next to the source code
// calling this function, and is compiled into the binary using `package embed`.
func WriteFileFromTemplate(fileName, templateCode string, data map[string]any) error {
	funcMap := template.FuncMap{
		"containsIgnoreCase": func(s, substr string) bool {
			return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
		},
		"trimSpace": strings.TrimSpace,
	}
	t, err := template.New(fileName).Funcs(funcMap).Parse(templateCode)
	if err != nil {
		return fmt.Errorf("could not load template for %s: %w", fileName, err)
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("could not render %s: %w", fileName, err)
	}
	return os.WriteFile(fileName, buf.Bytes(), 0o666)
}
