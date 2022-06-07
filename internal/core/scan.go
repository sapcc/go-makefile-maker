/******************************************************************************
*
*  Copyright 2022 SAP SE
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

package core

import (
	"errors"
	"os"

	"golang.org/x/mod/modfile"

	"github.com/sapcc/go-makefile-maker/internal/util"
)

// ScanResult contains data obtained through a scan of the configuration files
// in the repository. At the moment, only `go.mod` is scanned.
type ScanResult struct {
	ModulePath string // from "module" directive in go.mod, e.g. "github.com/foo/bar"
	GoVersion  string // from "go" directive in go.mod, e.g. "1.17"
}

func Scan() (ScanResult, error) {
	modFilename := "go.mod"
	modFileBytes, err := os.ReadFile(modFilename)
	if err != nil {
		return ScanResult{}, err
	}
	modFile, err := modfile.Parse(modFilename, modFileBytes, nil)
	if err != nil {
		return ScanResult{}, err
	}

	return ScanResult{
		GoVersion:  modFile.Go.Version,
		ModulePath: modFile.Module.Mod.Path,
	}, nil
}

//MustModulePath reads the ModulePath field, but fails if it is empty.
func (sr ScanResult) MustModulePath() string {
	if sr.ModulePath == "" {
		util.Must(errors.New("could not find module path from go.mod file, make sure it is defined"))
	}
	return sr.ModulePath
}
