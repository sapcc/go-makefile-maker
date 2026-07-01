// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package hyperspace

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

type config struct {
	Schema   string   `json:"$schema"`
	Features features `json:"features"`
}

type features struct {
	ControlPanel bool            `json:"control_panel"`
	Summarize    summarizeConfig `json:"summarize"`
	Review       reviewConfig    `json:"review"`
}

type summarizeConfig struct {
	AutoGenerateSummary bool `json:"auto_generate_summary"`
	AutoInsertSummary   bool `json:"auto_insert_summary"`
}

type reviewConfig struct {
	AutoGenerateReview bool `json:"auto_generate_review"`
}

var defaultConfig = config{
	Schema: "https://devops-insights-pr-bot.cfapps.eu10-004.hana.ondemand.com/schema/pull_request_bot.json",
	Features: features{
		ControlPanel: false,
		Summarize: summarizeConfig{
			AutoGenerateSummary: false,
			AutoInsertSummary:   false,
		},
		Review: reviewConfig{
			AutoGenerateReview: false,
		},
	},
}

// RenderConfig writes the renovate configuration files from the provided config and scan results.
func RenderConfig(cfg core.Configuration) {
	isInternalRepo := strings.HasPrefix(cfg.Metadata.URL, "https://github.wdf.sap.corp") || strings.HasPrefix(cfg.Metadata.URL, "https://github.tools.sap")
	if !isInternalRepo {
		return
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	must.Succeed(encoder.Encode(defaultConfig))

	must.Succeed(os.MkdirAll(".hyperspace", 0750))
	must.Succeed(util.WriteFile(".hyperspace/pull_request_bot.json", buf.Bytes()))
}
