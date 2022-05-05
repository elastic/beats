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

package dashboards

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestReplaceStringInDashboard(t *testing.T) {
	tests := []struct {
		content  []byte
		old      string
		new      string
		expected []byte
	}{
		{
			content:  []byte(`{"test": "CHANGEME"}`),
			old:      "CHANGEME",
			new:      "hostname",
			expected: []byte(`{"test": "hostname"}`),
		},
		{
			content:  []byte(`{"test": "hello"}`),
			old:      "CHANGEME",
			new:      "hostname",
			expected: []byte(`{"test": "hello"}`),
		},
		{
			content:  []byte(`{"test": {"key": "\"CHANGEME\""}}`),
			old:      "CHANGEME",
			new:      "hostname.local",
			expected: []byte(`{"test": {"key": "\"hostname.local\""}}`),
		},
		{
			content: []byte(`{"kibanaSavedObjectMeta": {"searchSourceJSON": "{\"filter\":[],\"highlightAll\":true,\"version\":true,\"query\":{\"query\":\"beat.name:\\\"CHANGEME_HOSTNAME\\\"\",\"language\":\"kuery\"}}}}`),

			old:      "CHANGEME_HOSTNAME",
			new:      "hostname.local",
			expected: []byte(`{"kibanaSavedObjectMeta": {"searchSourceJSON": "{\"filter\":[],\"highlightAll\":true,\"version\":true,\"query\":{\"query\":\"beat.name:\\\"hostname.local\\\"\",\"language\":\"kuery\"}}}}`),
		},
	}

	for _, test := range tests {
		result := ReplaceStringInDashboard(test.old, test.new, test.content)
		assert.Equal(t, string(test.expected), string(result))
	}
}

func TestReplaceIndexInDashboardObject(t *testing.T) {
	tests := []struct {
		dashboard []byte
		pattern   string
		expected  []byte
	}{
		{
			[]byte(`{"attributes":{"kibanaSavedObjectMeta":{"searchSourceJSON":"{\"index\":\"metricbeat-*\"}"}}}`),
			"otherindex-*",
			[]byte(`{"attributes":{"kibanaSavedObjectMeta":{"searchSourceJSON":"{\"index\":\"otherindex-*\"}"}}}`),
		},
		{
			[]byte(`{"attributes":{"layerListJSON":[{"joins":[{"leftField":"iso2","right":{"indexPatternTitle":"filebeat-*"}}]}]}}`),
			"otherindex-*",
			[]byte(`{"attributes":{"layerListJSON":[{"joins":[{"leftField":"iso2","right":{"indexPatternTitle":"otherindex-*"}}]}]}}`),
		},
		{
			[]byte(`{"attributes":{"panelsJSON":[{"embeddableConfig":{"attributes":{"references":[{"id":"filebeat-*","type":"index-pattern"}]}}}]}}`),
			"otherindex-*",
			[]byte(`{"attributes":{"panelsJSON":[{"embeddableConfig":{"attributes":{"references":[{"id":"otherindex-*","type":"index-pattern"}]}}}]}}`),
		},
		{
			[]byte(`{"attributes":{},"references":[{"id":"auditbeat-*","name":"kibanaSavedObjectMeta.searchSourceJSON.index","type":"index-pattern"}]}`),
			"otherindex-*",
			[]byte(`{"attributes":{},"references":[{"id":"otherindex-*","name":"kibanaSavedObjectMeta.searchSourceJSON.index","type":"index-pattern"}]}`),
		},
		{
			[]byte(`{"attributes":{"visState":{"params":{"index_pattern":"winlogbeat-*"}}}}`),
			"otherindex-*",
			[]byte(`{"attributes":{"visState":{"params":{"index_pattern":"otherindex-*"}}}}`),
		},
		{
			[]byte(`{"attributes":{"visState":{"params":{"series":[{"series_index_pattern":"filebeat-*"}]}}}}`),
			"otherindex-*",
			[]byte(`{"attributes":{"visState":{"params":{"series":[{"series_index_pattern":"otherindex-*"}]}}}}`),
		},
		{
			[]byte(`{"attributes":{"mapStateJSON":{"filters":[{"meta":{"index":"filebeat-*"}}]}}}`),
			"otherindex-*",
			[]byte(`{"attributes":{"mapStateJSON":{"filters":[{"meta":{"index":"otherindex-*"}}]}}}`),
		},
	}

	for _, test := range tests {
		result := ReplaceIndexInDashboardObject(test.pattern, test.dashboard)
		assert.Equal(t, string(test.expected), string(result))
	}
}

func TestReplaceIndexInIndexPattern(t *testing.T) {
	// Test that replacing of index name in index pattern works no matter
	// what the inner types are (MapStr, map[string]interface{} or interface{}).
	// Also ensures that the inner types are not modified after replacement.
	tests := []struct {
		title    string
		input    mapstr.M
		index    string
		expected mapstr.M
	}{
		{
			title: "Replace in []mapstr.mapstr",
			input: mapstr.M{
				"id":   "phonybeat-*",
				"type": "index-pattern",
				"attributes": mapstr.M{
					"title":         "phonybeat-*",
					"timeFieldName": "@timestamp",
				}},
			index: "otherindex-*",
			expected: mapstr.M{
				"id":   "otherindex-*",
				"type": "index-pattern",
				"attributes": mapstr.M{
					"title":         "otherindex-*",
					"timeFieldName": "@timestamp",
				}},
		},
		{
			title: "Replace in []mapstr.interface(mapstr)",
			input: mapstr.M{
				"id":   "phonybeat-*",
				"type": "index-pattern",
				"attributes": interface{}(mapstr.M{
					"title":         "phonybeat-*",
					"timeFieldName": "@timestamp",
				})},
			index: "otherindex-*",
			expected: mapstr.M{
				"id":   "otherindex-*",
				"type": "index-pattern",
				"attributes": interface{}(mapstr.M{
					"title":         "otherindex-*",
					"timeFieldName": "@timestamp",
				})},
		},
		{
			title: "Do not create missing attributes",
			input: mapstr.M{
				"attributes": mapstr.M{},
				"id":         "phonybeat-*",
				"type":       "index-pattern",
			},
			index: "otherindex-*",
			expected: mapstr.M{
				"attributes": mapstr.M{},
				"id":         "otherindex-*",
				"type":       "index-pattern",
			},
		},
		{
			title: "Create missing id",
			input: mapstr.M{
				"attributes": mapstr.M{},
				"type":       "index-pattern",
			},
			index: "otherindex-*",
			expected: mapstr.M{
				"attributes": mapstr.M{},
				"id":         "otherindex-*",
				"type":       "index-pattern",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			err := ReplaceIndexInIndexPattern(test.index, test.input)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, test.input)
		})
	}
}
