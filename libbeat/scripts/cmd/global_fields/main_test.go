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

package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/generator/fields"
)

type testcase struct {
	fieldsPath string
	files      []*fields.YmlFile
}

var (
	beatsPath          = filepath.Join("testdata")
	expectedFieldFiles = []*fields.YmlFile{
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "module", "module1", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "module", "module1", "set1", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "module", "module1", "set2", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "module", "module2", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "module", "module2", "set1", "_meta", "fields.yml"),
			Indent: 8,
		},
	}
)

// TestCollectModuleFiles validates if the required files are collected
func TestCollectModuleFiles(t *testing.T) {
	fieldFiles, err := fields.CollectModuleFiles(filepath.Join(beatsPath, "module"))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, reflect.DeepEqual(fieldFiles, expectedFieldFiles))
}
