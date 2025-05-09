// SPDX-FileCopyrightText: Copyright 2025 SAP SE
// SPDX-License-Identifier: Apache-2.0

package reuse

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

//go:embed go-licence-detector.tmpl
var goLicenceDetectorTemplate embed.FS

var reuseTemplate = strings.ReplaceAll(`# SPDX-FileCopyrightText: SAP SE
# SPDX-License-Identifier: Apache-2.0
version = 1

[[annotations]]
path = [
	".github/CODEOWNERS",
	".github/renovate.json",
	".gitignore",
	".license-scan-overrides.jsonl",
	".license-scan-rules.json",
	"go.mod",
	"go.sum",
	"Makefile.maker.yaml",
	"vendor/modules.txt",
]
SPDX-FileCopyrightText = "SAP SE"
SPDX-License-Identifier = "Apache-2.0"
%[1]s`, "\t", "  ")

func RenderConfig(cfg core.Configuration, sr golang.ScanResult) {
	// If disabled, the REUSE.toml file should not be overridden.
	// This is useful if the project needs additional information in
	// the REUSE.toml file, e.g., specific disclaimers.
	if cfg.Reuse.Enabled != nil && !*cfg.Reuse.Enabled {
		return
	}

	var appendConfig string

	if cfg.Golang.EnableVendoring {
		gLDT := must.Return(goLicenceDetectorTemplate.Open("go-licence-detector.tmpl"))
		defer gLDT.Close()

		tmpGLDT := must.Return(os.CreateTemp("", "go-makefile-maker-*"))
		defer os.Remove(tmpGLDT.Name())

		_ = must.Return(io.Copy(tmpGLDT, gLDT))

		_ = must.Return(tmpGLDT.Seek(0, 0))

		// otherwise we might miss some direct dependencies which is really strange...
		cmd := exec.Command("go", "mod", "tidy")
		must.Return(cmd.Output())

		cmd = exec.Command("sh", "-c",
			"go list -m -mod=readonly -json all | go-licence-detector -includeIndirect -rules .license-scan-rules.json -overrides .license-scan-overrides.jsonl -depsOut /dev/stdout -depsTemplate /dev/fd/3")
		cmd.ExtraFiles = []*os.File{tmpGLDT}
		output := must.Return(cmd.Output())

		type dependency struct {
			Name    string `json:"name"`
			License string `json:"license"`
		}
		dependencyTemplate := `
	[[annotations]]
	path = [ "vendor/%[1]s/**" ]
	precedence = "aggregate"
	SPDX-FileCopyrightText = "Other"
	SPDX-License-Identifier = "%[2]s"
	`

		for _, dependencyString := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			var aDependency dependency
			must.Succeed(json.Unmarshal([]byte(dependencyString), &aDependency))
			appendConfig += fmt.Sprintf(dependencyTemplate, aDependency.Name, aDependency.License)
		}
	}

	reuseFile := fmt.Sprintf(reuseTemplate, appendConfig)
	must.Succeed(os.WriteFile("REUSE.toml", []byte(reuseFile), 0o666))
}
