// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build requirefips

package file_integrity

// fileParserFor returns the name of the file parser for the given field. It is
// statically defined to catch parser collisions at compile time.
var fileParserFor = map[string]string{
	"file.elf.sections":               "executable_object",
	"file.elf.sections.name":          "executable_object",
	"file.elf.sections.physical_size": "executable_object",
	"file.elf.sections.virtual_size":  "executable_object",
	"file.elf.sections.entropy":       "executable_object",
	"file.elf.sections.var_entropy":   "executable_object",
	"file.elf.go_stripped":            "executable_object",

	"file.macho.sections":               "executable_object",
	"file.macho.sections.name":          "executable_object",
	"file.macho.sections.physical_size": "executable_object",
	"file.macho.sections.virtual_size":  "executable_object",
	"file.macho.sections.entropy":       "executable_object",
	"file.macho.sections.var_entropy":   "executable_object",
	"file.macho.go_stripped":            "executable_object",

	"file.pe.sections":               "executable_object",
	"file.pe.sections.name":          "executable_object",
	"file.pe.sections.physical_size": "executable_object",
	"file.pe.sections.virtual_size":  "executable_object",
	"file.pe.sections.entropy":       "executable_object",
	"file.pe.sections.var_entropy":   "executable_object",
	"file.pe.go_stripped":            "executable_object",
}
