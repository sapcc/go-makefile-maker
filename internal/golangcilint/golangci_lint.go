// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package golangcilint

import (
	"cmp"
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
	must.Succeed(util.WriteFileFromTemplate(".golangci.yaml", configTemplate, map[string]any{
		"ReplaceAllowList":  cfg.GolangciLint.ReplaceAllowList,
		"EnableVendoring":   cfg.Golang.EnableVendoring,
		"ErrcheckExcludes":  cfg.GolangciLint.ErrcheckExcludes,
		"ForbidigoRules":    cfg.GolangciLint.ForbidigoRules,
		"GoMinorVersion":    must.Return(strconv.Atoi(strings.Split(sr.GoVersion, ".")[1])),
		"ModulePath":        sr.ModulePath,
		"ReviveRules":       cfg.GolangciLint.ReviveRules,
		"SkipDirs":          cfg.GolangciLint.SkipDirs,
		"Timeout":           cmp.Or(cfg.GolangciLint.Timeout, 5*time.Minute), // default to 5m0s
		"WithControllerGen": cfg.ControllerGen.Enabled.UnwrapOr(sr.KubernetesController),
		// liquid-ceph has an insane vendoring setup that we tried to replace with Go workspaces,
		// but after getting stuck on bizarre module lookup errors, we decided to grandfather this in for now
		"AllowReplaceLocal": sr.ModulePath == "github.com/cobaltcore-dev/liquid-ceph",
	}))
}
