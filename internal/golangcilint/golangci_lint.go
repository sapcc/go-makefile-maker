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

// RenderConfig writes the golanci-lint configuration files from the provided config and scan results.
func RenderConfig(cfg core.Configuration, sr golang.ScanResult) {
	timeout := 3 * time.Minute
	if cfg.GolangciLint.Timeout != 0 {
		timeout = cfg.GolangciLint.Timeout
	}

	must.Succeed(util.WriteFileFromTemplate(".golangci.yaml", configTemplate, map[string]any{
		"ReplaceAllowList":  cfg.GolangciLint.ReplaceAllowList,
		"EnableVendoring":   cfg.Golang.EnableVendoring,
		"ErrcheckExcludes":  cfg.GolangciLint.ErrcheckExcludes,
		"ForbidigoRules":    cfg.GolangciLint.ForbidigoRules,
		"GoMinorVersion":    must.Return(strconv.Atoi(strings.Split(sr.GoVersion, ".")[1])),
		"ModulePath":        sr.ModulePath,
		"ReviveRules":       cfg.GolangciLint.ReviveRules,
		"SkipDirs":          cfg.GolangciLint.SkipDirs,
		"Timeout":           timeout,
		"WithControllerGen": cfg.ControllerGen.Enabled.UnwrapOr(sr.KubernetesController),
	}))
}
