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

package main

//Configuration is the data structure that we read from the input file.
type Configuration struct {
	Verbatim       string                   `yaml:"verbatim"`
	VariableValues map[string]string        `yaml:"variables"`
	Binaries       []BinaryConfiguration    `yaml:"binaries"`
	Coverage       CoverageConfiguration    `yaml:"coverageTest"`
	Vendoring      VendoringConfiguration   `yaml:"vendoring"`
	StaticCheck    StaticCheckConfiguration `yaml:"staticCheck"`
}

//Variable returns the value of this variable if it's overridden in the config,
//or the default value otherwise.
func (c Configuration) Variable(name string, defaultValue string) string {
	value, exists := c.VariableValues[name]
	if exists {
		return value
	}
	return defaultValue
}

//BinaryConfiguration appears in type Configuration.
type BinaryConfiguration struct {
	Name        string `yaml:"name"`
	FromPackage string `yaml:"fromPackage"`
	InstallTo   string `yaml:"installTo"`
}

//CoverageConfiguration appears in type Configuration.
type CoverageConfiguration struct {
	Only   string `yaml:"only"`
	Except string `yaml:"except"`
}

//VendoringConfiguration appears in type Configuration.
type VendoringConfiguration struct {
	Enabled bool `yaml:"enabled"`
}

//StaticCheckConfiguration appears in type Configuration.
type StaticCheckConfiguration struct {
	GolangciLint bool `yaml:"golangciLint"`
}
