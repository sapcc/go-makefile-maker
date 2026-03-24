// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package nix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/majewsky/gg/option"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

// assertFileExists checks if a file exists and fails the test if the expectation is not met.
func assertFileExists(t *testing.T, path string, shouldExist bool) {
	t.Helper()
	_, err := os.Stat(path)
	exists := !os.IsNotExist(err)

	if shouldExist && !exists {
		t.Errorf("%s should exist but doesn't", path)
	}
	if !shouldExist && exists {
		t.Errorf("%s should not exist but does", path)
	}
}

// assertFileContains checks if a file contains the expected substrings.
func assertFileContains(t *testing.T, path string, substrings ...string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", path, err)
	}
	contentStr := string(content)
	for _, substring := range substrings {
		if !strings.Contains(contentStr, substring) {
			t.Errorf("%s should contain %q", path, substring)
		}
	}
}

func TestRenderShell_ConfigurationOptions(t *testing.T) {
	tests := []struct {
		name           string
		nixConfig      core.NixConfig
		expectShellNix bool
		expectEnvrc    bool
	}{
		{
			name: "Nix disabled",
			nixConfig: core.NixConfig{
				Enabled: Some(false),
			},
			expectShellNix: false,
			expectEnvrc:    false,
		},
		{
			name:      "Nix enabled by default",
			nixConfig: core.NixConfig{
				// Enabled not set, should default to true
			},
			expectShellNix: true,
			expectEnvrc:    true,
		},
		{
			name: "Nix explicitly enabled",
			nixConfig: core.NixConfig{
				Enabled: Some(true),
			},
			expectShellNix: true,
			expectEnvrc:    true,
		},
		{
			name: "WriteEnvRc disabled",
			nixConfig: core.NixConfig{
				Enabled:    Some(true),
				WriteEnvRc: Some(false),
			},
			expectShellNix: true,
			expectEnvrc:    false,
		},
		{
			name: "WriteEnvRc enabled by default",
			nixConfig: core.NixConfig{
				Enabled: Some(true),
				// WriteEnvRc not set, should default to true
			},
			expectShellNix: true,
			expectEnvrc:    true,
		},
		{
			name: "WriteEnvRc explicitly enabled",
			nixConfig: core.NixConfig{
				Enabled:    Some(true),
				WriteEnvRc: Some(true),
			},
			expectShellNix: true,
			expectEnvrc:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			cfg := core.Configuration{
				Nix: tt.nixConfig,
			}
			sr := golang.ScanResult{}

			RenderShell(cfg, sr, false)

			assertFileExists(t, "shell.nix", tt.expectShellNix)
			assertFileExists(t, ".envrc", tt.expectEnvrc)
		})
	}
}

func TestRenderShell_WithExtraPackages(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := core.Configuration{
		Nix: core.NixConfig{
			Enabled:       Some(true),
			ExtraPackages: []string{"jq", "curl"},
			WriteEnvRc:    Some(true),
		},
	}
	sr := golang.ScanResult{}

	RenderShell(cfg, sr, false)

	assertFileExists(t, "shell.nix", true)
	assertFileExists(t, ".envrc", true)
	assertFileContains(t, "shell.nix", "jq", "curl")
}

func TestRenderShell_WithVariables(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := core.Configuration{
		Nix: core.NixConfig{
			Enabled:    Some(true),
			WriteEnvRc: Some(true),
		},
		VariableValues: map[string]string{
			"TEST_VAR": "test_value",
		},
	}
	sr := golang.ScanResult{}

	RenderShell(cfg, sr, false)

	assertFileExists(t, ".envrc", true)
	assertFileContains(t, ".envrc", "TEST_VAR")
}

func TestRenderShell_PackageInclusion(t *testing.T) {
	tests := []struct {
		name                   string
		golangciLintConfig     core.GolangciLintConfiguration
		renderGoreleaserConfig bool
		expectedPackages       []string
	}{
		{
			name: "with golangci-lint",
			golangciLintConfig: core.GolangciLintConfiguration{
				CreateConfig: true,
			},
			renderGoreleaserConfig: false,
			expectedPackages:       []string{"golangci-lint"},
		},
		{
			name:                   "with goreleaser",
			golangciLintConfig:     core.GolangciLintConfiguration{},
			renderGoreleaserConfig: true,
			expectedPackages:       []string{"goreleaser", "syft"},
		},
		{
			name: "with both golangci-lint and goreleaser",
			golangciLintConfig: core.GolangciLintConfiguration{
				CreateConfig: true,
			},
			renderGoreleaserConfig: true,
			expectedPackages:       []string{"golangci-lint", "goreleaser", "syft"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			cfg := core.Configuration{
				Nix: core.NixConfig{
					Enabled: Some(true),
				},
				GolangciLint: tt.golangciLintConfig,
			}
			sr := golang.ScanResult{}

			RenderShell(cfg, sr, tt.renderGoreleaserConfig)

			assertFileContains(t, "shell.nix", tt.expectedPackages...)
		})
	}
}

func TestRenderShell_ComplexScenario(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	cfg := core.Configuration{
		Nix: core.NixConfig{
			Enabled:        Some(true),
			ExtraPackages:  []string{"postgresql", "redis"},
			ExtraLibraries: []string{"libpq", "libssl"},
			WriteEnvRc:     Some(true),
		},
		GolangciLint: core.GolangciLintConfiguration{
			CreateConfig: true,
		},
		Renovate: core.RenovateConfig{
			Enabled: true,
		},
		Typos: core.TyposConfiguration{
			Enabled: Some(true),
		},
		Reuse: core.ReuseConfiguration{
			Enabled: Some(true),
		},
		VariableValues: map[string]string{
			"ENV_VAR1": "value1",
			"ENV_VAR2": "value2",
		},
	}
	sr := golang.ScanResult{
		UsesPostgres: true,
	}

	RenderShell(cfg, sr, true)

	shellNixPath := filepath.Join(tempDir, "shell.nix")
	envrcPath := filepath.Join(tempDir, ".envrc")

	assertFileExists(t, shellNixPath, true)
	assertFileExists(t, envrcPath, true)

	expectedPackages := []string{
		"golangci-lint",
		"goreleaser",
		"syft",
		"renovate",
		"typos",
		"reuse",
		"postgresql",
		"redis",
	}
	assertFileContains(t, shellNixPath, expectedPackages...)

	expectedLibraries := []string{"libpq", "libssl"}
	assertFileContains(t, shellNixPath, expectedLibraries...)
	assertFileContains(t, envrcPath, "ENV_VAR1", "ENV_VAR2")
}
