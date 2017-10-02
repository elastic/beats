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
				"name":              "_id",
				"type":              "string",
				"scripted":          false,
				"aggregatable":      false,
				"count":             0,
				"indexed":           false,
				"readFromDocValues": false,
				"searchable":        false,
			},
			common.MapStr{
				"name":              "_type",
				"type":              "string",
				"scripted":          false,
				"count":             0,
				"aggregatable":      false,
				"indexed":           false,
				"readFromDocValues": false,
				"searchable":        false,
			},
			common.MapStr{
				"name":              "_index",
				"type":              "string",
				"scripted":          false,
				"count":             0,
				"aggregatable":      false,
				"indexed":           false,
				"readFromDocValues": false,
				"searchable":        false,
			},
			common.MapStr{
				"name":              "_score",
				"type":              "number",
				"scripted":          false,
				"count":             0,
				"aggregatable":      false,
				"indexed":           false,
				"readFromDocValues": false,
				"searchable":        false,
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
		{commonField: common.Field{Type: "string"}, expected: ""},
		{commonField: common.Field{Type: "date"}, expected: "date"},
		{commonField: common.Field{Type: "geo_point"}, expected: "geo_point"},
		{commonField: common.Field{Type: "invalid"}, expected: ""},
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
		// count
		{commonField: common.Field{}, expected: 0, attr: "count"},
		{commonField: common.Field{Count: 4}, expected: 4, attr: "count"},

		// searchable
		{commonField: common.Field{}, expected: true, attr: "searchable"},
		{commonField: common.Field{Searchable: &truthy}, expected: true, attr: "searchable"},
		{commonField: common.Field{Searchable: &falsy}, expected: false, attr: "searchable"},
		{commonField: common.Field{Index: &falsy}, expected: false, attr: "searchable"},
		{commonField: common.Field{Searchable: &truthy, Index: &falsy}, expected: false, attr: "searchable"},

		// aggregatable
		{commonField: common.Field{}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &falsy}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Type: "keyword"}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Type: "string"}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy, Type: "text"}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Type: "text"}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Index: &falsy}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy, Index: &falsy}, expected: false, attr: "aggregatable"},

		// indexed
		{commonField: common.Field{}, expected: true, attr: "indexed"},
		{commonField: common.Field{Index: &truthy}, expected: true, attr: "indexed"},
		{commonField: common.Field{Index: &falsy}, expected: false, attr: "indexed"},

		// readFromDocValues
		{commonField: common.Field{}, expected: true, attr: "readFromDocValues"},
		{commonField: common.Field{DocValues: &truthy}, expected: true, attr: "readFromDocValues"},
		{commonField: common.Field{DocValues: &falsy}, expected: false, attr: "readFromDocValues"},
		{commonField: common.Field{Index: &falsy}, expected: false, attr: "readFromDocValues"},
		{commonField: common.Field{DocValues: &truthy, Index: &falsy}, expected: false, attr: "readFromDocValues"},

		// scripted
		{commonField: common.Field{}, expected: false, attr: "scripted"},
		{commonField: common.Field{Script: "doc[]"}, expected: true, attr: "scripted"},

		// language
		{commonField: common.Field{}, expected: nil, attr: "lang"},
		{commonField: common.Field{Lang: "lucene"}, expected: nil, attr: "lang"},
		{commonField: common.Field{Lang: "lucene", Script: "doc[]"}, expected: "lucene", attr: "lang"},
		{commonField: common.Field{Script: "doc[]"}, expected: "painless", attr: "lang"},
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
			expected: common.MapStr{},
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
						"pattern":     "[^-]",
						"inputFormat": "string",
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
						"pattern":         "[^-]",
						"inputFormat":     "string",
						"outputFormat":    "float",
						"outputPrecision": "3",
						"labelTemplate":   "lblT",
						"urlTemplate":     "urlT",
					},
				},
			},
		},
	}
	for idx, test := range tests {
		trans := NewTransformer("", "", common.Fields{test.commonField})
		out := trans.TransformFields()["fieldFormatMap"]
		assert.Equal(t, test.expected, out, fmt.Sprintf("Failed for idx %v", idx))
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
	trans := NewTransformer("", "", common.Fields{f})
	out := trans.TransformFields()["fields"].([]common.MapStr)
	assert.Equal(t, "context", out[0]["name"])
	assert.Equal(t, "context.keyword", out[1]["name"])
	assert.Equal(t, "context.text", out[2]["name"])
	assert.Equal(t, "string", out[0]["type"])
	assert.Equal(t, "string", out[1]["type"])
	assert.Equal(t, "string", out[2]["type"])
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
		trans := NewTransformer("", "", test.commonFields)
		out := trans.TransformFields()["fields"].([]common.MapStr)
		assert.Equal(t, len(test.expected)+ctMetaData, len(out))
		for i, e := range test.expected {
			assert.Equal(t, e, out[i]["name"], fmt.Sprintf("Failed for idx %v", idx))
		}
	}
}
