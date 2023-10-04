/******************************************************************************
*
*  Copyright 2023 SAP SE
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
******************************************************************************/

package goreleaser

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
)

const goreleaserTemplate = `before:
  hooks:
    - go mod tidy
    - rm -rf completions
    - mkdir completions
    - go run main.go completion bash >"completions/%[1]s.bash"
    - go run main.go completion fish >"completions/%[1]s.fish"
    - go run main.go completion zsh >"completions/%[1]s.zsh"

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/sapcc/go-api-declarations/bininfo.binName=%[1]s
      - -X github.com/sapcc/go-api-declarations/bininfo.version={{ .Version }}
      - -X github.com/sapcc/go-api-declarations/bininfo.commit={{ .FullCommit  }}
      # Use CommitDate instead of Date for reproducibility.
      - -X github.com/sapcc/go-api-declarations/bininfo.buildDate={{ .CommitDate }}
    # Set the modified timestamp on the output binary to ensure that builds are reproducible.
    mod_timestamp: "{{ .CommitTimestamp }}"

snapshot:
  name_template: "{{ .Tag }}-next"

checksum:
  name_template: "checksums.txt"

archives:
  - name_template: '{{ .ProjectName }}-{{ replace .Version "v" "" }}-{{ .Os }}-{{ .Arch }}'
    format_overrides:
      - goos: windows
        format: zip
    files:
      - CHANGELOG.md
      - LICENSE
      - README.md
      - completions/*

brews:
  - repository:
      owner: %[3]s
      name: %[4]s
    folder: HomebrewFormula
    homepage: %[2]s
    description: Command-line interface for Limes
    license: Apache-2.0
    install: |-
      bin.install "%[1]s"
      bash_completion.install "completions/%[1]s.bash" => "%[1]s"
      fish_completion.install "completions/%[1]s.fish"
      zsh_completion.install "completions/%[1]s.zsh" => "_%[1]s"
    test: |
      system "#{bin}/%[1]s --version"
    commit_msg_template: "Homebrew: update formula to {{ .Tag }}"
`

func RenderConfig(cfg core.Configuration) {
	if len(cfg.Binaries) < 1 {
		logg.Fatal("Goreleaser requires at least 1 binary to be configured in binaries!")
	}
	if cfg.Metadata.URL == "" {
		logg.Fatal("Goreleasre requires metadata.url to be configured!")
	}

	metadataurl := must.Return(url.Parse(cfg.Metadata.URL))
	githubOrg, githubRepo := filepath.Split(metadataurl.Path)

	goreleaserFile := fmt.Sprintf(goreleaserTemplate, cfg.Binaries[0].Name, cfg.Metadata.URL, strings.ReplaceAll(githubOrg, "/", ""), githubRepo)

	must.Succeed(os.WriteFile(".goreleaser.yml", []byte(goreleaserFile), 0666))
}
