// Copyright 2022 SAP SE
// SPDX-License-Identifier: Apache-2.0

package renovate

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/sapcc/go-bits/must"

	"github.com/sapcc/go-makefile-maker/internal/core"
	"github.com/sapcc/go-makefile-maker/internal/golang"
)

type constraints struct {
	Go string `json:"go,omitempty"`
}

type config struct {
	Schema                                     string             `json:"$schema"`
	Extends                                    []string           `json:"extends"`
	Assignees                                  []string           `json:"assignees,omitempty"`
	CommitMessageAction                        string             `json:"commitMessageAction,omitempty"`
	Constraints                                *constraints       `json:"constraints,omitempty"`
	DependencyDashboardOSVVulnerabilitySummary string             `json:"dependencyDashboardOSVVulnerabilitySummary,omitempty"`
	OsvVulnerabilityAlerts                     bool               `json:"osvVulnerabilityAlerts,omitempty"`
	PostUpdateOptions                          []string           `json:"postUpdateOptions,omitempty"`
	PackageRules                               []core.PackageRule `json:"packageRules,omitempty"`
	CustomManagers                             []interface{}      `json:"customManagers,omitempty"`
	PrHourlyLimit                              int                `json:"prHourlyLimit"`
	Schedule                                   []string           `json:"schedule,omitempty"`
	SemanticCommits                            string             `json:"semanticCommits,omitempty"`
}

func RenderConfig(cfgRenovate core.RenovateConfig, scanResult golang.ScanResult, url string, isApplicationRepo bool) {
	isGoMakefileMakerRepo := scanResult.ModulePath == "github.com/sapcc/go-makefile-maker"
	isInternalRenovate := strings.HasPrefix(url, "https://github.wdf.sap.corp")

	// Our default rule is to have Renovate send us PRs once a week. (More
	// frequent PRs become too overwhelming with the sheer amount of repos that
	// we manage.) Friday works well for this because when we merge, we can let
	// the changes simmer in QA over the weekend, and then have high confidence
	// when we deploy these updates on Monday.
	schedule := "before 8am on Friday"
	if isInternalRenovate {
		schedule = "on Friday"
	}
	// However, for pure library repos, we do the PRs on Thursday instead, so
	// that the dependency updates in these library repos trickle down into the
	// application repos without an extra week of delay.
	if !isApplicationRepo {
		schedule = "before 8am on Thursday"
		if isInternalRenovate {
			schedule = "on Thursday"
		}
	}

	cfg := config{
		Schema: "https://docs.renovatebot.com/renovate-schema.json",
		Extends: []string{
			"config:recommended",
			"default:pinDigestsDisabled",
			"mergeConfidence:all-badges",
		},
		Assignees: cfgRenovate.Assignees,
		// CommitMessageAction is the verb that appears at the start of Renovate's
		// commit messages (and therefore, PR titles). The default value is "Update".
		// We choose something more specific because some of us have filter rules
		// in their mail client to separate Renovate PRs from other PRs.
		CommitMessageAction:                        "Renovate: Update",
		DependencyDashboardOSVVulnerabilitySummary: "all",
		OsvVulnerabilityAlerts:                     true,
		PrHourlyLimit:                              0,
		Schedule:                                   []string{schedule},
		SemanticCommits:                            "disabled",
	}

	if scanResult.GoVersion != "" {
		cfg.Constraints = &constraints{
			Go: cfgRenovate.GoVersion,
		}
		cfg.PostUpdateOptions = append([]string{"gomodTidy", "gomodUpdateImportPaths"}, cfg.PostUpdateOptions...)

		// Default package rules.
		//
		// NOTE: When changing this list, please also adjust the documentation for
		// default package rules in the README.
		cfg.PackageRules = append(cfg.PackageRules, core.PackageRule{
			MatchPackageNames: []string{"golang"},
			AllowedVersions:   cfgRenovate.GoVersion + ".x",
		})

		// combine and automerge all dependencies under github.com/sapcc/
		cfg.PackageRules = append(cfg.PackageRules, core.PackageRule{
			MatchPackageNames: []string{`/^github\.com\/sapcc\/.*/`},
			GroupName:         "github.com/sapcc",
			AutoMerge:         true,
		})

		// combine all dependencies not under github.com/sapcc/
		cfg.PackageRules = append(cfg.PackageRules, core.PackageRule{
			MatchPackageNames: []string{`!/^github\.com\/sapcc\/.*/`, `/.*/`},
			GroupName:         "External dependencies",
			AutoMerge:         false,
		})
	}

	// Only enable Dockerfile and github-actions updates for go-makefile-maker itself.
	if isGoMakefileMakerRepo {
		cfg.Extends = append(cfg.Extends, "docker:enableMajor", "regexManagers:dockerfileVersions")
	} else {
		cfg.Extends = append(cfg.Extends, "docker:disable")
	}
	hasK8sIOPkgs := false
	for _, v := range scanResult.GoDirectDependencies {
		if strings.HasPrefix(v.Path, "k8s.io/") {
			hasK8sIOPkgs = true
		}
	}
	if hasK8sIOPkgs {
		cfg.PackageRules = append(cfg.PackageRules, core.PackageRule{
			MatchPackageNames: []string{`/^k8s.io\//`},
			// Since our clusters use k8s v1.26 and k8s has a support policy of -/+ 1 minor version we set the allowedVersions to `0.27.x`.
			// k8s.io/* deps use v0.x.y instead of v1.x.y therefore we use 0.x instead of 1.x.
			// Ref: https://docs.renovatebot.com/configuration-options/#allowedversions
			AllowedVersions: "0.28.x",
			// ^ NOTE: When bumping this version, also adjust the rendition of this rule in the README appropriately.
		})
	}

	// Custom package rules specified in config.
	//
	// Renovate will evaluate all packageRules and not stop once it gets a first match
	// therefore the packageRules should be in the order of importance so that user
	// defined rules can override settings from earlier rules.
	cfg.PackageRules = append(cfg.PackageRules, cfgRenovate.PackageRules...)

	// CustomManagers specified in config.
	//
	// With customManagers using regex you can configure Renovate so it finds dependencies
	// that are not detected by its other built-in package managers.
	cfg.CustomManagers = append(cfg.CustomManagers, cfgRenovate.CustomManagers...)

	must.Succeed(os.MkdirAll(".github", 0750))
	must.Succeed(os.RemoveAll("renovate.json"))
	f := must.Return(os.Create(".github/renovate.json"))

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // in order to preserve `<` in allowedVersions field
	must.Succeed(encoder.Encode(cfg))

	must.Succeed(f.Close())
}
