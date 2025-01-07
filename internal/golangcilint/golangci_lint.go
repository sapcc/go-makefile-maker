// Copyright 2021 SAP SE
// SPDX-License-Identifier: Apache-2.0

package golangcilint

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

var configTmpl = template.Must(template.New("golangci").Parse(strings.TrimSpace(strings.ReplaceAll(`
run:
	timeout: 3m # 1m by default
	modules-download-mode: {{ .ModDownloadMode }}

output:
	# Do not print lines of code with issue.
	print-issued-lines: false

issues:
	exclude:
		# It is idiomatic Go to reuse the name 'err' with ':=' for subsequent errors.
		# Ref: https://go.dev/doc/effective_go#redeclaration
		- 'declaration of "err" shadows declaration at'
	{{- if .SkipDirs }}
	exclude-dirs:
		{{- range .SkipDirs }}
		- {{ . }}
		{{- end }}
	{{- end }}
	exclude-rules:
		- path: _test\.go
			linters:
				- bodyclose
				- dupl
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
		# Do not report about not checking of errors in type assertions.
		# This is not as dangerous as skipping error values because an unchecked type assertion just immediately panics.
		# We disable this because it makes a ton of useless noise esp. in test code.
		check-type-assertions: false
		{{- if .ErrcheckExcludes }}
		exclude-functions:
			{{- range .ErrcheckExcludes }}
			- {{ . }}
			{{- end }}
		{{- end }}
	forbidigo:
		analyze-types: true # required for pkg:
		forbid:
			# ioutil package has been deprecated: https://github.com/golang/go/issues/42026
			- ^ioutil\..*$
			# Using http.DefaultServeMux is discouraged because it's a global variable that some packages silently and magically add handlers to (esp. net/http/pprof).
			# Applications wishing to use http.ServeMux should obtain local instances through http.NewServeMux() instead of using the global default instance.
			- ^http\.DefaultServeMux$
			- ^http\.Handle(?:Func)?$
			# Forbid usage of old and archived square/go-jose
			- pkg: ^gopkg\.in/square/go-jose\.v2$
				msg: "gopk.in/square/go-jose is archived and has CVEs. Replace it with gopkg.in/go-jose/go-jose.v2"
			- pkg: ^github.com/coreos/go-oidc$
				msg: "github.com/coreos/go-oidc depends on gopkg.in/square/go-jose which has CVEs. Replace it with github.com/coreos/go-oidc/v3"

			- pkg: ^github.com/howeyc/gopass$
				msg: "github.com/howeyc/gopass is archived, use golang.org/x/term instead"
	goconst:
		ignore-tests: true
		min-occurrences: 5
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
		gomoddirectives:
			toolchain-forbidden: true
			go-version-pattern: '1\.\d+(\.0)?$'
	gosec:
		excludes:
			# gosec wants us to set a short ReadHeaderTimeout to avoid Slowloris attacks, but doing so would expose us to Keep-Alive race conditions (see https://iximiuz.com/en/posts/reverse-proxy-http-keep-alive-and-502s/)
			- G112
			# created file permissions are restricted by umask if necessary
			- G306
	govet:
    enable-all: true
    disable:
      - fieldalignment
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
		sql-isolation-level: true
		time-layout: true
		time-month: true
		time-weekday: true
		tls-signature-scheme: true
		usetesting:
			os-setenv: true
			os-temp-dir: true
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
		- copyloopvar
		- dupl
		- dupword
		- durationcheck
		- errcheck
		- errname
		- errorlint
  {{- if lt .GoMinorVersion 22 }}
		- exportloopref
  {{- end }}
		- exptostd
		- forbidigo
		- ginkgolinter
		- gocheckcompilerdirectives
		- goconst
		- gocritic
		- gofmt
		- goimports
		- gomoddirectives
		- gosec
		- gosimple
		- govet
		- ineffassign
		- intrange
		- misspell
		- nilerr
		- noctx
		- nolintlint
		- nosprintfhostport
		- perfsprint
		- predeclared
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
		- usetesting
		- whitespace
`, "\t", "  "))))

type configTmplData struct {
	GoMinorVersion      int
	ModulePath          string
	ModDownloadMode     string
	MisspellIgnoreWords []string
	ErrcheckExcludes    []string
	SkipDirs            []string
}

func RenderConfig(cfg core.Configuration, sr golang.ScanResult) {
	mode := "readonly"
	if cfg.Golang.EnableVendoring {
		mode = "vendor"
	}

	f := must.Return(os.Create(".golangci.yaml"))
	fmt.Fprintln(f, core.AutogeneratedHeader+"\n")
	must.Succeed(configTmpl.Execute(f, configTmplData{
		GoMinorVersion:      must.Return(strconv.Atoi(strings.Split(sr.GoVersion, ".")[1])),
		ModulePath:          sr.ModulePath,
		ModDownloadMode:     mode,
		MisspellIgnoreWords: cfg.SpellCheck.IgnoreWords,
		ErrcheckExcludes:    cfg.GolangciLint.ErrcheckExcludes,
		SkipDirs:            cfg.GolangciLint.SkipDirs,
	}))
	fmt.Fprintln(f) // empty line at end

	must.Succeed(f.Close())
}
