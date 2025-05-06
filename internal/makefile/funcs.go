// SPDX-FileCopyrightText: Copyright 2020 SAP SE
// SPDX-License-Identifier: Apache-2.0

package makefile

import (
	"regexp"
	"strings"
)

var isRecipeRx = regexp.MustCompile(`^\s+\S`)

// FixRuleIndentation takes a Makefile snippet from our YAML input, and ensures
// that recipes (the shell commands inside a Makefile rule) are correctly
// indented with tabs instead of spaces. This is important because YAML requires
// spaces for indentation, so writing out the tabs correctly in the Makefile is
// cumbersome and error-prone.
func FixRuleIndentation(in string) string {
	var out strings.Builder
	var currentRecipeLines []string

	for _, line := range strings.SplitAfter(in, "\n") {
		// when inside a recipe, collect all lines belonging to it first
		if isRecipeRx.MatchString(line) {
			currentRecipeLines = append(currentRecipeLines, line)
			continue
		}

		// when not inside a recipe, return this line unchanged, but first transform
		// the collected recipe (if any)
		if len(currentRecipeLines) > 0 {
			for _, line := range fixRecipeIndentation(currentRecipeLines) {
				out.WriteString(line)
			}
			currentRecipeLines = nil
		}
		out.WriteString(line)
	}

	return out.String()
}

// Helper for FixRuleIndentation(): Replace consistent leading whitespace on the
// given set of lines with a single tab.
func fixRecipeIndentation(lines []string) []string {
	prefixLen := 0
	for prefixLen+1 < len(lines[0]) {
		longerPrefix := lines[0][0 : prefixLen+1]
		if strings.TrimSpace(longerPrefix) == "" && isCommonPrefix(lines, longerPrefix) {
			prefixLen++
		} else {
			break
		}
	}

	commonPrefix := lines[0][0:prefixLen]
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, "\t"+strings.TrimPrefix(line, commonPrefix))
	}
	return out
}

func isCommonPrefix(lines []string, prefix string) bool {
	for _, line := range lines {
		if !strings.HasPrefix(line, prefix) {
			return false
		}
	}
	return true
}
