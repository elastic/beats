package template

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestProcessor(t *testing.T) {
	esVersion2, err := common.NewVersion("2.0.0")
	assert.NoError(t, err)

	falseVar := false
	trueVar := true
	p := &Processor{}
	pEsVersion2 := &Processor{EsVersion: *esVersion2}

	tests := []struct {
		output   common.MapStr
		expected common.MapStr
	}{
		{
			output:   p.other(&common.Field{Type: "long"}),
			expected: common.MapStr{"type": "long"},
		},
		{
			output: p.scaledFloat(&common.Field{Type: "scaled_float"}),
			expected: common.MapStr{
				"type":           "scaled_float",
				"scaling_factor": 1000,
			},
		},
		{
			output: p.scaledFloat(&common.Field{Type: "scaled_float", ScalingFactor: 100}),
			expected: common.MapStr{
				"type":           "scaled_float",
				"scaling_factor": 100,
			},
		},
		{
			output:   pEsVersion2.scaledFloat(&common.Field{Type: "scaled_float"}),
			expected: common.MapStr{"type": "float"},
		},
		{
			output: p.object(&common.Field{Type: "object", Enabled: &falseVar}),
			expected: common.MapStr{
				"type":    "object",
				"enabled": false,
			},
		},
		{
			output: p.integer(&common.Field{Type: "long", CopyTo: "hello.world"}),
			expected: common.MapStr{
				"type":    "long",
				"copy_to": "hello.world",
			},
		},
		{
			output:   p.array(&common.Field{Type: "array"}),
			expected: common.MapStr{},
		},
		{
			output:   p.array(&common.Field{Type: "array", ObjectType: "text"}),
			expected: common.MapStr{"type": "text"},
		},
		{
			output:   p.array(&common.Field{Type: "array", Index: &falseVar, ObjectType: "keyword"}),
			expected: common.MapStr{"index": false, "type": "keyword"},
		},
		{
			output: p.object(&common.Field{Type: "object", Enabled: &falseVar}),
			expected: common.MapStr{
				"type":    "object",
				"enabled": false,
			},
		},
		{
			output: p.text(&common.Field{Type: "text", Analyzer: "autocomplete"}),
			expected: common.MapStr{
				"type":     "text",
				"analyzer": "autocomplete",
				"norms":    false,
			},
		},
		{
			output: p.text(&common.Field{Type: "text", Analyzer: "autocomplete", Norms: true}),
			expected: common.MapStr{
				"type":     "text",
				"analyzer": "autocomplete",
			},
		},
		{
			output: p.text(&common.Field{Type: "text", SearchAnalyzer: "standard", Norms: true}),
			expected: common.MapStr{
				"type":            "text",
				"search_analyzer": "standard",
			},
		},
		{
			output: p.text(&common.Field{Type: "text", Analyzer: "autocomplete", SearchAnalyzer: "standard", Norms: true}),
			expected: common.MapStr{
				"type":            "text",
				"analyzer":        "autocomplete",
				"search_analyzer": "standard",
			},
		},
		{
			output: p.text(&common.Field{Type: "text", MultiFields: common.Fields{common.Field{Name: "raw", Type: "keyword"}}, Norms: true}),
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
			output: p.text(&common.Field{Type: "text", MultiFields: common.Fields{
				common.Field{Name: "raw", Type: "keyword"},
				common.Field{Name: "indexed", Type: "text"},
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
			output: p.object(&common.Field{Dynamic: common.DynamicType{Value: false}}),
			expected: common.MapStr{
				"dynamic": false, "type": "object",
			},
		},
		{
			output: p.object(&common.Field{Dynamic: common.DynamicType{Value: true}}),
			expected: common.MapStr{
				"dynamic": true, "type": "object",
			},
		},
		{
			output: p.object(&common.Field{Dynamic: common.DynamicType{Value: "strict"}}),
			expected: common.MapStr{
				"dynamic": "strict", "type": "object",
			},
		},
		{
			output: p.other(&common.Field{Type: "long", Index: &falseVar}),
			expected: common.MapStr{
				"type": "long", "index": false,
			},
		},
		{
			output: p.other(&common.Field{Type: "text", Index: &trueVar}),
			expected: common.MapStr{
				"type": "text", "index": true,
			},
		},
		{
			output: p.other(&common.Field{Type: "long", DocValues: &falseVar}),
			expected: common.MapStr{
				"type": "long", "doc_values": false,
			},
		},
		{
			output: p.other(&common.Field{Type: "text", DocValues: &trueVar}),
			expected: common.MapStr{
				"type": "text", "doc_values": true,
			},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.output)
	}
}

func TestDynamicTemplate(t *testing.T) {
	p := &Processor{}
	tests := []struct {
		field    common.Field
		expected common.MapStr
	}{
		{
			field: common.Field{
				Type: "object", ObjectType: "keyword",
				Name: "context",
			},
			expected: common.MapStr{
				"context": common.MapStr{
					"mapping":            common.MapStr{"type": "keyword"},
					"match_mapping_type": "string",
					"path_match":         "context.*",
				},
			},
		},
		{
			field: common.Field{
				Type: "object", ObjectType: "long", ObjectTypeMappingType: "futuretype",
				Path: "language", Name: "english",
			},
			expected: common.MapStr{
				"language.english": common.MapStr{
					"mapping":            common.MapStr{"type": "long"},
					"match_mapping_type": "futuretype",
					"path_match":         "language.english.*",
				},
			},
		},
		{
			field: common.Field{
				Type: "object", ObjectType: "long", ObjectTypeMappingType: "*",
				Path: "language", Name: "english",
			},
			expected: common.MapStr{
				"language.english": common.MapStr{
					"mapping":            common.MapStr{"type": "long"},
					"match_mapping_type": "*",
					"path_match":         "language.english.*",
				},
			},
		},
		{
			field: common.Field{
				Type: "object", ObjectType: "long",
				Path: "language", Name: "english",
			},
			expected: common.MapStr{
				"language.english": common.MapStr{
					"mapping":            common.MapStr{"type": "long"},
					"match_mapping_type": "long",
					"path_match":         "language.english.*",
				},
			},
		},
		{
			field: common.Field{
				Type: "object", ObjectType: "text",
				Path: "language", Name: "english",
			},
			expected: common.MapStr{
				"language.english": common.MapStr{
					"mapping":            common.MapStr{"type": "text"},
					"match_mapping_type": "string",
					"path_match":         "language.english.*",
				},
			},
		},
		{
			field: common.Field{
				Type: "object", ObjectType: "scaled_float",
				Name: "core.*.pct",
			},
			expected: common.MapStr{
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
		{
			field: common.Field{
				Type: "object", ObjectType: "scaled_float",
				Name: "core.*.pct", ScalingFactor: 100, ObjectTypeMappingType: "float",
			},
			expected: common.MapStr{
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
	}

	for _, test := range tests {
		dynamicTemplates = nil
		p.object(&test.field)
		assert.Equal(t, test.expected, dynamicTemplates[0])
	}
}

func TestPropertiesCombine(t *testing.T) {
	// Test common fields are combined even if they come from different objects
	fields := common.Fields{
		common.Field{
			Name: "test",
			Type: "group",
			Fields: common.Fields{
				common.Field{
					Name: "one",
					Type: "text",
				},
			},
		},
		common.Field{
			Name: "test",
			Type: "group",
			Fields: common.Fields{
				common.Field{
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
	err = p.process(fields, "", output)
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
	fields := common.Fields{
		common.Field{
			Fields: common.Fields{
				common.Field{
					Name: "one",
					Type: "text",
				},
			},
		},
		common.Field{
			Name: "test",
			Type: "group",
			Fields: common.Fields{
				common.Field{
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
	err = p.process(fields, "", output)
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
