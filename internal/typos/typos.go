// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package typos

import (
	_ "embed"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

var (
	//go:embed typos.toml.tmpl
	typosConfigTemplate string
)

func RenderConfig(cfg core.Configuration) {
	extendExcludes := []string{"go.mod"}
	if cfg.Golang.EnableVendoring {
		extendExcludes = append(extendExcludes, "vendor/")
	}
	extendExcludes = append(extendExcludes, cfg.Typos.ExtendExcludes...)

	must.Succeed(util.WriteFileFromTemplate(".typos.toml", typosConfigTemplate, map[string]any{
		"ExtendExcludes": extendExcludes,
		"ExtendWords":    cfg.Typos.ExtendWords,
	}))
}
