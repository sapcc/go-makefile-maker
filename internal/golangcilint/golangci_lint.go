// Copyright 2021 SAP SE
// SPDX-License-Identifier: Apache-2.0

package golangcilint

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

var configTmpl = template.Must(template.New("golangci").Parse(strings.TrimSpace(strings.ReplaceAll(`
version: "2"
run:
	modules-download-mode: {{ .ModDownloadMode }}
	timeout: {{ .Timeout }} # none by default in v2

formatters:
	enable:
		- gofmt
		- goimports
	settings:
		goimports:
			# Put local imports after 3rd-party packages
			local-prefixes:
				- {{ .ModulePath }}
	exclusions:
		generated: lax
		paths:
			- third_party$
			- builtin$
			- examples$

issues:
	# '0' disables the following options
	max-issues-per-linter: 0
	max-same-issues: 0

linters:
	# Disable all pre-enabled linters and enable them explicitly so that a newer version does not introduce new linters unexpectedly
	default: none
	enable:
		- bodyclose
		- containedctx
		- copyloopvar
		- dupword
		- durationcheck
		- errcheck
		- errname
		- errorlint
		- exptostd
		- forbidigo
		- ginkgolinter
		- gocheckcompilerdirectives
		- goconst
		- gocritic
		- gomoddirectives
		- gosec
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
		- unconvert
		- unparam
		- unused
		- usestdlibvars
		- usetesting
		- whitespace
	settings:
		errcheck:
			check-type-assertions: false
			# Report about assignment of errors to blank identifier.
			check-blank: true
			# Do not report about not checking of errors in type assertions.
			# This is not as dangerous as skipping error values because an unchecked type assertion just immediately panics.
			# We disable this because it makes a ton of useless noise esp. in test code.
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
				- pattern: ^ioutil\..*$
				# Using http.DefaultServeMux is discouraged because it's a global variable that some packages silently and magically add handlers to (esp. net/http/pprof).
				# Applications wishing to use http.ServeMux should obtain local instances through http.NewServeMux() instead of using the global default instance.
				- pattern: ^http\.DefaultServeMux$
				- pattern: ^http\.Handle(?:Func)?$
				- pkg: ^gopkg\.in/square/go-jose\.v2$
					msg: gopk.in/square/go-jose is archived and has CVEs. Replace it with gopkg.in/go-jose/go-jose.v2
				- pkg: ^github.com/coreos/go-oidc$
					msg: github.com/coreos/go-oidc depends on gopkg.in/square/go-jose which has CVEs. Replace it with github.com/coreos/go-oidc/v3
				- pkg: ^github.com/howeyc/gopass$
					msg: github.com/howeyc/gopass is archived, use golang.org/x/term instead
		goconst:
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
		gomoddirectives:
			replace-allow-list:
				# for go-pmtud
				- github.com/mdlayher/arp
			toolchain-forbidden: true
			go-version-pattern: 1\.\d+(\.0)?$
		gosec:
			excludes:
				# gosec wants us to set a short ReadHeaderTimeout to avoid Slowloris attacks, but doing so would expose us to Keep-Alive race conditions (see https://iximiuz.com/en/posts/reverse-proxy-http-keep-alive-and-502s/
				- G112
				# created file permissions are restricted by umask if necessary
				- G306
		govet:
			disable:
				- fieldalignment
			enable-all: true
		nolintlint:
			require-specific: true
		{{- if .MisspellIgnoreWords }}
		misspell:
			ignore-words:
				{{- range .MisspellIgnoreWords }}
				- {{ . }}
				{{- end }}
		{{- end }}
		staticcheck:
			dot-import-whitelist:
				- github.com/majewsky/gg/option
				- github.com/onsi/ginkgo/v2
				- github.com/onsi/gomega
		usestdlibvars:
			http-method: true
			http-status-code: true
			time-weekday: true
			time-month: true
			time-layout: true
			crypto-hash: true
			default-rpc-path: true
			sql-isolation-level: true
			tls-signature-scheme: true
			constant-kind: true
		usetesting:
			os-temp-dir: true
		whitespace:
			# Enforce newlines (or comments) after multi-line function signatures.
			multi-func: true
	exclusions:
		generated: lax
		presets:
			- comments
			- common-false-positives
			- legacy
			- std-error-handling
		rules:
			- linters:
					- bodyclose
				path: _test\.go
			# It is idiomatic Go to reuse the name 'err' with ':=' for subsequent errors.
			# Ref: https://go.dev/doc/effective_go#redeclaration
			- path: (.+)\.go$
				text: declaration of "err" shadows declaration at
			- linters:
					- goconst
				path: (.+)_test\.go
			{{- if .SkipDirs }}
			{{- range .SkipDirs }}
			- path:
				- {{ . }}
			{{- end }}
			{{- end }}
		paths:
			- third_party$
			- builtin$
			- examples$

output:
  formats:
    text:
			# Do not print lines of code with issue.
      print-issued-lines: false
`, "\t", "  "))))

type configTmplData struct {
	GoMinorVersion      int
	ModulePath          string
	ModDownloadMode     string
	MisspellIgnoreWords []string
	ErrcheckExcludes    []string
	SkipDirs            []string
	Timeout             time.Duration
}

func RenderConfig(cfg core.Configuration, sr golang.ScanResult) {
	mode := "readonly"
	if cfg.Golang.EnableVendoring {
		mode = "vendor"
	}

	timeout := 3 * time.Minute
	if cfg.GolangciLint.Timeout != 0 {
		timeout = cfg.GolangciLint.Timeout
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
		Timeout:             timeout,
	}))
	fmt.Fprintln(f) // empty line at end

	must.Succeed(f.Close())
}
