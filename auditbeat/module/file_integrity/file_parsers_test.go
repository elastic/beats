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

package file_integrity

import (
	"reflect"
	"testing"

	"github.com/elastic/elastic-agent-libs/config"
)

func TestFileParsers(t *testing.T) {
	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"paths":        []string{"/usr/bin"},
		"file_parsers": []string{"file.elf.sections", `/\.pe\./`},
	})
	if err != nil {
		t.Fatal(err)
	}

	c := defaultConfig
	if err := cfg.Unpack(&c); err != nil {
		t.Fatal(err)
	}

	wantParserNames := map[string]bool{
		"executable_object": true,
	}
	wantFields := map[string]bool{
		"file.elf.sections":                    true,
		"file.pe.sections":                     true,
		"file.pe.sections.name":                true,
		"file.pe.sections.physical_size":       true,
		"file.pe.sections.virtual_size":        true,
		"file.pe.sections.entropy":             true,
		"file.pe.sections.var_entropy":         true,
		"file.pe.import_hash":                  true,
		"file.pe.imphash":                      true,
		"file.pe.imports":                      true,
		"file.pe.imports_names_entropy":        true,
		"file.pe.imports_names_var_entropy":    true,
		"file.pe.go_import_hash":               true,
		"file.pe.go_imports":                   true,
		"file.pe.go_imports_names_entropy":     true,
		"file.pe.go_imports_names_var_entropy": true,
		"file.pe.go_stripped":                  true,
	}

	gotParserNames, gotFields := parserNamesAndFields(c)
	if !reflect.DeepEqual(gotParserNames, wantParserNames) {
		t.Errorf("unexpected parser name set: got:%v want:%v", gotParserNames, wantParserNames)
	}
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Errorf("unexpected fields set: got:%v want:%v", gotFields, wantFields)
	}
}
