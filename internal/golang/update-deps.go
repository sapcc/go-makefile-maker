// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

// AutoupdateDependencies will run `go get -u $MODULEPATH@latest` for each
// direct dependency matching `cfg.AutoupdateableDepsRx`.
func AutoupdateDependencies(sr ScanResult, cfg core.GolangConfiguration) {
	// we will know whether any dependencies are actually updated by looking at go.mod
	modTimeOld := must.Return(getModTime(ModFilename))

	// update dependencies
	for _, dep := range sr.GoDirectDependencies {
		if !cfg.AutoupdateableDepsRx.MatchString(dep.Path) {
			continue
		}

		must.Succeed(runGo("get", "-u", dep.Path+"@latest"))
	}

	// if we updated something, run a tidy/verify/vendor
	// (`go mod vendor` is the only one of these that's strictly required, but let's be thorough)
	modTimeNew := must.Return(getModTime(ModFilename))
	if !modTimeOld.Equal(modTimeNew) {
		must.Succeed(runGo("mod", "tidy"))
		must.Succeed(runGo("mod", "verify"))
		if cfg.EnableVendoring {
			must.Succeed(runGo("mod", "vendor"))
		}
	}
}

func runGo(args ...string) error {
	argsJoined := strings.Join(args, " ")
	logg.Debug("-> running go %s", argsJoined)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("could not run `go %s`: %w", argsJoined, err)
	}
	return nil
}

func getModTime(path string) (time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime(), nil
}
