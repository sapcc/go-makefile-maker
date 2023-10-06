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
	"os"

	"github.com/sapcc/go-makefile-maker/internal/core"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
)

const goreleaserTemplate = `before:
  hooks:
    - go mod tidy

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
      - -X github.com/sapcc/go-api-declarations/bininfo.buildDate={{ .CommitDate }} # use CommitDate instead of Date for reproducibility
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
`

func RenderConfig(cfg core.Configuration) {
	if len(cfg.Binaries) < 1 {
		logg.Fatal("Goreleaser requires at least 1 binary to be configured in binaries!")
	}
	if cfg.Metadata.URL == "" {
		logg.Fatal("Goreleasre requires metadata.url to be configured!")
	}

	goreleaserFile := fmt.Sprintf(goreleaserTemplate, cfg.Binaries[0].Name)

	// Remove renamed file
	must.Succeed(os.RemoveAll(".goreleaser.yml"))
	must.Succeed(os.WriteFile(".goreleaser.yaml", []byte(goreleaserFile), 0666))
}
