// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package golangcilint

import (
	_ "embed"
	"strconv"
	"strings"
	"time"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

var (
	//go:embed golangci.yaml.tmpl
	configTemplate string
)

func RenderConfig(cfg core.Configuration, sr golang.ScanResult) {
	timeout := 3 * time.Minute
	if cfg.GolangciLint.Timeout != 0 {
		timeout = cfg.GolangciLint.Timeout
	}

	must.Succeed(util.WriteFileFromTemplate(".golangci.yaml", configTemplate, map[string]any{
		"EnableVendoring":     cfg.Golang.EnableVendoring,
		"GoMinorVersion":      must.Return(strconv.Atoi(strings.Split(sr.GoVersion, ".")[1])),
		"ModulePath":          sr.ModulePath,
		"MisspellIgnoreWords": cfg.SpellCheck.IgnoreWords,
		"ErrcheckExcludes":    cfg.GolangciLint.ErrcheckExcludes,
		"SkipDirs":            cfg.GolangciLint.SkipDirs,
		"Timeout":             timeout,
		"ReviveRules":         cfg.GolangciLint.ReviveRules,
	}))
}
