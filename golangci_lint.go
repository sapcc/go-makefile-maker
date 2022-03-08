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

package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

var configTmpl = template.Must(template.New("golangci").Parse(strings.TrimSpace(strings.ReplaceAll(`
run:
	deadline: 3m # 1m by default
	modules-download-mode: {{ .ModDownloadMode }}

output:
	# Do not print lines of code with issue.
	print-issued-lines: false

issues:
	# '0' disables the following options.
	max-issues-per-linter: 0
	max-same-issues: 0
	exclude:
		# It is idiomatic Go to reuse the name 'err' with ':=' for subsequent errors.
		# Ref: https://go.dev/doc/effective_go#redeclaration
		- 'declaration of "err" shadows declaration at'

linters-settings:
	dupl:
		# Tokens count to trigger issue, 150 by default.
		threshold: 100
	errcheck:
		# Report about assignment of errors to blank identifier.
		check-blank: true
		# Report about not checking of errors in type assertions.
		check-type-assertions: true
	forbidigo:
		forbid:
			# ioutil package has been deprecated: https://github.com/golang/go/issues/42026
			- ^ioutil\..*$
	goimports:
		# Put local imports after 3rd-party packages.
		local-prefixes: {{ .ModulePath }}
	govet:
		# Report about shadowed variables.
		check-shadowing: true
	{{- if .MisspellIgnoreWords }}
	misspell:
		ignore-words:
			{{- range .MisspellIgnoreWords }}
			- {{ . }}
			{{- end }}
	{{- end }}
	whitespace:
		# Enforce newlines (or comments) after multi-line function signatures.
		multi-func: true

linters:
	# We use 'disable-all' and enable linters explicitly so that a newer version
	# does not introduce new linters unexpectedly.
	disable-all: true
	enable:
		- deadcode
		- dupl
		- errcheck
		- exportloopref
		- forbidigo
		- gofmt
		- goimports
		- gosimple
		- govet
		- ineffassign
		- misspell
		- rowserrcheck
		- staticcheck
		- structcheck
		- stylecheck
		- typecheck
		- unconvert
		- unparam
		- unused
		- varcheck
		- whitespace
`, "\t", "  "))))

type configTmplData struct {
	ModulePath          string
	ModDownloadMode     string
	MisspellIgnoreWords []string
}

func renderGolangciLintConfig(cfg GolangciLintConfiguration, vendoring bool, modulePath string, misspellIgnoreWords []string) error {
	if !cfg.CreateConfig {
		return nil
	}

	f, err := os.Create(".golangci.yaml")
	if err != nil {
		return err
	}

	fmt.Fprintln(f, autogenHeader+"\n")
	mode := "readonly"
	if vendoring {
		mode = "vendor"
	}
	err = configTmpl.Execute(f, configTmplData{
		ModulePath:          modulePath,
		ModDownloadMode:     mode,
		MisspellIgnoreWords: misspellIgnoreWords,
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(f) // empty line at end

	return f.Close()
}
