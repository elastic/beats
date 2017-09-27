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

func TestErrors(t *testing.T) {
	commonFields := common.Fields{
		common.Field{Name: "context", Path: "something"},
		common.Field{Name: "context", Path: "something", Type: "keyword"},
	}
	assert.Panics(t, func() { TransformFields("", "", commonFields) })
}

func TestEmpty(t *testing.T) {
	out := TransformFields("name", "title", common.Fields{})
	expected := common.MapStr{
		"timeFieldName":  "name",
		"title":          "title",
		"fieldFormatMap": common.MapStr{},
		"fields": []common.MapStr{
			common.MapStr{
				"name":              "_id",
				"type":              "string",
				"aggregatable":      true,
				"analyzed":          false,
				"count":             0,
				"index":             true,
				"readFromDocValues": true,
				"searchable":        true,
			},
			common.MapStr{
				"name":              "_type",
				"type":              "string",
				"aggregatable":      true,
				"analyzed":          false,
				"count":             0,
				"index":             true,
				"readFromDocValues": true,
				"searchable":        true,
			},
			common.MapStr{
				"name":              "_index",
				"type":              "string",
				"aggregatable":      true,
				"analyzed":          false,
				"count":             0,
				"index":             true,
				"readFromDocValues": true,
				"searchable":        true,
			},
			common.MapStr{
				"name":              "_score",
				"type":              "number",
				"aggregatable":      true,
				"analyzed":          false,
				"count":             0,
				"index":             true,
				"readFromDocValues": true,
				"searchable":        true,
			},
		},
	}
	assert.Equal(t, expected, out)
}

func TestTransformTypes(t *testing.T) {
	tests := []struct {
		commonField common.Field
		expected    string
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
		{commonField: common.Field{Type: "string"}, expected: "string"},
		{commonField: common.Field{Type: "date"}, expected: "date"},
		{commonField: common.Field{Type: "geo_point"}, expected: "geo_point"},
		{commonField: common.Field{Type: "invalid"}, expected: ""},
	}
	for idx, test := range tests {
		out := TransformFields("", "", common.Fields{test.commonField})["fields"].([]common.MapStr)[0]
		assert.Equal(t, test.expected, out["type"], fmt.Sprintf("Failed for idx %v", idx))
	}
}

func TestTransformGroupAndEnabled(t *testing.T) {
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
			expected: []string{"context.type", "context.metric.object"},
		},
		{
			commonFields: common.Fields{
				common.Field{Name: "enabledField"},
				common.Field{Name: "disabledField", Enabled: &falsy}, //enabled is ignored for Type!=group
				common.Field{
					Name:    "enabledGroup",
					Type:    "group",
					Enabled: &truthy,
					Fields: common.Fields{
						common.Field{Name: "type", Type: ""},
					},
				},
				common.Field{
					Name:    "context",
					Type:    "group",
					Enabled: &falsy,
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
			expected: []string{"enabledField", "disabledField", "enabledGroup.type"},
		},
	}
	for idx, test := range tests {
		out := TransformFields("", "", test.commonFields)["fields"].([]common.MapStr)
		assert.Equal(t, len(test.expected)+ctMetaData, len(out))
		for i, e := range test.expected {
			assert.Equal(t, e, out[i]["name"], fmt.Sprintf("Failed for idx %v", idx))
		}
	}
}

func TestTransformMultiField(t *testing.T) {
	f := common.Field{
		Name: "context",
		Type: "",
		MultiFields: common.Fields{
			common.Field{Name: "keyword", Type: "keyword"},
			common.Field{Name: "text", Type: "text"},
		},
	}
	out := TransformFields("", "", common.Fields{f})["fields"].([]common.MapStr)
	assert.Equal(t, "context", out[0]["name"])
	assert.Equal(t, "context.keyword", out[1]["name"])
	assert.Equal(t, "context.text", out[2]["name"])
	assert.Equal(t, "string", out[0]["type"])
	assert.Equal(t, "string", out[1]["type"])
	assert.Equal(t, "string", out[2]["type"])
}

func TestTransformMisc(t *testing.T) {
	tests := []struct {
		commonField common.Field
		expected    interface{}
		attr        string
	}{
		{commonField: common.Field{}, expected: 0, attr: "count"},
		{commonField: common.Field{Count: 3}, expected: 3, attr: "count"},
		{commonField: common.Field{}, expected: true, attr: "searchable"},

		{commonField: common.Field{Searchable: &truthy}, expected: true, attr: "searchable"},
		{commonField: common.Field{Searchable: &falsy}, expected: false, attr: "searchable"},
		{commonField: common.Field{Index: &falsy}, expected: false, attr: "searchable"},
		{commonField: common.Field{Searchable: &truthy, Index: &falsy}, expected: true, attr: "searchable"},

		{commonField: common.Field{}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &falsy}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy, Type: "text"}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Type: "text"}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Index: &falsy}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy, Index: &falsy}, expected: true, attr: "aggregatable"},

		{commonField: common.Field{}, expected: false, attr: "analyzed"},
		{commonField: common.Field{Analyzer: "", SearchAnalyzer: ""}, expected: false, attr: "analyzed"},
		{commonField: common.Field{Analyzer: "text"}, expected: true, attr: "analyzed"},
		{commonField: common.Field{SearchAnalyzer: "text"}, expected: true, attr: "analyzed"},
		{commonField: common.Field{Analyzer: "text", SearchAnalyzer: "text"}, expected: true, attr: "analyzed"},
		{commonField: common.Field{Index: &falsy}, expected: false, attr: "analyzed"},
		{commonField: common.Field{Analyzer: "text", Index: &falsy}, expected: false, attr: "analyzed"},

		{commonField: common.Field{Scripted: &truthy}, expected: true, attr: "scripted"},
		{commonField: common.Field{Scripted: &falsy, Script: "doc[]"}, expected: false, attr: "scripted"},
		{commonField: common.Field{Script: "doc[]"}, expected: true, attr: "scripted"},

		{commonField: common.Field{}, expected: nil, attr: "lang"},
		{commonField: common.Field{Lang: "lucene"}, expected: "lucene", attr: "lang"},
		{commonField: common.Field{Lang: "lucene", Script: "doc[]"}, expected: "lucene", attr: "lang"},
		{commonField: common.Field{Script: "doc[]"}, expected: "painless", attr: "lang"},

		{commonField: common.Field{}, expected: true, attr: "readFromDocValues"},
		{commonField: common.Field{DocValues: &falsy}, expected: false, attr: "readFromDocValues"},
		{commonField: common.Field{DocValues: &truthy, Script: "doc[]"}, expected: true, attr: "readFromDocValues"},
		{commonField: common.Field{Script: "doc[]"}, expected: false, attr: "readFromDocValues"},
		{commonField: common.Field{Index: &falsy}, expected: false, attr: "readFromDocValues"},
		{commonField: common.Field{DocValues: &truthy, Index: &falsy}, expected: true, attr: "readFromDocValues"},
		{commonField: common.Field{DocValues: &truthy, Analyzer: "text", Type: "text"}, expected: true, attr: "readFromDocValues"},
		{commonField: common.Field{Analyzer: "text", Type: "text"}, expected: false, attr: "readFromDocValues"},
		{commonField: common.Field{Analyzer: "text", Type: "keyword"}, expected: false, attr: "readFromDocValues"},
		{commonField: common.Field{Analyzer: "text", Type: "string"}, expected: false, attr: "readFromDocValues"},
		{commonField: common.Field{Analyzer: "text", Type: ""}, expected: false, attr: "readFromDocValues"},

		{commonField: common.Field{}, expected: nil, attr: "script"},
		{commonField: common.Field{Script: "doc[]"}, expected: "doc[]", attr: "script"},

		{commonField: common.Field{}, expected: true, attr: "index"},
		{commonField: common.Field{Index: &truthy}, expected: true, attr: "index"},
		{commonField: common.Field{Index: &falsy}, expected: false, attr: "index"},
	}
	for idx, test := range tests {
		out := TransformFields("", "", common.Fields{test.commonField})["fields"].([]common.MapStr)[0]
		assert.Equal(t, test.expected, out[test.attr], fmt.Sprintf("Failed for idx %v", idx))
	}
}

func TestTransformFielFormatMap(t *testing.T) {
	tests := []struct {
		commonField common.Field
		expected    common.MapStr
	}{
		{commonField: common.Field{Name: "c"}, expected: common.MapStr{}},
		{
			commonField: common.Field{Name: "c", Format: "url"},
			expected:    common.MapStr{"c": common.MapStr{"id": "url"}},
		},
		{
			commonField: common.Field{Name: "c", Format: "url", Pattern: "p"},
			expected: common.MapStr{
				"c": common.MapStr{
					"id":     "url",
					"params": common.MapStr{"pattern": "p"},
				},
			},
		},
		{
			commonField: common.Field{
				Name:            "c",
				Format:          "url",
				Pattern:         "[^-]",
				InputFormat:     "string",
				OutputFormat:    "float",
				OutputPrecision: "3",
				LabelTemplate:   "lblT",
				UrlTemplate:     "urlT",
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"id": "url",
					"params": common.MapStr{
						"pattern":          "[^-]",
						"input_format":     "string",
						"output_format":    "float",
						"output_precision": "3",
						"label_template":   "lblT",
						"url_template":     "urlT",
					},
				},
			},
		},
		{
			commonField: common.Field{Name: "c", Format: "url"},
			expected:    common.MapStr{"c": common.MapStr{"id": "url"}},
		},
		{
			commonField: common.Field{Name: "c", Format: "url"},
			expected:    common.MapStr{"c": common.MapStr{"id": "url"}},
		},
		{
			commonField: common.Field{Name: "c", Format: "url"},
			expected:    common.MapStr{"c": common.MapStr{"id": "url"}},
		},
	}
	for idx, test := range tests {
		out := TransformFields("", "", common.Fields{test.commonField})["fieldFormatMap"]
		assert.Equal(t, test.expected, out, fmt.Sprintf("Failed for idx %v", idx))
	}
}
