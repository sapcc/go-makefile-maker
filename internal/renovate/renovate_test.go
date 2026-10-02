// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package renovate

import (
	"os"
	"strings"
	"testing"

	. "go.xyrillian.de/gg/option"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

func readRenovateConfig(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(".github/renovate.json5")
	if err != nil {
		t.Fatalf("failed to read generated renovate.json5: %v", err)
	}
	return string(data)
}

func baseConfig() core.Configuration {
	return core.Configuration{
		GitHubWorkflow: &core.GithubWorkflowConfiguration{},
	}
}

func TestPackageRule_PlatformAutomergeAbsent(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := baseConfig()
	cfg.Renovate = core.RenovateConfig{
		PackageRules: []core.PackageRule{
			{
				MatchPackageNames: []string{"some/package"},
				AutoMerge:         true,
				// PlatformAutomerge intentionally not set
			},
		},
	}

	RenderConfig(cfg, golang.ScanResult{}, nil)

	content := readRenovateConfig(t)
	if strings.Contains(content, "platformAutomerge") {
		t.Error("expected platformAutomerge to be absent when not set, but it was present in the output")
	}
}

func TestPackageRule_PlatformAutomergeFalse(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := baseConfig()
	cfg.Renovate = core.RenovateConfig{
		PackageRules: []core.PackageRule{
			{
				MatchPackageNames: []string{"some/package"},
				AutoMerge:         true,
				PlatformAutomerge: Some(false),
			},
		},
	}

	RenderConfig(cfg, golang.ScanResult{}, nil)

	content := readRenovateConfig(t)
	if !strings.Contains(content, `"platformAutomerge": false`) {
		t.Errorf("expected \"platformAutomerge\": false in output, got:\n%s", content)
	}
}

func TestPackageRule_PlatformAutomergeTrue(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := baseConfig()
	cfg.Renovate = core.RenovateConfig{
		PackageRules: []core.PackageRule{
			{
				MatchPackageNames: []string{"some/package"},
				AutoMerge:         false,
				PlatformAutomerge: Some(true),
			},
		},
	}

	RenderConfig(cfg, golang.ScanResult{}, nil)

	content := readRenovateConfig(t)
	if !strings.Contains(content, `"platformAutomerge": true`) {
		t.Errorf("expected \"platformAutomerge\": true in output, got:\n%s", content)
	}
}
