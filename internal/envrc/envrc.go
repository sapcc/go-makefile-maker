// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package envrc

import (
	_ "embed"
	"maps"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

var (
	//go:embed envrc.tmpl
	envrcTemplate string
)

// RenderEnvRc renders the Nix shell.
func RenderEnvRc(cfg core.Configuration) {
	if !cfg.EnvRc.Enabled.UnwrapOr(false) && (!cfg.Nix.Enabled.UnwrapOr(true) || !cfg.EnvRc.Enabled.IsNone()) {
		return
	}
	logg.Debug("rendering envrc file")

	variables := maps.Clone(cfg.VariableValues)
	maps.Copy(variables, cfg.EnvRc.VariableValues)

	must.Succeed(util.WriteFileFromTemplate(".envrc", envrcTemplate, map[string]any{
		"Variables": variables,
	}))
}
