// Copyright 2020 SAP SE
// SPDX-License-Identifier: Apache-2.0

package makefile

import (
	"strings"
	"testing"
)

const indentedWithSpaces = `
foo:
    echo one
    echo two
      echo three

bar:
  echo ten
    echo twenty
  	echo thirty

qux:
	echo this already has tabs

.PHONY: foo bar
`

const indentedWithTabs = `
foo:
	echo one
	echo two
	  echo three

bar:
	echo ten
	  echo twenty
		echo thirty

qux:
	echo this already has tabs

.PHONY: foo bar
`

func TestFixRuleIndentation(t *testing.T) {
	actual := FixRuleIndentation(indentedWithSpaces)
	if actual != indentedWithTabs {
		t.Error("unexpected result")
		for _, line := range strings.SplitAfter(actual, "\n") {
			t.Logf("output line: %q", line)
		}
	}
}
