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

package golangcilint

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

var configTmpl = template.Must(template.New("golangci").Parse(strings.TrimSpace(strings.ReplaceAll(`
run:
	deadline: 3m # 1m by default
	modules-download-mode: {{ .ModDownloadMode }}
	{{- if .SkipDirs }}
	skip-dirs:
		{{- range .SkipDirs }}
		- {{ . }}
		{{- end }}
	{{- end }}

output:
	# Do not print lines of code with issue.
	print-issued-lines: false

issues:
	exclude:
		# It is idiomatic Go to reuse the name 'err' with ':=' for subsequent errors.
		# Ref: https://go.dev/doc/effective_go#redeclaration
		- 'declaration of "err" shadows declaration at'
	exclude-rules:
		- path: _test\.go
			linters:
				- bodyclose
	# '0' disables the following options.
	max-issues-per-linter: 0
	max-same-issues: 0

linters-settings:
	dupl:
		# Tokens count to trigger issue, 150 by default.
		threshold: 100
	errcheck:
		# Report about assignment of errors to blank identifier.
		check-blank: true
		# Report about not checking of errors in type assertions.
		check-type-assertions: true
		{{- if .ErrcheckExcludes }}
		exclude-functions:
			{{- range .ErrcheckExcludes }}
			- {{ . }}
			{{- end }}
		{{- end }}
	forbidigo:
		forbid:
			# ioutil package has been deprecated: https://github.com/golang/go/issues/42026
			- ^ioutil\..*$
			# Using http.DefaultServeMux is discouraged because it's a global variable
			# that some packages silently and magically add handlers to (esp. net/http/pprof).
			# Applications wishing to use http.ServeMux should obtain local instances
			# through http.NewServeMux() instead of using the global default instance.
			- ^http.DefaultServeMux$
			- ^http.Handle(?:Func)?$
	gocritic:
		enabled-checks:
			- boolExprSimplify
			- builtinShadow
			- emptyStringTest
			- evalOrder
			- httpNoBody
			- importShadow
			- initClause
			- methodExprCall
			- paramTypeCombine
			- preferFilepathJoin
			- ptrToRefParam
			- redundantSprint
			- returnAfterHttpError
			- stringConcatSimplify
			- timeExprSimplify
			- truncateCmp
			- typeAssertChain
			- typeUnparen
			- unnamedResult
			- unnecessaryBlock
			- unnecessaryDefer
			- weakCond
			- yodaStyleExpr
	goimports:
		# Put local imports after 3rd-party packages.
		local-prefixes: {{ .ModulePath }}
	gosec:
		excludes:
			# gosec wants us to set a short ReadHeaderTimeout to avoid Slowloris attacks,
			# but doing so would expose us to Keep-Alive race conditions.
			# See: https://iximiuz.com/en/posts/reverse-proxy-http-keep-alive-and-502s/
			- G112
			# created file permissions are restricted by umask if necessary
			- G306
	govet:
		# Report about shadowed variables.
		check-shadowing: true
	nolintlint:
		require-specific: true
	{{- if .MisspellIgnoreWords }}
	misspell:
		ignore-words:
			{{- range .MisspellIgnoreWords }}
			- {{ . }}
			{{- end }}
	{{- end }}
	stylecheck:
		dot-import-whitelist:
			- github.com/onsi/ginkgo/v2
			- github.com/onsi/gomega
	usestdlibvars:
		constant-kind: true
		crypto-hash: true
		default-rpc-path: true
		http-method: true
		http-status-code: true
		os-dev-null: true
		rpc-default-path: true
		time-weekday: true
		time-month: true
		time-layout: true
		tls-signature-scheme: true
	whitespace:
		# Enforce newlines (or comments) after multi-line function signatures.
		multi-func: true

linters:
	# We use 'disable-all' and enable linters explicitly so that a newer version
	# does not introduce new linters unexpectedly.
	disable-all: true
	enable:
		- bodyclose
		- containedctx
		- dupl
		- dupword
		- durationcheck
		- errcheck
		- errorlint
		- exportloopref
		- forbidigo
		- ginkgolinter
		- gocheckcompilerdirectives
		- gocritic
		- gofmt
		- goimports
		- gosec
		- gosimple
		- govet
		- ineffassign
		- misspell
		- noctx
		- nolintlint
		- nosprintfhostport
		- rowserrcheck
		- sqlclosecheck
		- staticcheck
		- stylecheck
		- tenv
		- typecheck
		- unconvert
		- unparam
		- unused
		- usestdlibvars
		- whitespace
`, "\t", "  "))))

type configTmplData struct {
	ModulePath          string
	ModDownloadMode     string
	MisspellIgnoreWords []string
	ErrcheckExcludes    []string
	SkipDirs            []string
}

func RenderConfig(cfg core.GolangciLintConfiguration, vendoring bool, modulePath string, misspellIgnoreWords []string) {
	mode := "readonly"
	if vendoring {
		mode = "vendor"
	}

	f := must.Return(os.Create(".golangci.yaml"))
	fmt.Fprintln(f, core.AutogeneratedHeader+"\n")
	must.Succeed(configTmpl.Execute(f, configTmplData{
		ModulePath:          modulePath,
		ModDownloadMode:     mode,
		MisspellIgnoreWords: misspellIgnoreWords,
		ErrcheckExcludes:    cfg.ErrcheckExcludes,
		SkipDirs:            cfg.SkipDirs,
	}))
	fmt.Fprintln(f) // empty line at end

	must.Succeed(f.Close())
}
