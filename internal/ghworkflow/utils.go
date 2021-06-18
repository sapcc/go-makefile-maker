// Copyright 2021 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ghworkflow

import "strings"

// makeMultilineYAMLString adds \n to the strings and joins them.
// yaml.Marshal() takes care of the rest.
func makeMultilineYAMLString(in []string) string {
	out := strings.Join(in, "\n")
	// We need the \n at the end to ensure that yaml.Marshal() puts the right
	// chomping indicator; i.e. `|` instead of `|-`.
	if len(in) > 1 {
		out += "\n"
	}
	return out
}

// stringsJoinAndTrimSpace uses a single space as separator.
func stringsJoinAndTrimSpace(in []string) string {
	return strings.TrimSpace(strings.Join(in, " "))
}
