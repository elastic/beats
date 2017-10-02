package kibana

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

var (
	truthy     = true
	falsy      = false
	ctMetaData = 4
)

func TestEmpty(t *testing.T) {
	trans := NewTransformer("name", "title", common.Fields{})
	out := trans.TransformFields()
	expected := common.MapStr{
		"timeFieldName":  "name",
		"title":          "title",
		"fieldFormatMap": common.MapStr{},
		"fields": []common.MapStr{
			common.MapStr{
				"name":         "_id",
				"type":         "string",
				"scripted":     false,
				"aggregatable": false,
				"analyzed":     false,
				"count":        0,
				"indexed":      false,
				"doc_values":   false,
				"searchable":   false,
			},
			common.MapStr{
				"name":         "_type",
				"type":         "string",
				"scripted":     false,
				"count":        0,
				"aggregatable": true,
				"analyzed":     false,
				"indexed":      false,
				"doc_values":   false,
				"searchable":   true,
			},
			common.MapStr{
				"name":         "_index",
				"type":         "string",
				"scripted":     false,
				"count":        0,
				"aggregatable": false,
				"analyzed":     false,
				"indexed":      false,
				"doc_values":   false,
				"searchable":   false,
			},
			common.MapStr{
				"name":         "_score",
				"type":         "number",
				"scripted":     false,
				"count":        0,
				"aggregatable": false,
				"analyzed":     false,
				"indexed":      false,
				"doc_values":   false,
				"searchable":   false,
			},
		},
	}
	assert.Equal(t, expected, out)
}

func TestErrors(t *testing.T) {
	commonFields := common.Fields{
		common.Field{Name: "context", Path: "something"},
		common.Field{Name: "context", Path: "something", Type: "keyword"},
	}
	trans := NewTransformer("name", "title", commonFields)
	assert.Panics(t, func() { trans.TransformFields() })
}

func TestTransformTypes(t *testing.T) {
	tests := []struct {
		commonField common.Field
		expected    interface{}
	}{
		{commonField: common.Field{}, expected: "string"},
		{commonField: common.Field{Type: "half_float"}, expected: "number"},
		{commonField: common.Field{Type: "scaled_float"}, expected: "number"},
		{commonField: common.Field{Type: "float"}, expected: "number"},
		{commonField: common.Field{Type: "integer"}, expected: "number"},
		{commonField: common.Field{Type: "long"}, expected: "number"},
		{commonField: common.Field{Type: "short"}, expected: "number"},
		{commonField: common.Field{Type: "byte"}, expected: "number"},
		{commonField: common.Field{Type: "keyword"}, expected: "string"},
		{commonField: common.Field{Type: "text"}, expected: "string"},
		{commonField: common.Field{Type: "string"}, expected: nil},
		{commonField: common.Field{Type: "date"}, expected: "date"},
		{commonField: common.Field{Type: "geo_point"}, expected: "geo_point"},
		{commonField: common.Field{Type: "invalid"}, expected: nil},
	}
	for idx, test := range tests {
		trans := NewTransformer("name", "title", common.Fields{test.commonField})
		out := trans.TransformFields()["fields"].([]common.MapStr)[0]
		assert.Equal(t, test.expected, out["type"], fmt.Sprintf("Failed for idx %v", idx))
	}
}

func TestTransformGroup(t *testing.T) {
	tests := []struct {
		commonFields common.Fields
		expected     []string
	}{
		{
			commonFields: common.Fields{common.Field{Name: "context", Path: "something"}},
			expected:     []string{"context"},
		},
		{
			commonFields: common.Fields{
				common.Field{
					Name: "context",
					Type: "group",
					Fields: common.Fields{
						common.Field{Name: "another", Type: ""},
					},
				},
				common.Field{
					Name: "context",
					Type: "group",
					Fields: common.Fields{
						common.Field{Name: "type", Type: ""},
						common.Field{
							Name: "metric",
							Type: "group",
							Fields: common.Fields{
								common.Field{Name: "object"},
							},
						},
					},
				},
			},
			expected: []string{"context.another", "context.type", "context.metric.object"},
		},
	}
	for idx, test := range tests {
		trans := NewTransformer("name", "title", test.commonFields)
		out := trans.TransformFields()["fields"].([]common.MapStr)
		assert.Equal(t, len(test.expected)+ctMetaData, len(out))
		for i, e := range test.expected {
			assert.Equal(t, e, out[i]["name"], fmt.Sprintf("Failed for idx %v", idx))
		}
	}
}

func TestTransformMisc(t *testing.T) {
	tests := []struct {
		commonField common.Field
		expected    interface{}
		attr        string
	}{
		{commonField: common.Field{}, expected: 0, attr: "count"},

		// searchable always set to true except for meta fields
		{commonField: common.Field{}, expected: true, attr: "searchable"},
		{commonField: common.Field{Searchable: &truthy}, expected: true, attr: "searchable"},
		{commonField: common.Field{Searchable: &falsy}, expected: true, attr: "searchable"},

		// aggregatable always set to true except for meta fields or type text
		{commonField: common.Field{}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &falsy}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Type: "keyword"}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy, Type: "text"}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Type: "text"}, expected: false, attr: "aggregatable"},

		// analyzed always set to false except for meta fields
		{commonField: common.Field{}, expected: false, attr: "analyzed"},
		{commonField: common.Field{Analyzed: &truthy}, expected: false, attr: "analyzed"},
		{commonField: common.Field{Analyzed: &falsy}, expected: false, attr: "analyzed"},

		// indexed always set to true except for meta fields
		{commonField: common.Field{}, expected: true, attr: "indexed"},
		{commonField: common.Field{Index: &truthy}, expected: true, attr: "indexed"},
		{commonField: common.Field{Index: &falsy}, expected: true, attr: "indexed"},

		// doc_values always set to true except for meta fields
		{commonField: common.Field{}, expected: true, attr: "doc_values"},
		{commonField: common.Field{DocValues: &truthy}, expected: true, attr: "doc_values"},
		{commonField: common.Field{DocValues: &falsy}, expected: true, attr: "doc_values"},

		// scripted always set to false
		{commonField: common.Field{}, expected: false, attr: "scripted"},
	}
	for idx, test := range tests {
		trans := NewTransformer("", "", common.Fields{test.commonField})
		out := trans.TransformFields()["fields"].([]common.MapStr)[0]
		msg := fmt.Sprintf("(%v): expected '%s' to be <%v> but was <%v>", idx, test.attr, test.expected, out[test.attr])
		assert.Equal(t, test.expected, out[test.attr], msg)
	}
}

func TestTransformFieldFormatMap(t *testing.T) {
	tests := []struct {
		commonField common.Field
		expected    common.MapStr
	}{
		{
			commonField: common.Field{Name: "c"},
			expected:    common.MapStr{},
		},
		{
			commonField: common.Field{Name: "c", Format: "url"},
			expected:    common.MapStr{"c": common.MapStr{"id": "url"}},
		},
		{
			commonField: common.Field{
				Name:    "c",
				Pattern: "p",
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"params": common.MapStr{"pattern": "p"},
				},
			},
		},
		{
			commonField: common.Field{
				Name:    "c",
				Format:  "url",
				Pattern: "p",
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"id":     "url",
					"params": common.MapStr{"pattern": "p"},
				},
			},
		},
		{
			commonField: common.Field{
				Name:        "c",
				Format:      "url",
				InputFormat: "string",
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"id": "url",
					"params": common.MapStr{
						"inputFormat": "string",
					},
				},
			},
		},
		{
			commonField: common.Field{
				Name:        "c",
				Format:      "url",
				Pattern:     "[^-]",
				InputFormat: "string",
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"id": "url",
					"params": common.MapStr{
						"pattern": "[^-]",
					},
				},
			},
		},
		{
			commonField: common.Field{
				Name:        "c",
				InputFormat: "string",
			},
			expected: common.MapStr{},
		},
	}
	for idx, test := range tests {
		trans := NewTransformer("", "", common.Fields{test.commonField})
		out := trans.TransformFields()["fieldFormatMap"]
		assert.Equal(t, test.expected, out, fmt.Sprintf("Failed for idx %v", idx))
	}
}
