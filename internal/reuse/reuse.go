// SPDX-FileCopyrightText: Copyright 2025 SAP SE
// SPDX-License-Identifier: Apache-2.0

package reuse

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"text/template"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

var (
	//go:embed reuse.toml.tmpl
	reuseTOMLTemplate string
	//go:embed go-licence-detector.tmpl
	goLicenceDetectorTemplate []byte
)

func RenderConfig(cfg core.Configuration, sr golang.ScanResult) {
	// If disabled, the REUSE.toml file should not be overridden.
	// This is useful if the project needs additional information in
	// the REUSE.toml file, e.g., specific disclaimers.
	if !cfg.Reuse.IsEnabled() {
		return
	}

	allAnnotations := slices.Clone(cfg.Reuse.Annotations)

	if cfg.Golang.EnableVendoring {
		tmpGLDT := must.Return(os.CreateTemp("", "go-makefile-maker-*"))
		defer os.Remove(tmpGLDT.Name())

		_ = must.Return(tmpGLDT.Write(goLicenceDetectorTemplate))

		_ = must.Return(tmpGLDT.Seek(0, 0))

		// otherwise we might miss some direct dependencies which is really strange...
		cmd := exec.Command("go", "mod", "tidy")
		output, err := cmd.CombinedOutput()
		if err != nil {
			logg.Fatal(string(output))
		}

		_ = must.Return(exec.LookPath("go-licence-detector"))

		cmd = exec.Command("sh", "-c",
			"go list -m -mod=readonly -json all | go-licence-detector -includeIndirect -rules .license-scan-rules.json -overrides .license-scan-overrides.jsonl -depsOut /dev/stdout -depsTemplate /dev/fd/3")
		cmd.ExtraFiles = []*os.File{tmpGLDT}
		output, err = cmd.CombinedOutput()
		if err != nil {
			logg.Fatal(string(output))
		}

		for _, dependencyString := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			var dep struct {
				Name    string `json:"name"`
				License string `json:"license"`
			}
			must.Succeed(json.Unmarshal([]byte(dependencyString), &dep))
			allAnnotations = append(allAnnotations, core.ReuseAnnotation{
				Paths:                 []string{fmt.Sprintf("vendor/%s/**", dep.Name)},
				Precedence:            "aggregate",
				SPDXFileCopyrightText: "Other",
				SPDXLicenseIdentifier: dep.License,
			})
		}
	}

	t := template.Must(template.New("REUSE.toml").Parse(reuseTOMLTemplate))
	var buf bytes.Buffer
	must.Succeed(t.Execute(&buf, map[string]any{"Annotations": allAnnotations}))
	must.Succeed(os.WriteFile("REUSE.toml", buf.Bytes(), 0o666))
}
