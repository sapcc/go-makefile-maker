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
	"strings"

	"github.com/sapcc/go-makefile-maker/internal/core"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
)

const goreleaserTemplate = `archives:
  - name_template: '%[1]s'%[2]s
    format_overrides:
      - goos: windows
        format: zip
    files:%[3]s

before:
  hooks:
    - go mod tidy

checksum:
  name_template: "checksums.txt"

builds:
  - binary: '%[4]s'
    env:
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
      - -X github.com/sapcc/go-api-declarations/bininfo.binName=%[5]s
      - -X github.com/sapcc/go-api-declarations/bininfo.version={{ .Version }}
      - -X github.com/sapcc/go-api-declarations/bininfo.commit={{ .FullCommit  }}
      - -X github.com/sapcc/go-api-declarations/bininfo.buildDate={{ .CommitDate }} # use CommitDate instead of Date for reproducibility%[6]s
    main: %[7]s
    # Set the modified timestamp on the output binary to ensure that builds are reproducible.
    mod_timestamp: "{{ .CommitTimestamp }}"

release:
  prerelease: auto
%[8]s
snapshot:
  name_template: "{{ .Tag }}-next"
`

func RenderConfig(cfg core.Configuration) {
	if len(cfg.Binaries) < 1 {
		logg.Fatal("GoReleaser requires at least 1 binary to be configured in binaries!")
	}
	if cfg.Metadata.URL == "" {
		logg.Fatal("GoReleaser requires metadata.url to be configured!")
	}

	var nameTemplate, format, ldflags, githubURLs string

	nameTemplate = `{{ .ProjectName }}-{{ replace .Version "v" "" }}-{{ .Os }}-{{ .Arch }}`
	if cfg.GoReleaser.NameTemplate != "" {
		nameTemplate = cfg.GoReleaser.NameTemplate
	}

	if cfg.GoReleaser.Format != "" {
		format = `
    format: ` + cfg.GoReleaser.Format
	}

	var files string
	if cfg.GoReleaser.Files == nil {
		files = `
      - CHANGELOG.md
      - LICENSE
      - README.md`
	} else {
		for _, file := range *cfg.GoReleaser.Files {
			files += "      - " + file
		}
	}

	binaryName := cfg.Binaries[0].Name
	if cfg.GoReleaser.BinaryName != "" {
		binaryName = cfg.GoReleaser.BinaryName
	}

	if len(cfg.Golang.LdFlags) > 0 {
		for name, value := range cfg.Golang.LdFlags {
			ldflags += fmt.Sprintf(`
      - %s={{.Env.%s}}`, name, value)
		}
	}

	if !strings.HasPrefix(cfg.Metadata.URL, "https://github.com/") {
		metadataURL, err := url.Parse(cfg.Metadata.URL)
		if err != nil {
			logg.Fatal("Metadata.URL is not a parsable URL: %w", err)
		}

		githubURLs = fmt.Sprintf(`
github_urls:
  api: https://%[1]s/api/v3/
  upload: https://%[1]s/api/uploads/
  download: https://%[1]s/
`, metadataURL.Host)
	}

	goreleaserFile := fmt.Sprintf(goreleaserTemplate, nameTemplate, format, files, binaryName, cfg.Binaries[0].Name, ldflags, cfg.Binaries[0].FromPackage, githubURLs)

	// Remove renamed file
	must.Succeed(os.RemoveAll(".goreleaser.yml"))
	must.Succeed(os.WriteFile(".goreleaser.yaml", []byte(goreleaserFile), 0666))
}
