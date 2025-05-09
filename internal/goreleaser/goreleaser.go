// SPDX-FileCopyrightText: Copyright 2023 SAP SE
// SPDX-License-Identifier: Apache-2.0

package goreleaser

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	_ "embed"

	"github.com/sapcc/go-makefile-maker/internal/core"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
)

var goreleaserTemplate = strings.ReplaceAll(`# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

archives:
	- name_template: '%[1]s'%[2]s
		format_overrides:
			- goos: windows
				format: zip
		files:%[3]s

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
		ignore:
			- goos: windows
				goarch: arm64
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
	make_latest: true
	prerelease: auto
%[8]s
snapshot:
	version_template: "{{ .Tag }}-next"
`, "\t", "  ")

//go:embed RELEASE.md
var releaseMD string

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
		format = strings.ReplaceAll(`
		format: `, "\t", "  ") + cfg.GoReleaser.Format
	}

	var files string
	if cfg.GoReleaser.Files == nil {
		files = strings.ReplaceAll(`
			- CHANGELOG.md
			- LICENSE
			- README.md`,
			"\t", "  ")
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
		var names []string
		for key := range cfg.Golang.LdFlags {
			names = append(names, key)
		}
		sort.Strings(names)

		for _, name := range names {
			value := cfg.Golang.LdFlags[name]
			ldflags += fmt.Sprintf(strings.ReplaceAll(`
			- %s={{.Env.%s}}`, "\t", "  "), name, value)
		}
	}

	if !strings.HasPrefix(cfg.Metadata.URL, "https://github.com/") {
		metadataURL, err := url.Parse(cfg.Metadata.URL)
		if err != nil {
			logg.Fatal("Metadata.URL is not a parsable URL: %w", err)
		}

		githubURLs = fmt.Sprintf(strings.ReplaceAll(`
github_urls:
	api: https://%[1]s/api/v3/
	upload: https://%[1]s/api/uploads/
	download: https://%[1]s/
`, "\t", "  "), metadataURL.Host)
	}

	goreleaserFile := fmt.Sprintf(goreleaserTemplate, nameTemplate, format, files, binaryName, cfg.Binaries[0].Name, ldflags, cfg.Binaries[0].FromPackage, githubURLs)

	// Remove renamed file
	must.Succeed(os.RemoveAll(".goreleaser.yml"))
	must.Succeed(os.WriteFile(".goreleaser.yaml", []byte(goreleaserFile), 0666))
	must.Succeed(os.WriteFile("RELEASE.md", []byte(releaseMD), 0666))
}
