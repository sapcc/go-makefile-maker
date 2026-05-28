// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"text/template"

	"github.com/sapcc/go-bits/logg"
	"go.yaml.in/yaml/v3"
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
		"sortedKeys": func(m map[string]string) []string {
			return slices.Sorted(maps.Keys(m))
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

	return WriteFile(fileName, buf.Bytes())
}

// WriteFile is like os.WriteFile, but it also writes a debug log about which file is being written.
func WriteFile(fileName string, contents []byte) error {
	logg.Debug("-> writing file %s", fileName)
	return os.WriteFile(fileName, contents, 0o666)
}

// RawString is a string type that marshals into a plain (unquoted) YAML scalar.
// If the string contains " # ", everything from " # " onwards is emitted as an inline YAML comment
// rather than as part of the scalar value.
type RawString string

// MarshalYAML implements yaml.Marshaler.
func (s RawString) MarshalYAML() (any, error) {
	value := string(s)
	var comment string
	if left, right, ok := strings.Cut(value, " # "); ok {
		comment = "# " + right
		value = strings.TrimSpace(left)
	}
	return &yaml.Node{
		Kind:        yaml.ScalarNode,
		Tag:         "!!str",
		Value:       value,
		LineComment: comment,
	}, nil
}
