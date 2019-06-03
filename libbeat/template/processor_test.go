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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/mapping"
)

func TestProcessor(t *testing.T) {
	falseVar := false
	trueVar := true
	p := &Processor{EsVersion: *common.MustNewVersion("7.0.0")}
	migrationP := &Processor{EsVersion: *common.MustNewVersion("7.0.0"), Migration: true}
	pEsVersion2 := &Processor{EsVersion: *common.MustNewVersion("2.0.0")}
	pEsVersion64 := &Processor{EsVersion: *common.MustNewVersion("6.4.0")}
	pEsVersion63 := &Processor{EsVersion: *common.MustNewVersion("6.3.6")}

	tests := []struct {
		output   common.MapStr
		expected common.MapStr
	}{
		{
			output:   p.other(&mapping.Field{Type: "long"}),
			expected: common.MapStr{"type": "long"},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float"}),
			expected: common.MapStr{
				"type":           "scaled_float",
				"scaling_factor": 1000,
			},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float", ScalingFactor: 100}),
			expected: common.MapStr{
				"type":           "scaled_float",
				"scaling_factor": 100,
			},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float"}, common.MapStr{scalingFactorKey: 0}),
			expected: common.MapStr{
				"type":           "scaled_float",
				"scaling_factor": 1000,
			},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float"}, common.MapStr{"someKey": 10}),
			expected: common.MapStr{
				"type":           "scaled_float",
				"scaling_factor": 1000,
			},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float"}, common.MapStr{scalingFactorKey: 10}),
			expected: common.MapStr{
				"type":           "scaled_float",
				"scaling_factor": 10,
			},
		},
		{
			output:   pEsVersion2.scaledFloat(&mapping.Field{Type: "scaled_float"}),
			expected: common.MapStr{"type": "float"},
		},
		{
			output: p.object(&mapping.Field{Type: "object", Enabled: &falseVar}),
			expected: common.MapStr{
				"type":    "object",
				"enabled": false,
			},
		},
		{
			output: p.integer(&mapping.Field{Type: "long", CopyTo: "hello.world"}),
			expected: common.MapStr{
				"type":    "long",
				"copy_to": "hello.world",
			},
		},
		{
			output:   p.array(&mapping.Field{Type: "array"}),
			expected: common.MapStr{},
		},
		{
			output:   p.array(&mapping.Field{Type: "array", ObjectType: "text"}),
			expected: common.MapStr{"type": "text"},
		},
		{
			output:   p.array(&mapping.Field{Type: "array", Index: &falseVar, ObjectType: "keyword"}),
			expected: common.MapStr{"index": false, "type": "keyword"},
		},
		{
			output:   pEsVersion64.alias(&mapping.Field{Type: "alias", AliasPath: "a.b"}),
			expected: common.MapStr{"path": "a.b", "type": "alias"},
		},
		{
			// alias unsupported in ES < 6.4
			output:   pEsVersion63.alias(&mapping.Field{Type: "alias", AliasPath: "a.b"}),
			expected: nil,
		},
		{
			output: p.object(&mapping.Field{Type: "object", Enabled: &falseVar}),
			expected: common.MapStr{
				"type":    "object",
				"enabled": false,
			},
		},
		{
			output: p.text(&mapping.Field{Type: "text", Analyzer: "autocomplete"}),
			expected: common.MapStr{
				"type":     "text",
				"analyzer": "autocomplete",
				"norms":    false,
			},
		},
		{
			output: p.text(&mapping.Field{Type: "text", Analyzer: "autocomplete", Norms: true}),
			expected: common.MapStr{
				"type":     "text",
				"analyzer": "autocomplete",
			},
		},
		{
			output: p.text(&mapping.Field{Type: "text", SearchAnalyzer: "standard", Norms: true}),
			expected: common.MapStr{
				"type":            "text",
				"search_analyzer": "standard",
			},
		},
		{
			output: p.text(&mapping.Field{Type: "text", Analyzer: "autocomplete", SearchAnalyzer: "standard", Norms: true}),
			expected: common.MapStr{
				"type":            "text",
				"analyzer":        "autocomplete",
				"search_analyzer": "standard",
			},
		},
		{
			output: p.text(&mapping.Field{Type: "text", MultiFields: mapping.Fields{mapping.Field{Name: "raw", Type: "keyword"}}, Norms: true}),
			expected: common.MapStr{
				"type": "text",
				"fields": common.MapStr{
					"raw": common.MapStr{
						"type":         "keyword",
						"ignore_above": 1024,
					},
				},
			},
		},
		{
			output: p.keyword(&mapping.Field{Type: "keyword", MultiFields: mapping.Fields{mapping.Field{Name: "analyzed", Type: "text", Norms: true}}}),
			expected: common.MapStr{
				"type":         "keyword",
				"ignore_above": 1024,
				"fields": common.MapStr{
					"analyzed": common.MapStr{
						"type": "text",
					},
				},
			},
		},
		{
			output: p.keyword(&mapping.Field{Type: "keyword", IgnoreAbove: 256}),
			expected: common.MapStr{
				"type":         "keyword",
				"ignore_above": 256,
			},
		},
		{
			output: p.keyword(&mapping.Field{Type: "keyword", IgnoreAbove: -1}),
			expected: common.MapStr{
				"type": "keyword",
			},
		},
		{
			output: p.keyword(&mapping.Field{Type: "keyword"}),
			expected: common.MapStr{
				"type":         "keyword",
				"ignore_above": 1024,
			},
		},
		{
			output: p.text(&mapping.Field{Type: "text", MultiFields: mapping.Fields{
				mapping.Field{Name: "raw", Type: "keyword"},
				mapping.Field{Name: "indexed", Type: "text"},
			}, Norms: true}),
			expected: common.MapStr{
				"type": "text",
				"fields": common.MapStr{
					"raw": common.MapStr{
						"type":         "keyword",
						"ignore_above": 1024,
					},
					"indexed": common.MapStr{
						"type":  "text",
						"norms": false,
					},
				},
			},
		},
		{
			output: p.text(&mapping.Field{Type: "text", MultiFields: mapping.Fields{
				mapping.Field{Name: "raw", Type: "keyword"},
				mapping.Field{Name: "indexed", Type: "text"},
			}, Norms: true}),
			expected: common.MapStr{
				"type": "text",
				"fields": common.MapStr{
					"raw": common.MapStr{
						"type":         "keyword",
						"ignore_above": 1024,
					},
					"indexed": common.MapStr{
						"type":  "text",
						"norms": false,
					},
				},
			},
		},
		{
			output: p.object(&mapping.Field{Dynamic: mapping.DynamicType{Value: false}}),
			expected: common.MapStr{
				"dynamic": false, "type": "object",
			},
		},
		{
			output: p.object(&mapping.Field{Dynamic: mapping.DynamicType{Value: true}}),
			expected: common.MapStr{
				"dynamic": true, "type": "object",
			},
		},
		{
			output: p.object(&mapping.Field{Dynamic: mapping.DynamicType{Value: "strict"}}),
			expected: common.MapStr{
				"dynamic": "strict", "type": "object",
			},
		},
		{
			output: p.other(&mapping.Field{Type: "long", Index: &falseVar}),
			expected: common.MapStr{
				"type": "long", "index": false,
			},
		},
		{
			output: p.other(&mapping.Field{Type: "text", Index: &trueVar}),
			expected: common.MapStr{
				"type": "text", "index": true,
			},
		},
		{
			output: p.other(&mapping.Field{Type: "long", DocValues: &falseVar}),
			expected: common.MapStr{
				"type": "long", "doc_values": false,
			},
		},
		{
			output: p.other(&mapping.Field{Type: "double", DocValues: &falseVar}),
			expected: common.MapStr{
				"type": "double", "doc_values": false,
			},
		},
		{
			output: p.other(&mapping.Field{Type: "text", DocValues: &trueVar}),
			expected: common.MapStr{
				"type": "text", "doc_values": true,
			},
		},
		{
			output:   p.alias(&mapping.Field{Type: "alias", AliasPath: "a.c", MigrationAlias: false}),
			expected: common.MapStr{"path": "a.c", "type": "alias"},
		},
		{
			output:   p.alias(&mapping.Field{Type: "alias", AliasPath: "a.d", MigrationAlias: true}),
			expected: nil,
		},
		{
			output:   migrationP.alias(&mapping.Field{Type: "alias", AliasPath: "a.e", MigrationAlias: false}),
			expected: common.MapStr{"path": "a.e", "type": "alias"},
		},
		{
			output:   migrationP.alias(&mapping.Field{Type: "alias", AliasPath: "a.f", MigrationAlias: true}),
			expected: common.MapStr{"path": "a.f", "type": "alias"},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.output)
	}
}

func TestDynamicTemplates(t *testing.T) {
	p := &Processor{}
	tests := []struct {
		field    mapping.Field
		expected []common.MapStr
	}{
		{
			field: mapping.Field{
				Type: "object", ObjectType: "keyword",
				Name: "context",
			},
			expected: []common.MapStr{
				common.MapStr{
					"context": common.MapStr{
						"mapping":            common.MapStr{"type": "keyword"},
						"match_mapping_type": "string",
						"path_match":         "context.*",
					},
				},
			},
		},
		{
			field: mapping.Field{
				Type: "object", ObjectType: "long", ObjectTypeMappingType: "futuretype",
				Path: "language", Name: "english",
			},
			expected: []common.MapStr{
				common.MapStr{
					"language.english": common.MapStr{
						"mapping":            common.MapStr{"type": "long"},
						"match_mapping_type": "futuretype",
						"path_match":         "language.english.*",
					},
				},
			},
		},
		{
			field: mapping.Field{
				Type: "object", ObjectType: "long", ObjectTypeMappingType: "*",
				Path: "language", Name: "english",
			},
			expected: []common.MapStr{
				common.MapStr{
					"language.english": common.MapStr{
						"mapping":            common.MapStr{"type": "long"},
						"match_mapping_type": "*",
						"path_match":         "language.english.*",
					},
				},
			},
		},
		{
			field: mapping.Field{
				Type: "object", ObjectType: "long",
				Path: "language", Name: "english",
			},
			expected: []common.MapStr{
				common.MapStr{
					"language.english": common.MapStr{
						"mapping":            common.MapStr{"type": "long"},
						"match_mapping_type": "long",
						"path_match":         "language.english.*",
					},
				},
			},
		},
		{
			field: mapping.Field{
				Type: "object", ObjectType: "text",
				Path: "language", Name: "english",
			},
			expected: []common.MapStr{
				common.MapStr{
					"language.english": common.MapStr{
						"mapping":            common.MapStr{"type": "text"},
						"match_mapping_type": "string",
						"path_match":         "language.english.*",
					},
				},
			},
		},
		{
			field: mapping.Field{
				Type: "object", ObjectType: "scaled_float",
				Name: "core.*.pct",
			},
			expected: []common.MapStr{
				common.MapStr{
					"core.*.pct": common.MapStr{
						"mapping": common.MapStr{
							"type":           "scaled_float",
							"scaling_factor": defaultScalingFactor,
						},
						"match_mapping_type": "*",
						"path_match":         "core.*.pct",
					},
				},
			},
		},
		{
			field: mapping.Field{
				Type: "object", ObjectType: "scaled_float",
				Name: "core.*.pct", ScalingFactor: 100, ObjectTypeMappingType: "float",
			},
			expected: []common.MapStr{
				common.MapStr{
					"core.*.pct": common.MapStr{
						"mapping": common.MapStr{
							"type":           "scaled_float",
							"scaling_factor": 100,
						},
						"match_mapping_type": "float",
						"path_match":         "core.*.pct",
					},
				},
			},
		},
		{
			field: mapping.Field{
				Type: "object", ObjectTypeParams: []mapping.ObjectTypeCfg{
					{ObjectType: "float", ObjectTypeMappingType: "float"},
					{ObjectType: "boolean"},
					{ObjectType: "scaled_float", ScalingFactor: 10000},
				},
				Name: "context",
			},
			expected: []common.MapStr{
				common.MapStr{
					"context": common.MapStr{
						"mapping":            common.MapStr{"type": "float"},
						"match_mapping_type": "float",
						"path_match":         "context.*",
					},
				},
				common.MapStr{
					"context": common.MapStr{
						"mapping":            common.MapStr{"type": "boolean"},
						"match_mapping_type": "boolean",
						"path_match":         "context.*",
					},
				},
				common.MapStr{
					"context": common.MapStr{
						"mapping":            common.MapStr{"type": "scaled_float", "scaling_factor": 10000},
						"match_mapping_type": "*",
						"path_match":         "context.*",
					},
				},
			},
		},
	}

	for _, numericType := range []string{"byte", "double", "float", "long", "short", "boolean"} {
		gen := struct {
			field    mapping.Field
			expected []common.MapStr
		}{
			field: mapping.Field{
				Type: "object", ObjectType: numericType,
				Name: "somefield", ObjectTypeMappingType: "long",
			},
			expected: []common.MapStr{
				common.MapStr{
					"somefield": common.MapStr{
						"mapping": common.MapStr{
							"type": numericType,
						},
						"match_mapping_type": "long",
						"path_match":         "somefield.*",
					},
				},
			},
		}
		tests = append(tests, gen)
	}

	for _, test := range tests {
		dynamicTemplates = nil
		p.object(&test.field)
		assert.Equal(t, test.expected, dynamicTemplates)
	}
}

func TestPropertiesCombine(t *testing.T) {
	// Test common fields are combined even if they come from different objects
	fields := mapping.Fields{
		mapping.Field{
			Name: "test",
			Type: "group",
			Fields: mapping.Fields{
				mapping.Field{
					Name: "one",
					Type: "text",
				},
			},
		},
		mapping.Field{
			Name: "test",
			Type: "group",
			Fields: mapping.Fields{
				mapping.Field{
					Name: "two",
					Type: "text",
				},
			},
		},
	}

	output := common.MapStr{}
	version, err := common.NewVersion("6.0.0")
	if err != nil {
		t.Fatal(err)
	}

	p := Processor{EsVersion: *version}
	err = p.Process(fields, "", output)
	if err != nil {
		t.Fatal(err)
	}

	v1, err := output.GetValue("test.properties.one")
	if err != nil {
		t.Fatal(err)
	}
	v2, err := output.GetValue("test.properties.two")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, v1, common.MapStr{"type": "text", "norms": false})
	assert.Equal(t, v2, common.MapStr{"type": "text", "norms": false})
}

func TestProcessNoName(t *testing.T) {
	// Test common fields are combined even if they come from different objects
	fields := mapping.Fields{
		mapping.Field{
			Fields: mapping.Fields{
				mapping.Field{
					Name: "one",
					Type: "text",
				},
			},
		},
		mapping.Field{
			Name: "test",
			Type: "group",
			Fields: mapping.Fields{
				mapping.Field{
					Name: "two",
					Type: "text",
				},
			},
		},
	}

	output := common.MapStr{}
	version, err := common.NewVersion("6.0.0")
	if err != nil {
		t.Fatal(err)
	}

	p := Processor{EsVersion: *version}
	err = p.Process(fields, "", output)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure fields without a name are skipped during template generation
	expectedOutput := common.MapStr{
		"test": common.MapStr{
			"properties": common.MapStr{
				"two": common.MapStr{
					"norms": false,
					"type":  "text",
				},
			},
		},
	}

	assert.Equal(t, expectedOutput, output)
}
