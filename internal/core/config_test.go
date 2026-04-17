// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/majewsky/gg/option"
)

func baseConfig() Configuration {
	return Configuration{
		VariableValues: map[string]string{},
	}
}

func withFakeGit(t *testing.T, mode string) func() {
	t.Helper()

	tmpDir := t.TempDir()
	gitPath := filepath.Join(tmpDir, "git")
	script := `#!/bin/sh
case "$GIT_FAKE_MODE" in
  error)
	echo "git failed" 1>&2
	exit 1
	;;
  badprefix)
	echo "refs/heads/main"
	exit 0
	;;
  ok)
	echo "refs/remotes/origin/main"
	exit 0
	;;
  *)
	echo "unsupported mode" 1>&2
	exit 2
	;;
esac
`
	if err := os.WriteFile(gitPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake git: %v", err)
	}

	oldPath := os.Getenv("PATH")
	oldMode := os.Getenv("GIT_FAKE_MODE")
	if err := os.Setenv("PATH", fmt.Sprintf("%s:%s", tmpDir, oldPath)); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	if err := os.Setenv("GIT_FAKE_MODE", mode); err != nil {
		t.Fatalf("failed to set GIT_FAKE_MODE: %v", err)
	}

	return func() {
		_ = os.Setenv("PATH", oldPath)
		_ = os.Setenv("GIT_FAKE_MODE", oldMode)
	}
}

func configForCase(name string) Configuration {
	cfg := baseConfig()

	switch name {
	case "ok_default":
		return cfg

	case "fail_spellcheck_deprecated":
		cfg.SpellCheck.IgnoreWords = []string{"legacy"}
		return cfg

	case "fail_docker_without_metadata":
		cfg.Dockerfile.Enabled = true
		return cfg

	case "fail_multiple_opt_resource":
		cfg.Binaries = []BinaryConfiguration{
			{Name: "a", InstallTo: "/opt/resource"},
			{Name: "b", InstallTo: "/opt/resource"},
		}
		return cfg

	case "fail_golangcilint_without_createconfig":
		cfg.GolangciLint.ErrcheckExcludes = []string{"io.Close"}
		cfg.GolangciLint.CreateConfig = false
		return cfg

	case "fail_forbidigo_empty_rule":
		cfg.GolangciLint.CreateConfig = true
		cfg.GolangciLint.ForbidigoRules = []ForbidigoRule{
			{},
		}
		return cfg

	case "fail_githubworkflow_without_metadata":
		cfg.GitHubWorkflow = &GithubWorkflowConfiguration{
			Global: struct {
				DefaultBranch string         `yaml:"defaultBranch"`
				GoVersion     Option[string] `yaml:"goVersion"`
			}{
				DefaultBranch: "main",
			},
		}
		return cfg

	case "fail_githubworkflow_defaultbranch_git_error":
		cfg.Metadata.URL = "https://github.com/example/repo"
		cfg.GitHubWorkflow = &GithubWorkflowConfiguration{}
		return cfg

	case "fail_githubworkflow_defaultbranch_bad_output":
		cfg.Metadata.URL = "https://github.com/example/repo"
		cfg.GitHubWorkflow = &GithubWorkflowConfiguration{}
		return cfg

	case "fail_ci_runson_non_ubuntu_multiple":
		cfg.Metadata.URL = "https://github.com/example/repo"
		cfg.GitHubWorkflow = &GithubWorkflowConfiguration{
			Global: struct {
				DefaultBranch string         `yaml:"defaultBranch"`
				GoVersion     Option[string] `yaml:"goVersion"`
			}{
				DefaultBranch: "main",
			},
			CI: CIWorkflowConfig{
				Enabled: true,
				RunsOn:  []string{"self-hosted", "ubuntu-latest"},
			},
		}
		return cfg

	case "fail_sap_securitychecks_disabled":
		cfg.Metadata.URL = "https://github.com/sapcc/repo"
		cfg.Renovate.Enabled = true
		cfg.GitHubWorkflow = &GithubWorkflowConfiguration{
			Global: struct {
				DefaultBranch string         `yaml:"defaultBranch"`
				GoVersion     Option[string] `yaml:"goVersion"`
			}{
				DefaultBranch: "main",
			},
			SecurityChecks: SecurityChecksWorkflowConfig{
				Enabled: Some(false),
			},
		}
		return cfg

	case "fail_sap_renovate_disabled":
		cfg.Metadata.URL = "https://github.com/sapcc/repo"
		cfg.Renovate.Enabled = false
		return cfg

	case "fail_reserved_variable":
		cfg.VariableValues["BININFO_VERSION"] = "override"
		return cfg

	case "ok_sap_compliant":
		cfg.Metadata.URL = "https://github.com/sapcc/repo"
		cfg.Renovate.Enabled = true
		cfg.GitHubWorkflow = &GithubWorkflowConfiguration{
			Global: struct {
				DefaultBranch string         `yaml:"defaultBranch"`
				GoVersion     Option[string] `yaml:"goVersion"`
			}{
				DefaultBranch: "main",
			},
		}
		return cfg

	default:
		panic("unknown case: " + name)
	}
}

func TestValidate_SubprocessCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantFail bool
		gitMode  string
	}{
		{name: "ok_default", wantFail: false},
		{name: "fail_spellcheck_deprecated", wantFail: true},
		{name: "fail_docker_without_metadata", wantFail: true},
		{name: "fail_multiple_opt_resource", wantFail: true},
		{name: "fail_golangcilint_without_createconfig", wantFail: true},
		{name: "fail_forbidigo_empty_rule", wantFail: true},
		{name: "fail_githubworkflow_without_metadata", wantFail: true},
		{name: "fail_githubworkflow_defaultbranch_git_error", wantFail: true, gitMode: "error"},
		{name: "fail_githubworkflow_defaultbranch_bad_output", wantFail: true, gitMode: "badprefix"},
		{name: "fail_ci_runson_non_ubuntu_multiple", wantFail: true},
		{name: "fail_sap_securitychecks_disabled", wantFail: true},
		{name: "fail_sap_renovate_disabled", wantFail: true},
		{name: "fail_reserved_variable", wantFail: true},
		{name: "ok_sap_compliant", wantFail: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestValidate_SubprocessHelper")
			cmd.Env = append(os.Environ(),
				"GO_WANT_VALIDATE_HELPER=1",
				"VALIDATE_CASE="+tc.name,
				"GIT_FAKE_MODE="+tc.gitMode,
			)

			if tc.gitMode != "" {
				tmpDir := t.TempDir()
				gitPath := filepath.Join(tmpDir, "git")
				script := `#!/bin/sh
case "$GIT_FAKE_MODE" in
  error)
	exit 1
	;;
  badprefix)
	echo "refs/heads/main"
	exit 0
	;;
  ok)
	echo "refs/remotes/origin/main"
	exit 0
	;;
esac
exit 2
`
				if err := os.WriteFile(gitPath, []byte(script), 0o755); err != nil {
					t.Fatalf("failed to write fake git: %v", err)
				}
				cmd.Env = append(cmd.Env, "PATH="+tmpDir+":"+os.Getenv("PATH"))
			}

			err := cmd.Run()
			if tc.wantFail {
				if err == nil {
					t.Fatalf("expected failure, got success")
				}
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("expected ExitError, got %T (%v)", err, err)
				}
			} else if err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
		})
	}
}

func TestValidate_SubprocessHelper(t *testing.T) {
	if os.Getenv("GO_WANT_VALIDATE_HELPER") != "1" {
		return
	}
	cfg := configForCase(os.Getenv("VALIDATE_CASE"))
	cfg.Validate()
}

func TestValidate_GitDefaultBranchResolved(t *testing.T) {
	cfg := baseConfig()
	cfg.Metadata.URL = "https://github.com/example/repo"
	cfg.GitHubWorkflow = &GithubWorkflowConfiguration{}

	restore := withFakeGit(t, "ok")
	defer restore()

	cfg.Validate()

	if cfg.GitHubWorkflow.Global.DefaultBranch != "main" {
		t.Fatalf("expected default branch to be resolved to main, got %q", cfg.GitHubWorkflow.Global.DefaultBranch)
	}
}
