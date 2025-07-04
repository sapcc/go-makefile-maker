// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"bytes"
	"os"
	"regexp"
	"strings"

	"github.com/sapcc/go-bits/must"
	"golang.org/x/mod/modfile"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/util"
)

// SetGoVersionInGoMod updates the go directive in the go.mod file unless some exceptions are met.
//
// If the major and minor version match the new version and the current patch version is 0, updating is skipped.
// This is requriered as some Kubernetes libraries mandate this through the peer dependency chain
// and tools like govulncheck and controller-gen refuse to work with this mismatch and mandate to run go mod tidy.
func SetGoVersionInGoMod() {
	// read and parse the go. mod file early. This is done deliberately outside of Scan() to only parse what we need in this function.
	modFileBytes := must.Return(os.ReadFile(ModFilename))
	goVersionSlice := strings.Split(core.DefaultGoVersion, ".")
	modFile := must.Return(modfile.Parse(ModFilename, modFileBytes, nil))
	currentGoVersionSlice := strings.Split(modFile.Go.Version, ".")

	// join the major and minor part together for easy comparison
	currentGoVersion := strings.Join(currentGoVersionSlice[:len(currentGoVersionSlice)-1], ".")
	goVersion := strings.Join(goVersionSlice[:len(goVersionSlice)-1], ".")

	// if set patch version is 0 and the other parts match, don't do anything
	if currentGoVersion == goVersion && len(currentGoVersionSlice) == 3 && currentGoVersionSlice[len(currentGoVersionSlice)-1] == "0" {
		return
	}

	// otherwise update the version
	rgx := regexp.MustCompile(`go \d\.\d+(\.\d+)?`)
	modFileBytesReplaced := rgx.ReplaceAll(modFileBytes, []byte("go "+goVersion))
	if !bytes.Equal(modFileBytes, modFileBytesReplaced) {
		must.Succeed(util.WriteFile(ModFilename, modFileBytesReplaced))
	}
}
