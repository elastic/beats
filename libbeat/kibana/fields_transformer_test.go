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
	version, _ = common.NewVersion("6.0.0")
)

func TestEmpty(t *testing.T) {
	trans, err := newFieldsTransformer(version, common.Fields{})
	assert.NoError(t, err)
	out, err := trans.transform()
	assert.NoError(t, err)
	expected := common.MapStr{
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

func TestMissingVersion(t *testing.T) {
	var c *common.Version
	_, err := newFieldsTransformer(c, common.Fields{})
	assert.Error(t, err)
}

func TestDuplicateField(t *testing.T) {
	commonFields := common.Fields{
		common.Field{Name: "context", Path: "something"},
		common.Field{Name: "context", Path: "something", Type: "keyword"},
	}
	trans, err := newFieldsTransformer(version, commonFields)
	assert.NoError(t, err)
	_, err = trans.transform()
	assert.Error(t, err)
}

func TestInvalidVersion(t *testing.T) {
	commonFields := common.Fields{
		common.Field{
			Name:   "versionTest",
			Format: "url",
			UrlTemplate: []common.VersionizedString{
				{MinVersion: "3", Value: ""},
			},
		},
	}
	trans, err := newFieldsTransformer(version, commonFields)
	assert.NoError(t, err)
	_, err = trans.transform()
	assert.Error(t, err)
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
		trans, _ := newFieldsTransformer(version, common.Fields{test.commonField})
		transformed, err := trans.transform()
		assert.NoError(t, err)
		out := transformed["fields"].([]common.MapStr)[0]
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
		trans, _ := newFieldsTransformer(version, test.commonFields)
		transformed, err := trans.transform()
		assert.NoError(t, err)
		out := transformed["fields"].([]common.MapStr)
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
		{commonField: common.Field{Count: 4}, expected: 4, attr: "count"},

		// searchable
		{commonField: common.Field{}, expected: true, attr: "searchable"},
		{commonField: common.Field{Searchable: &truthy}, expected: true, attr: "searchable"},
		{commonField: common.Field{Searchable: &falsy}, expected: false, attr: "searchable"},
		{commonField: common.Field{Type: "binary"}, expected: false, attr: "searchable"},
		{commonField: common.Field{Searchable: &truthy, Type: "binary"}, expected: false, attr: "searchable"},

		// aggregatable
		{commonField: common.Field{}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &falsy}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Type: "binary"}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy, Type: "binary"}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Type: "keyword"}, expected: true, attr: "aggregatable"},
		{commonField: common.Field{Aggregatable: &truthy, Type: "text"}, expected: false, attr: "aggregatable"},
		{commonField: common.Field{Type: "text"}, expected: false, attr: "aggregatable"},

		// analyzed
		{commonField: common.Field{}, expected: false, attr: "analyzed"},
		{commonField: common.Field{Analyzed: &truthy}, expected: true, attr: "analyzed"},
		{commonField: common.Field{Analyzed: &falsy}, expected: false, attr: "analyzed"},
		{commonField: common.Field{Type: "binary"}, expected: false, attr: "analyzed"},
		{commonField: common.Field{Analyzed: &truthy, Type: "binary"}, expected: false, attr: "analyzed"},

		// doc_values always set to true except for meta fields
		{commonField: common.Field{}, expected: true, attr: "doc_values"},
		{commonField: common.Field{DocValues: &truthy}, expected: true, attr: "doc_values"},
		{commonField: common.Field{DocValues: &falsy}, expected: false, attr: "doc_values"},
		{commonField: common.Field{Script: "doc[]"}, expected: false, attr: "doc_values"},
		{commonField: common.Field{DocValues: &truthy, Script: "doc[]"}, expected: false, attr: "doc_values"},
		{commonField: common.Field{Type: "binary"}, expected: false, attr: "doc_values"},
		{commonField: common.Field{DocValues: &truthy, Type: "binary"}, expected: true, attr: "doc_values"},

		// indexed
		{commonField: common.Field{Type: "binary"}, expected: false, attr: "indexed"},
		{commonField: common.Field{Index: &truthy, Type: "binary"}, expected: false, attr: "indexed"},

		// script, scripted
		{commonField: common.Field{}, expected: false, attr: "scripted"},
		{commonField: common.Field{}, expected: nil, attr: "script"},
		{commonField: common.Field{Script: "doc[]"}, expected: true, attr: "scripted"},
		{commonField: common.Field{Script: "doc[]"}, expected: "doc[]", attr: "script"},
		{commonField: common.Field{Type: "binary"}, expected: false, attr: "scripted"},

		// language
		{commonField: common.Field{}, expected: nil, attr: "lang"},
		{commonField: common.Field{Script: "doc[]"}, expected: "painless", attr: "lang"},
	}
	for idx, test := range tests {
		trans, _ := newFieldsTransformer(version, common.Fields{test.commonField})
		transformed, err := trans.transform()
		assert.NoError(t, err)
		out := transformed["fields"].([]common.MapStr)[0]
		msg := fmt.Sprintf("(%v): expected '%s' to be <%v> but was <%v>", idx, test.attr, test.expected, out[test.attr])
		assert.Equal(t, test.expected, out[test.attr], msg)
	}
}

func TestTransformFieldFormatMap(t *testing.T) {
	precision := 3
	version620, _ := common.NewVersion("6.2.0")
	truthy := true
	falsy := false

	tests := []struct {
		commonField common.Field
		version     *common.Version
		expected    common.MapStr
	}{
		{
			commonField: common.Field{Name: "c"},
			expected:    common.MapStr{},
			version:     version,
		},
		{
			commonField: common.Field{Name: "c", Format: "url"},
			expected:    common.MapStr{"c": common.MapStr{"id": "url"}},
			version:     version,
		},
		{
			commonField: common.Field{Name: "c", Pattern: "p"},
			expected:    common.MapStr{"c": common.MapStr{"params": common.MapStr{"pattern": "p"}}},
			version:     version,
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
			version: version,
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
			version: version,
		},
		{
			commonField: common.Field{
				Name:                 "c",
				Format:               "url",
				Pattern:              "[^-]",
				InputFormat:          "string",
				OpenLinkInCurrentTab: &falsy,
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"id": "url",
					"params": common.MapStr{
						"pattern":              "[^-]",
						"inputFormat":          "string",
						"openLinkInCurrentTab": false,
					},
				},
			},
			version: version,
		},
		{
			commonField: common.Field{
				Name:        "c",
				InputFormat: "string",
			},
			expected: common.MapStr{},
			version:  version,
		},
		{
			version: version620,
			commonField: common.Field{
				Name:                 "c",
				Format:               "url",
				Pattern:              "[^-]",
				OpenLinkInCurrentTab: &truthy,
				InputFormat:          "string",
				OutputFormat:         "float",
				OutputPrecision:      &precision,
				LabelTemplate:        "lblT",
				UrlTemplate: []common.VersionizedString{
					{MinVersion: "5.0.0", Value: "5x.urlTemplate"},
					{MinVersion: "6.0.0", Value: "6x.urlTemplate"},
				},
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"id": "url",
					"params": common.MapStr{
						"pattern":              "[^-]",
						"inputFormat":          "string",
						"outputFormat":         "float",
						"outputPrecision":      3,
						"labelTemplate":        "lblT",
						"urlTemplate":          "6x.urlTemplate",
						"openLinkInCurrentTab": true,
					},
				},
			},
		},
		{
			version: version620,
			commonField: common.Field{
				Name:   "c",
				Format: "url",
				UrlTemplate: []common.VersionizedString{
					{MinVersion: "6.4.0", Value: "6x.urlTemplate"},
				},
			},
			expected: common.MapStr{
				"c": common.MapStr{"id": "url"},
			},
		},
		{
			version: version620,
			commonField: common.Field{
				Name:   "c",
				Format: "url",
				UrlTemplate: []common.VersionizedString{
					{MinVersion: "4.7.2", Value: "4x.urlTemplate"},
					{MinVersion: "6.5.1", Value: "6x.urlTemplate"},
				},
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"id": "url",
					"params": common.MapStr{
						"urlTemplate": "4x.urlTemplate",
					},
				},
			},
		},
		{
			version: version620,
			commonField: common.Field{
				Name:   "c",
				Format: "url",
				UrlTemplate: []common.VersionizedString{
					{MinVersion: "6.2.0", Value: "6.2.0.urlTemplate"},
					{MinVersion: "6.2.0-alpha", Value: "6.2.0-alpha.urlTemplate"},
					{MinVersion: "6.2.7", Value: "6.2.7.urlTemplate"},
				},
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"id": "url",
					"params": common.MapStr{
						"urlTemplate": "6.2.0.urlTemplate",
					},
				},
			},
		},
		{
			version: version620,
			commonField: common.Field{
				Name:   "c",
				Format: "url",
				UrlTemplate: []common.VersionizedString{
					{MinVersion: "4.1.0", Value: "4x.urlTemplate"},
					{MinVersion: "5.2.0-rc2", Value: "5.2.0-rc2.urlTemplate"},
					{MinVersion: "5.2.0-rc3", Value: "5.2.0-rc3.urlTemplate"},
					{MinVersion: "5.2.0-rc1", Value: "5.2.0-rc1.urlTemplate"},
				},
			},
			expected: common.MapStr{
				"c": common.MapStr{
					"id": "url",
					"params": common.MapStr{
						"urlTemplate": "5.2.0-rc3.urlTemplate",
					},
				},
			},
		},
	}
	for idx, test := range tests {
		trans, _ := newFieldsTransformer(test.version, common.Fields{test.commonField})
		transformed, err := trans.transform()
		assert.NoError(t, err)
		out := transformed["fieldFormatMap"]
		assert.Equal(t, test.expected, out, fmt.Sprintf("Failed for idx %v", idx))
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
		trans, _ := newFieldsTransformer(version, test.commonFields)
		transformed, err := trans.transform()
		assert.NoError(t, err)
		out := transformed["fields"].([]common.MapStr)
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
	trans, _ := newFieldsTransformer(version, common.Fields{f})
	transformed, err := trans.transform()
	assert.NoError(t, err)
	out := transformed["fields"].([]common.MapStr)
	assert.Equal(t, "context", out[0]["name"])
	assert.Equal(t, "context.keyword", out[1]["name"])
	assert.Equal(t, "context.text", out[2]["name"])
	assert.Equal(t, "string", out[0]["type"])
	assert.Equal(t, "string", out[1]["type"])
	assert.Equal(t, "string", out[2]["type"])
}
