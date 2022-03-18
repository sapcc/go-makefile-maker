/******************************************************************************
*
*  Copyright 2020 SAP SE
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
