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

//go:build !integration
// +build !integration

package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type testTemplate struct {
	t    *testing.T
	tmpl *Template
	data mapstr.M
}

func TestNumberOfRoutingShards(t *testing.T) {
	const notPresent = 0 // setting missing indicator
	const settingKey = "number_of_routing_shards"
	const fullKey = "template.settings.index." + settingKey

	cases := map[string]struct {
		esVersion string
		set       int
		want      int
	}{
		"Do not set by default for ES 7.x": {
			esVersion: "7.0.0",
			want:      notPresent,
		},
		"Still configurable for ES 7.x": {
			esVersion: "7.0.0",
			set:       1024,
			want:      1024,
		},
		"Do not set with current version": {
			want: notPresent,
		},
		"Still configurable with current version": {
			set:  1024,
			want: 1024,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			beatVersion := getVersion("")
			esVersion := getVersion(test.esVersion)

			indexSettings := map[string]interface{}{}
			if test.set > 0 {
				indexSettings[settingKey] = test.set
			}

			template := createTestTemplate(t, beatVersion, esVersion, TemplateConfig{
				Settings: TemplateSettings{
					Index: indexSettings,
				},
			})

			if test.want == notPresent {
				template.AssertMissing(fullKey)
			} else {
				template.Assert(fullKey, test.want)
			}
		})
	}
}

func TestTemplate(t *testing.T) {
	currentVersion := getVersion("")
	info := beat.Info{Beat: "testbeat", Version: currentVersion}

	t.Run("for ES 7.x", func(t *testing.T) {
		template := createTestTemplate(t, currentVersion, "7.10.0", DefaultConfig(info))
		template.Assert("index_patterns", []string{"testbeat-" + currentVersion})
		template.Assert("template.mappings._meta", mapstr.M{"beat": "testbeat", "version": currentVersion})
		template.Assert("template.settings.index.max_docvalue_fields_search", 200)
	})

	t.Run("for ES 8.x", func(t *testing.T) {
		template := createTestTemplate(t, currentVersion, "8.0.0", DefaultConfig(info))
		template.Assert("index_patterns", []string{"testbeat-" + currentVersion})
		template.Assert("template.mappings._meta", mapstr.M{"beat": "testbeat", "version": currentVersion})
		template.Assert("template.settings.index.max_docvalue_fields_search", 200)
	})
}

func createTestTemplate(t *testing.T, beatVersion, esVersion string, config TemplateConfig) *testTemplate {
	beatVersion = getVersion(beatVersion)
	esVersion = getVersion(esVersion)
	ver := common.MustNewVersion(esVersion)
	template, err := New(beatVersion, "testbeat", false, *ver, config, false)
	if err != nil {
		t.Fatalf("Failed to create the template: %+v", err)
	}

	return &testTemplate{t: t, tmpl: template, data: template.Generate(nil, nil, nil)}
}

func (t *testTemplate) Has(path string) bool {
	t.t.Helper()
	has, err := t.data.HasKey(path)
	if err != nil && err != common.ErrKeyNotFound {
		serialized, _ := json.MarshalIndent(t.data, "", "    ")
		t.t.Fatalf("error accessing '%v': %v\ntemplate: %s", path, err, serialized)
	}
	return has
}

func (t *testTemplate) Get(path string) interface{} {
	t.t.Helper()
	val, err := t.data.GetValue(path)
	if err != nil {
		serialized, _ := json.MarshalIndent(t.data, "", "    ")
		t.t.Fatalf("error accessing '%v': %v\ntemplate: %s", path, err, serialized)
	}
	return val
}

func (t *testTemplate) AssertMissing(path string) {
	t.t.Helper()
	if t.Has(path) {
		t.t.Fatalf("Expected '%v' to be missing", path)
	}
}

func (t *testTemplate) Assert(path string, val interface{}) {
	t.t.Helper()
	assert.Equal(t.t, val, t.Get(path))
}

func getVersion(in string) string {
	if in == "" {
		return version.GetDefaultVersion()
	}
	return in
}
