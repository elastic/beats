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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/mapping"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestProcessor(t *testing.T) {
	falseVar := false
	trueVar := true
	p := &Processor{EsVersion: *common.MustNewVersion("7.0.0")}
	migrationP := &Processor{EsVersion: *common.MustNewVersion("7.0.0"), Migration: true}
	pEsVersion76 := &Processor{EsVersion: *common.MustNewVersion("7.6.0")}

	tests := []struct {
		output   mapstr.M
		expected mapstr.M
	}{
		{
			output:   p.other(&mapping.Field{Type: "long"}),
			expected: mapstr.M{"type": "long"},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float"}),
			expected: mapstr.M{
				"type":           "scaled_float",
				"scaling_factor": 1000,
			},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float", ScalingFactor: 100}),
			expected: mapstr.M{
				"type":           "scaled_float",
				"scaling_factor": 100,
			},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float"}, mapstr.M{scalingFactorKey: 0}),
			expected: mapstr.M{
				"type":           "scaled_float",
				"scaling_factor": 1000,
			},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float"}, mapstr.M{"someKey": 10}),
			expected: mapstr.M{
				"type":           "scaled_float",
				"scaling_factor": 1000,
			},
		},
		{
			output: p.scaledFloat(&mapping.Field{Type: "scaled_float"}, mapstr.M{scalingFactorKey: 10}),
			expected: mapstr.M{
				"type":           "scaled_float",
				"scaling_factor": 10,
			},
		},
		{
			output: p.object(&mapping.Field{Type: "object", Enabled: &falseVar}),
			expected: mapstr.M{
				"type":    "object",
				"enabled": false,
			},
		},
		{
			output: p.integer(&mapping.Field{Type: "long", CopyTo: "hello.world"}),
			expected: mapstr.M{
				"type":    "long",
				"copy_to": "hello.world",
			},
		},
		{
			output:   p.array(&mapping.Field{Type: "array"}),
			expected: mapstr.M{},
		},
		{
			output:   p.array(&mapping.Field{Type: "array", ObjectType: "text"}),
			expected: mapstr.M{"type": "text"},
		},
		{
			output:   p.array(&mapping.Field{Type: "array", Index: &falseVar, ObjectType: "keyword"}),
			expected: mapstr.M{"index": false, "type": "keyword"},
		},
		{
			output: p.object(&mapping.Field{Type: "object", Enabled: &falseVar}),
			expected: mapstr.M{
				"type":    "object",
				"enabled": false,
			},
		},
		{
			output: fieldsOnly(p.text(&mapping.Field{Type: "text", Analyzer: mapping.Analyzer{Name: "autocomplete"}}, nil)),
			expected: mapstr.M{
				"type":     "text",
				"analyzer": "autocomplete",
				"norms":    false,
			},
		},
		{
			output: fieldsOnly(p.text(&mapping.Field{Type: "text", Analyzer: mapping.Analyzer{Name: "autocomplete"}, Norms: true}, nil)),
			expected: mapstr.M{
				"type":     "text",
				"analyzer": "autocomplete",
			},
		},
		{
			output: fieldsOnly(p.text(&mapping.Field{Type: "text", SearchAnalyzer: mapping.Analyzer{Name: "standard"}, Norms: true}, nil)),
			expected: mapstr.M{
				"type":            "text",
				"search_analyzer": "standard",
			},
		},
		{
			output: fieldsOnly(p.text(&mapping.Field{Type: "text", Analyzer: mapping.Analyzer{Name: "autocomplete"}, SearchAnalyzer: mapping.Analyzer{Name: "standard"}, Norms: true}, nil)),
			expected: mapstr.M{
				"type":            "text",
				"analyzer":        "autocomplete",
				"search_analyzer": "standard",
			},
		},
		{
			output: fieldsOnly(p.text(&mapping.Field{Type: "text", MultiFields: mapping.Fields{mapping.Field{Name: "raw", Type: "keyword"}}, Norms: true}, nil)),
			expected: mapstr.M{
				"type": "text",
				"fields": mapstr.M{
					"raw": mapstr.M{
						"type":         "keyword",
						"ignore_above": 1024,
					},
				},
			},
		},
		{
			output: p.keyword(&mapping.Field{Type: "keyword", MultiFields: mapping.Fields{mapping.Field{Name: "analyzed", Type: "text", Norms: true}}}, nil),
			expected: mapstr.M{
				"type":         "keyword",
				"ignore_above": 1024,
				"fields": mapstr.M{
					"analyzed": mapstr.M{
						"type": "text",
					},
				},
			},
		},
		{
			output: p.keyword(&mapping.Field{Type: "keyword", IgnoreAbove: 256}, nil),
			expected: mapstr.M{
				"type":         "keyword",
				"ignore_above": 256,
			},
		},
		{
			output: p.keyword(&mapping.Field{Type: "keyword", IgnoreAbove: -1}, nil),
			expected: mapstr.M{
				"type": "keyword",
			},
		},
		{
			output: p.keyword(&mapping.Field{Type: "keyword"}, nil),
			expected: mapstr.M{
				"type":         "keyword",
				"ignore_above": 1024,
			},
		},
		{
			output: fieldsOnly(p.text(&mapping.Field{Type: "text", MultiFields: mapping.Fields{
				mapping.Field{Name: "raw", Type: "keyword"},
				mapping.Field{Name: "indexed", Type: "text"},
			}, Norms: true}, nil)),
			expected: mapstr.M{
				"type": "text",
				"fields": mapstr.M{
					"raw": mapstr.M{
						"type":         "keyword",
						"ignore_above": 1024,
					},
					"indexed": mapstr.M{
						"type":  "text",
						"norms": false,
					},
				},
			},
		},
		{
			output: fieldsOnly(p.text(&mapping.Field{Type: "text", MultiFields: mapping.Fields{
				mapping.Field{Name: "raw", Type: "keyword"},
				mapping.Field{Name: "indexed", Type: "text"},
			}, Norms: true}, nil)),
			expected: mapstr.M{
				"type": "text",
				"fields": mapstr.M{
					"raw": mapstr.M{
						"type":         "keyword",
						"ignore_above": 1024,
					},
					"indexed": mapstr.M{
						"type":  "text",
						"norms": false,
					},
				},
			},
		},
		{
			output: p.object(&mapping.Field{Dynamic: mapping.DynamicType{Value: false}}),
			expected: mapstr.M{
				"dynamic": false, "type": "object",
			},
		},
		{
			output: p.object(&mapping.Field{Dynamic: mapping.DynamicType{Value: true}}),
			expected: mapstr.M{
				"dynamic": true, "type": "object",
			},
		},
		{
			output: p.object(&mapping.Field{Dynamic: mapping.DynamicType{Value: "strict"}}),
			expected: mapstr.M{
				"dynamic": "strict", "type": "object",
			},
		},
		{
			output: p.other(&mapping.Field{Type: "long", Index: &falseVar}),
			expected: mapstr.M{
				"type": "long", "index": false,
			},
		},
		{
			output: p.other(&mapping.Field{Type: "text", Index: &trueVar}),
			expected: mapstr.M{
				"type": "text", "index": true,
			},
		},
		{
			output: p.other(&mapping.Field{Type: "long", DocValues: &falseVar}),
			expected: mapstr.M{
				"type": "long", "doc_values": false,
			},
		},
		{
			output: p.other(&mapping.Field{Type: "double", DocValues: &falseVar}),
			expected: mapstr.M{
				"type": "double", "doc_values": false,
			},
		},
		{
			output: p.other(&mapping.Field{Type: "text", DocValues: &trueVar}),
			expected: mapstr.M{
				"type": "text", "doc_values": true,
			},
		},
		{
			output:   p.alias(&mapping.Field{Type: "alias", AliasPath: "a.c", MigrationAlias: false}),
			expected: mapstr.M{"path": "a.c", "type": "alias"},
		},
		{
			output:   p.alias(&mapping.Field{Type: "alias", AliasPath: "a.d", MigrationAlias: true}),
			expected: nil,
		},
		{
			output:   migrationP.alias(&mapping.Field{Type: "alias", AliasPath: "a.e", MigrationAlias: false}),
			expected: mapstr.M{"path": "a.e", "type": "alias"},
		},
		{
			output:   migrationP.alias(&mapping.Field{Type: "alias", AliasPath: "a.f", MigrationAlias: true}),
			expected: mapstr.M{"path": "a.f", "type": "alias"},
		},
		{
			output:   p.histogram(&mapping.Field{Type: "histogram"}),
			expected: nil,
		},
		{
			output:   pEsVersion76.histogram(&mapping.Field{Type: "histogram"}),
			expected: mapstr.M{"type": "histogram"},
		},
		{
			// "p" has EsVersion 7.0.0; field metadata requires ES 7.6.0+
			output:   p.other(&mapping.Field{Type: "long", MetricType: "gauge", Unit: "nanos"}),
			expected: mapstr.M{"type": "long"},
		},
		{
			output:   pEsVersion76.other(&mapping.Field{Type: "long", MetricType: "gauge"}),
			expected: mapstr.M{"type": "long", "meta": mapstr.M{"metric_type": "gauge"}},
		},
		{
			output:   pEsVersion76.other(&mapping.Field{Type: "long", Unit: "nanos"}),
			expected: mapstr.M{"type": "long", "meta": mapstr.M{"unit": "nanos"}},
		},
		{
			output:   pEsVersion76.other(&mapping.Field{Type: "long", MetricType: "gauge", Unit: "nanos"}),
			expected: mapstr.M{"type": "long", "meta": mapstr.M{"metric_type": "gauge", "unit": "nanos"}},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.output)
	}
}

func fieldsOnly(f mapstr.M, _, _ mapping.Analyzer) mapstr.M {
	return f
}

func TestDynamicTemplates(t *testing.T) {
	tests := []struct {
		field    mapping.Field
		expected []mapstr.M
	}{
		{
			field: mapping.Field{
				Type: "object", ObjectType: "keyword",
				Name: "context",
			},
			expected: []mapstr.M{
				mapstr.M{
					"context": mapstr.M{
						"mapping":            mapstr.M{"type": "keyword"},
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
			expected: []mapstr.M{
				mapstr.M{
					"language.english": mapstr.M{
						"mapping":            mapstr.M{"type": "long"},
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
			expected: []mapstr.M{
				mapstr.M{
					"language.english": mapstr.M{
						"mapping":            mapstr.M{"type": "long"},
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
			expected: []mapstr.M{
				mapstr.M{
					"language.english": mapstr.M{
						"mapping":            mapstr.M{"type": "long"},
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
			expected: []mapstr.M{
				mapstr.M{
					"language.english": mapstr.M{
						"mapping":            mapstr.M{"type": "text"},
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
			expected: []mapstr.M{
				mapstr.M{
					"core.*.pct": mapstr.M{
						"mapping": mapstr.M{
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
			expected: []mapstr.M{
				mapstr.M{
					"core.*.pct": mapstr.M{
						"mapping": mapstr.M{
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
			expected: []mapstr.M{
				{
					"context_float": mapstr.M{
						"mapping":            mapstr.M{"type": "float"},
						"match_mapping_type": "float",
						"path_match":         "context.*",
					},
				},
				{
					"context_boolean": mapstr.M{
						"mapping":            mapstr.M{"type": "boolean"},
						"match_mapping_type": "boolean",
						"path_match":         "context.*",
					},
				},
				{
					"context_*": mapstr.M{
						"mapping":            mapstr.M{"type": "scaled_float", "scaling_factor": 10000},
						"match_mapping_type": "*",
						"path_match":         "context.*",
					},
				},
			},
		},
		{
			field: mapping.Field{
				Name:            "dynamic_histogram",
				Type:            "histogram",
				DynamicTemplate: true,
			},
			expected: []mapstr.M{
				{
					"dynamic_histogram": mapstr.M{
						"mapping": mapstr.M{"type": "histogram"},
					},
				},
			},
		},
	}

	for _, numericType := range []string{"byte", "double", "float", "long", "short", "boolean"} {
		gen := struct {
			field    mapping.Field
			expected []mapstr.M
		}{
			field: mapping.Field{
				Type: "object", ObjectType: numericType,
				Name: "somefield", ObjectTypeMappingType: "long",
			},
			expected: []mapstr.M{
				mapstr.M{
					"somefield": mapstr.M{
						"mapping": mapstr.M{
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
		output := make(mapstr.M)
		analyzers := make(mapstr.M)
		p := &Processor{EsVersion: *common.MustNewVersion("8.0.0")}
		err := p.Process(mapping.Fields{
			test.field,
			test.field, // should not be added twice
		}, &fieldState{Path: test.field.Path}, output, analyzers)
		require.NoError(t, err)
		assert.Equal(t, test.expected, p.dynamicTemplates)
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

	output := mapstr.M{}
	analyzers := mapstr.M{}
	version, err := common.NewVersion("6.0.0")
	if err != nil {
		t.Fatal(err)
	}

	p := Processor{EsVersion: *version}
	err = p.Process(fields, nil, output, analyzers)
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

	assert.Equal(t, v1, mapstr.M{"type": "text", "norms": false})
	assert.Equal(t, v2, mapstr.M{"type": "text", "norms": false})
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

	output := mapstr.M{}
	analyzers := mapstr.M{}
	version, err := common.NewVersion("6.0.0")
	if err != nil {
		t.Fatal(err)
	}

	p := Processor{EsVersion: *version}
	err = p.Process(fields, nil, output, analyzers)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure fields without a name are skipped during template generation
	expectedOutput := mapstr.M{
		"test": mapstr.M{
			"properties": mapstr.M{
				"two": mapstr.M{
					"norms": false,
					"type":  "text",
				},
			},
		},
	}

	assert.Equal(t, expectedOutput, output)
}

func TestProcessDefaultField(t *testing.T) {
	// NOTE: This package stores global state. It must be cleared before this test.
	defaultFields = nil

	var (
		enableDefaultField  = true
		disableDefaultField = false
	)

	fields := mapping.Fields{
		// By default foo will be excluded in default_field.
		mapping.Field{
			Name: "foo",
			Type: "keyword",
		},
		// bar is explicitly set to be included in default_field.
		mapping.Field{
			Name:         "bar",
			Type:         "keyword",
			DefaultField: &enableDefaultField,
		},
		// baz is explicitly set to be excluded from default_field.
		mapping.Field{
			Name:         "baz",
			Type:         "keyword",
			DefaultField: &disableDefaultField,
		},
		mapping.Field{
			Name:         "nested",
			Type:         "group",
			DefaultField: &enableDefaultField,
			Fields: mapping.Fields{
				mapping.Field{
					Name: "bar",
					Type: "keyword",
				},
			},
		},
		// The nested group is disabled default_field but one of the children
		// has explicitly requested to be included.
		mapping.Field{
			Name:         "nested",
			Type:         "group",
			DefaultField: &disableDefaultField,
			Fields: mapping.Fields{
				mapping.Field{
					Name:         "foo",
					Type:         "keyword",
					DefaultField: &enableDefaultField,
				},
				mapping.Field{
					Name: "baz",
					Type: "keyword",
				},
			},
		},
		// Check that multi_fields are correctly stored in defaultFields.
		mapping.Field{
			Name:         "qux",
			Type:         "keyword",
			DefaultField: &enableDefaultField,
			MultiFields: []mapping.Field{
				{
					Name: "text",
					Type: "text",
				},
			},
		},
		mapping.Field{
			Name:         "bouba",
			Type:         "keyword",
			DefaultField: &disableDefaultField,
			MultiFields: []mapping.Field{
				{
					Name:         "text",
					Type:         "text",
					DefaultField: &enableDefaultField,
				},
			},
		},
		mapping.Field{
			Name:         "kiki",
			Type:         "keyword",
			DefaultField: &enableDefaultField,
			MultiFields: []mapping.Field{
				{
					Name:         "text",
					Type:         "text",
					DefaultField: &disableDefaultField,
				},
			},
		},
		// Ensure that text_only_keyword fields can be added to default_field
		mapping.Field{
			Name:         "a_match_only_text_field",
			Type:         "match_only_text",
			DefaultField: &enableDefaultField,
		},
		// Ensure that wildcard fields can be added to default_field
		mapping.Field{
			Name:         "a_wildcard_field",
			Type:         "wildcard",
			DefaultField: &enableDefaultField,
		},
	}

	version, err := common.NewVersion("7.0.0")
	if err != nil {
		t.Fatal(err)
	}

	p := Processor{EsVersion: *version}
	output := mapstr.M{}
	analyzers := mapstr.M{}
	if err = p.Process(fields, nil, output, analyzers); err != nil {
		t.Fatal(err)
	}

	expectedFields := []string{
		"a_match_only_text_field",
		"a_wildcard_field",
		"bar",
		"nested.bar",
		"nested.foo",
		"qux",
		"qux.text",
		"bouba.text",
		"kiki",
	}
	sort.Strings(defaultFields)
	sort.Strings(expectedFields)
	assert.Equal(t, expectedFields, defaultFields)
}

func TestProcessWildcardOSS(t *testing.T) {
	// Test common fields are combined even if they come from different objects
	fields := mapping.Fields{
		mapping.Field{
			Name: "test",
			Type: "group",
			Fields: mapping.Fields{
				mapping.Field{
					Name: "one",
					Type: "wildcard",
				},
			},
		},
	}

	output := mapstr.M{}
	analyzers := mapstr.M{}
	version, err := common.NewVersion("8.0.0")
	if err != nil {
		t.Fatal(err)
	}

	p := Processor{EsVersion: *version}
	err = p.Process(fields, nil, output, analyzers)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure fields without a name are skipped during template generation
	expectedOutput := mapstr.M{
		"test": mapstr.M{
			"properties": mapstr.M{
				"one": mapstr.M{
					"ignore_above": 1024,
					"type":         "keyword",
				},
			},
		},
	}

	assert.Equal(t, expectedOutput, output)
}

func TestProcessWildcardElastic(t *testing.T) {
	for _, test := range []struct {
		title    string
		fields   mapping.Fields
		expected mapstr.M
	}{
		{
			title: "default",
			fields: mapping.Fields{
				mapping.Field{
					Name: "test",
					Type: "group",
					Fields: mapping.Fields{
						mapping.Field{
							Name: "one",
							Type: "wildcard",
						},
					},
				},
			},
			expected: mapstr.M{
				"test": mapstr.M{
					"properties": mapstr.M{
						"one": mapstr.M{
							"type": "wildcard",
						},
					},
				},
			},
		},
		{
			title: "explicit ignore_above",
			fields: mapping.Fields{
				mapping.Field{
					Name: "test",
					Type: "group",
					Fields: mapping.Fields{
						mapping.Field{
							Name:        "one",
							Type:        "wildcard",
							IgnoreAbove: 4096,
						},
					},
				},
			},
			expected: mapstr.M{
				"test": mapstr.M{
					"properties": mapstr.M{
						"one": mapstr.M{
							"ignore_above": 4096,
							"type":         "wildcard",
						},
					},
				},
			},
		},
	} {
		t.Run(test.title, func(t *testing.T) {
			output := mapstr.M{}
			analyzers := mapstr.M{}
			version, err := common.NewVersion("8.0.0")
			if err != nil {
				t.Fatal(err)
			}
			p := Processor{EsVersion: *version, ElasticLicensed: true}
			err = p.Process(test.fields, nil, output, analyzers)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.expected, output)
		})
	}
}

func TestProcessWildcardPreSupport(t *testing.T) {
	// Test common fields are combined even if they come from different objects
	fields := mapping.Fields{
		mapping.Field{
			Name: "test",
			Type: "group",
			Fields: mapping.Fields{
				mapping.Field{
					Name: "one",
					Type: "wildcard",
				},
			},
		},
	}

	output := mapstr.M{}
	analyzers := mapstr.M{}
	version, err := common.NewVersion("7.8.0")
	if err != nil {
		t.Fatal(err)
	}

	p := Processor{EsVersion: *version, ElasticLicensed: true}
	err = p.Process(fields, nil, output, analyzers)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure fields without a name are skipped during template generation
	expectedOutput := mapstr.M{
		"test": mapstr.M{
			"properties": mapstr.M{
				"one": mapstr.M{
					"ignore_above": 1024,
					"type":         "keyword",
				},
			},
		},
	}

	assert.Equal(t, expectedOutput, output)
}

func TestProcessNestedSupport(t *testing.T) {
	fields := mapping.Fields{
		mapping.Field{
			Name: "test",
			Type: "nested",
			Fields: mapping.Fields{
				mapping.Field{
					Name: "one",
					Type: "keyword",
				},
			},
		},
	}

	output := mapstr.M{}
	analyzers := mapstr.M{}
	version, err := common.NewVersion("7.8.0")
	if err != nil {
		t.Fatal(err)
	}

	p := Processor{EsVersion: *version, ElasticLicensed: true}
	err = p.Process(fields, nil, output, analyzers)
	if err != nil {
		t.Fatal(err)
	}

	expectedOutput := mapstr.M{
		"test": mapstr.M{
			"type": "nested",
			"properties": mapstr.M{
				"one": mapstr.M{
					"ignore_above": 1024,
					"type":         "keyword",
				},
			},
		},
	}

	assert.Equal(t, expectedOutput, output)
}

func TestProcessNestedSupportNoSubfields(t *testing.T) {
	fields := mapping.Fields{
		mapping.Field{
			Name: "test",
			Type: "nested",
		},
	}

	output := mapstr.M{}
	analyzers := mapstr.M{}
	version, err := common.NewVersion("7.8.0")
	if err != nil {
		t.Fatal(err)
	}

	p := Processor{EsVersion: *version, ElasticLicensed: true}
	err = p.Process(fields, nil, output, analyzers)
	if err != nil {
		t.Fatal(err)
	}

	expectedOutput := mapstr.M{
		"test": mapstr.M{
			"type": "nested",
		},
	}

	assert.Equal(t, expectedOutput, output)
}
