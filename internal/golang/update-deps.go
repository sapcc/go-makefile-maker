// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/majewsky/gg/option"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

// AutoupdateConfiguration contains runtime configuration for func AutoupdateDependencies().
type AutoupdateConfiguration struct {
	// List of paths to go.mod files.
	ExtraDependencySets []string
}

// AutoupdateDependencies will:
//   - run `go get -u $MODULEPATH@latest` for each direct dependency matching `cfg.AutoupdateDependencies.ModuleNameRx`, and
//   - run `go get $MODULEPATH@$VERSION` for each existing dependency with a newer version in `aucfg.ExtraDependencySets`.
func AutoupdateDependencies(cfg core.GolangConfiguration, aucfg AutoupdateConfiguration) {
	// we will know whether any dependencies are actually updated by looking at go.mod
	modTimeOld := must.Return(getModTime(ModFilename))

	// helper function: machinery to re-read the modfile after every change; this is necessary for two reasons:
	// 1. we cannot use a previous golang.ScanResult because we run before Scan()
	//    on account of this autoupdate possibly changing the scan result
	// 2. every dependency update can also change what's in the go.mod file, so we need to re-read it after every dependency update;
	//    otherwise we might work with outdated version information in the next step and accidentally downgrade instead of upgrading
	ourModfile := None[*modfile.File]()
	getOurModfile := func() *modfile.File {
		if mf, ok := ourModfile.Unpack(); ok {
			return mf
		}
		buf := must.Return(os.ReadFile(ModFilename))
		mf, err := modfile.Parse(ModFilename, buf, nil)
		if err != nil {
			logg.Fatal("could not parse %s after dependency updates: %s", ModFilename, err.Error())
		}
		ourModfile = Some(mf) // cache the parsed file until another dependency update is done
		return mf
	}

	// update direct dependencies matched by golang.autoupdateDependencies.matchModule setting
	if cfg.AutoupdateDependencies.ModuleNameRx != "" {
		for _, v := range getOurModfile().Require {
			if v.Indirect {
				continue
			}
			if cfg.AutoupdateDependencies.ModuleNameRx.MatchString(v.Mod.Path) {
				must.Succeed(runGo("get", "-u", v.Mod.Path+"@latest"))
				ourModfile = None[*modfile.File]()
			}
		}
	}

	// helper function: return the version of a specific dependency in our go.mod file (or None if we don't have this dependency)
	getOurVersion := func(modPath string) Option[string] {
		for _, req := range getOurModfile().Require {
			if req.Mod.Path == modPath {
				return Some(req.Mod.Version)
			}
		}
		return None[string]()
	}

	// update dependencies matched by any of the ExtraDependencySets
	for _, modfilePath := range aucfg.ExtraDependencySets {
		buf := must.Return(os.ReadFile(modfilePath))
		mf, err := modfile.Parse(modfilePath, buf, nil)
		if err != nil {
			logg.Fatal("could not parse `--additional-autoupdateable-dependencies %s`: %s", modfilePath, err.Error())
		}

		for _, req := range mf.Require {
			logg.Debug("-> considering whether %s@%s in %s is a worthy update", req.Mod.Path, req.Mod.Version, modfilePath)
			ourVersion, ok := getOurVersion(req.Mod.Path).Unpack()
			if !ok {
				continue
			}
			if semver.Compare(ourVersion, req.Mod.Version) < 0 {
				// NOTE: This does not use `-u` to avoid introducing unexpected (i.e. unvetted) updates of transitive dependencies.
				must.Succeed(runGo("get", req.Mod.Path+"@"+req.Mod.Version))
				ourModfile = None[*modfile.File]() // invalidate cache of go.mod contents
			}
		}
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
