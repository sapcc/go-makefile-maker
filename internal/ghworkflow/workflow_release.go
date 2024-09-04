// Copyright 2023 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// limitations under the License.

package ghworkflow

import (
	"bytes"
	"net/url"
	"os"
	"strings"
	"text/template"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

const enterpriseGoReleaserConfig string = `github_urls:
  api: https://{{ . }}/api/v3/
  upload: https://{{ . }}/api/uploads/
  download: https://{{ . }}/
`

func releaseWorkflow(cfg core.Configuration) {
	// https://docs.github.com/en/packages/managing-github-packages-using-github-actions-workflows/publishing-and-installing-a-package-with-github-actions#publishing-a-package-using-an-action
	ghwCfg := cfg.GitHubWorkflow
	w := newWorkflow("goreleaser", ghwCfg.Global.DefaultBranch, nil)

	if w.deleteIf(ghwCfg.Release.Enabled) {
		return
	}

	w.Permissions.Contents = tokenScopeWrite
	w.Permissions.Packages = tokenScopeWrite

	w.On.Push.Branches = nil
	w.On.PullRequest.Branches = nil
	w.On.Push.Tags = []string{"*"} // goreleaser uses semver to decide if this is a prerelease or not

	j := baseJobWithGo("goreleaser", cfg)
	// This is needed because: https://goreleaser.com/ci/actions/#fetch-depthness
	j.Steps[0].With = map[string]any{
		"fetch-depth": 0,
	}
	j.addStep(jobStep{
		Name: "Generate release info",
		Run: makeMultilineYAMLString([]string{
			"go install github.com/sapcc/go-bits/tools/release-info@latest",
			"mkdir -p build",
			`release-info CHANGELOG.md "$(git describe --tags --abbrev=0)" > build/release-info`,
		}),
	})
	j.addStep(jobStep{
		Name: "Run GoReleaser",
		Uses: core.GoreleaserAction,
		With: map[string]any{
			"version": "latest",
			"args":    "release --clean --release-notes=./build/release-info",
		},
		Env: map[string]string{
			"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		},
	})
	w.Jobs = map[string]job{"release": j}

	writeWorkflowToFile(w)

	isInternal := !strings.HasPrefix(cfg.Metadata.URL, "https://github.com")
	if !isInternal {
		return
	}
	ghURL := must.Return(url.Parse(cfg.Metadata.URL))
	tpl := template.New("")
	must.Return(tpl.Parse(enterpriseGoReleaserConfig))
	var buf bytes.Buffer
	must.Succeed(tpl.Execute(&buf, ghURL.Host))
	must.Succeed(os.WriteFile(".goreleaser.yaml", buf.Bytes(), 0644))
}
