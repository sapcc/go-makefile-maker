// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package reuse

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
	"github.com/sapcc/go-makefile-maker/internal/util"
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
		logg.Debug("-> running go-mod-tidy")
		cmd := exec.Command("go", "mod", "tidy")
		output, err := cmd.CombinedOutput()
		if err != nil {
			logg.Fatal(string(output))
		}

		_ = must.Return(exec.LookPath("go-licence-detector"))

		// Create temporary file for output
		tmpOutput := must.Return(os.CreateTemp("", "go-makefile-maker-output-*"))
		defer os.Remove(tmpOutput.Name())
		tmpOutput.Close() // Close so external command can write to it

		logg.Debug("-> running go-licence-detector")
		cmd = exec.Command("sh", "-c", //nolint:gosec // Command is run by the user
			// On Linux we would just use /dev/stdout but that does not work on ✨ macOS ✨
			fmt.Sprintf("go list -m -mod=readonly -json all | go-licence-detector -includeIndirect -rules .license-scan-rules.json -overrides .license-scan-overrides.jsonl -depsOut %s -depsTemplate /dev/fd/3", tmpOutput.Name()))
		cmd.ExtraFiles = []*os.File{tmpGLDT}
		output, err = cmd.CombinedOutput() // Capture output only in case of an error
		if err != nil {
			logg.Fatal(string(output))
		}

		// Read the output from temp file
		output, err = os.ReadFile(tmpOutput.Name())
		if err != nil {
			logg.Fatal(err.Error())
		}

		for dependencyString := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
			dependencyString = strings.TrimSpace(dependencyString)
			if dependencyString == "" {
				continue
			}
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

	must.Succeed(util.WriteFileFromTemplate("REUSE.toml", reuseTOMLTemplate, map[string]any{
		"Annotations": allAnnotations,
		"PackageName": filepath.Base(cfg.Metadata.URL),
		"URL":         cfg.Metadata.URL,
	}))
}
