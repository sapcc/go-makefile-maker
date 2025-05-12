// SPDX-FileCopyrightText: 2023 SAP SE
// SPDX-License-Identifier: Apache-2.0

package goreleaser

import (
	"net/url"
	"os"
	"strings"
	"text/template"

	_ "embed"

	"github.com/sapcc/go-makefile-maker/internal/core"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
)

var (
	//go:embed RELEASE.md
	releaseMD string

	//go:embed goreleaser.yaml.tmpl
	goreleaserTemplate string
)

func RenderConfig(cfg core.Configuration) {
	if len(cfg.Binaries) < 1 {
		logg.Fatal("GoReleaser requires at least 1 binary to be configured in binaries!")
	}
	if cfg.Metadata.URL == "" {
		logg.Fatal("GoReleaser requires metadata.url to be configured!")
	}

	nameTemplate := `{{ .ProjectName }}-{{ replace .Version "v" "" }}-{{ .Os }}-{{ .Arch }}`
	if cfg.GoReleaser.NameTemplate != "" {
		nameTemplate = cfg.GoReleaser.NameTemplate
	}

	if cfg.GoReleaser.Files == nil {
		cfg.GoReleaser.Files = &[]string{
			"CHANGELOG.md",
			"LICENSE",
			"README.md",
		}
	}

	binaryName := cfg.Binaries[0].Name
	if cfg.GoReleaser.BinaryName != "" {
		binaryName = cfg.GoReleaser.BinaryName
	}

	var (
		metadataURL *url.URL
		err         error
	)
	if !strings.HasPrefix(cfg.Metadata.URL, "https://github.com/") {
		metadataURL, err = url.Parse(cfg.Metadata.URL)
		if err != nil {
			logg.Fatal("Metadata.URL is not a parsable URL: %w", err)
		}
	}

	t := template.Must(template.New("goreleaser.yaml").Parse(goreleaserTemplate))
	goreleaserFile := must.Return(os.OpenFile(".goreleaser.yaml", os.O_WRONLY|os.O_TRUNC, 0666))
	defer goreleaserFile.Close()
	must.Succeed(t.Execute(goreleaserFile, map[string]any{
		"nameTemplate": nameTemplate,
		"format":       cfg.GoReleaser.Format,
		"files":        cfg.GoReleaser.Files,
		"binaryName":   binaryName,
		"binName":      cfg.Binaries[0].Name,
		"ldflags":      cfg.Golang.LdFlags,
		"fromPackage":  cfg.Binaries[0].FromPackage,
		"githubDomain": metadataURL,
	}))

	// Remove renamed file
	must.Succeed(os.RemoveAll(".goreleaser.yml"))
	must.Succeed(os.WriteFile("RELEASE.md", []byte(releaseMD), 0666))
}
