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

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
)

func TestNumberOfRoutingShards(t *testing.T) {

	beatVersion := "6.1.0"
	beatName := "testbeat"
	config := Config{}

	// Test it exists in 6.1
	ver := common.MustNewVersion("6.1.0")
	template, err := New(beatVersion, beatName, *ver, config, false)
	assert.NoError(t, err)

	data := template.Generate(nil, nil)
	shards, err := data.GetValue("settings.index.number_of_routing_shards")
	assert.NoError(t, err)

	assert.Equal(t, 30, shards.(int))

	// Test it does not exist in 6.0
	ver = common.MustNewVersion("6.0.0")
	template, err = New(beatVersion, beatName, *ver, config, false)
	assert.NoError(t, err)

	data = template.Generate(nil, nil)
	shards, err = data.GetValue("settings.index.number_of_routing_shards")
	assert.Error(t, err)
	assert.Equal(t, nil, shards)
}

func TestNumberOfRoutingShardsOverwrite(t *testing.T) {

	beatVersion := "6.1.0"
	beatName := "testbeat"
	config := Config{
		Settings: Settings{
			Index: map[string]interface{}{"number_of_routing_shards": 5},
		},
	}

	// Test it exists in 6.1
	ver := common.MustNewVersion("6.1.0")
	template, err := New(beatVersion, beatName, *ver, config, false)
	assert.NoError(t, err)

	data := template.Generate(nil, nil)
	shards, err := data.GetValue("settings.index.number_of_routing_shards")
	assert.NoError(t, err)

	assert.Equal(t, 5, shards.(int))
}

func TestNew(t *testing.T) {
	v, version, name, esVersion := "7.0.0", common.MustNewVersion("7.0.0"), "beatName", common.MustNewVersion("7.0.0-alpha1")
	data := []struct {
		name string
		c    Config
	}{
		{
			name: "test beat replacement",
			c: Config{
				Name:     "beat-%{[beat.version]}",
				Pattern:  "beat-%{[beat.name]}",
				Settings: Settings{Index: map[string]interface{}{}, Source: map[string]interface{}{}},
			},
		},
		{

			name: "test agent replacement",
			c: Config{
				Name:    "beat-%{[agent.version]}",
				Pattern: "beat-%{[agent.name]}",
				Settings: Settings{
					Index:  map[string]interface{}{"lifecycle": map[string]interface{}{}},
					Source: map[string]interface{}{}},
			},
		},
		{
			name: "test observer replacement",
			c: Config{
				Name:    "beat-%{[observer.version]}",
				Pattern: "beat-%{[observer.name]}",
				Settings: Settings{
					Index: map[string]interface{}{"lifecycle": map[string]interface{}{
						"rollover_alias": "beat-%{[observer.name]}",
						"name":           "beat-%{[observer.name]}",
					}},
					Source: map[string]interface{}{}},
			},
		},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			template, err := New(v, name, *esVersion, d.c, false)
			require.NoError(t, err)
			expected := &Template{name: "beat-7.0.0", pattern: "beat-beatName", beatVersion: *version, esVersion: *esVersion, config: d.c}
			assert.Equal(t, expected, template)
		})
	}

}

func TestAppendFields(t *testing.T) {
	tests := []struct {
		fields       common.Fields
		appendFields common.Fields
		error        bool
	}{
		{
			fields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "b",
						},
					},
				},
			},
			appendFields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "c",
						},
					},
				},
			},
			error: false,
		},
		{
			fields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "b",
						},
						common.Field{
							Name: "c",
						},
					},
				},
			},
			appendFields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "c",
						},
					},
				},
			},
			error: true,
		},
		{
			fields: common.Fields{
				common.Field{
					Name: "a",
				},
			},
			appendFields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "c",
						},
					},
				},
			},
			error: true,
		},
		{
			fields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "c",
						},
					},
				},
			},
			appendFields: common.Fields{
				common.Field{
					Name: "a",
				},
			},
			error: true,
		},
	}

	for _, test := range tests {
		_, err := appendFields(test.fields, test.appendFields)
		if test.error {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
