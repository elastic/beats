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

	"github.com/elastic/beats/libbeat/common"
)

func TestReplaceStringInDashboard(t *testing.T) {
	tests := []struct {
		content  common.MapStr
		old      string
		new      string
		expected common.MapStr
	}{
		{
			content:  common.MapStr{"test": "CHANGEME"},
			old:      "CHANGEME",
			new:      "hostname",
			expected: common.MapStr{"test": "hostname"},
		},
		{
			content:  common.MapStr{"test": "hello"},
			old:      "CHANGEME",
			new:      "hostname",
			expected: common.MapStr{"test": "hello"},
		},
		{
			content:  common.MapStr{"test": map[string]interface{}{"key": "\"CHANGEME\""}},
			old:      "CHANGEME",
			new:      "hostname.local",
			expected: common.MapStr{"test": map[string]interface{}{"key": "\"hostname.local\""}},
		},
		{
			content: common.MapStr{
				"kibanaSavedObjectMeta": map[string]interface{}{
					"searchSourceJSON": "{\"filter\":[],\"highlightAll\":true,\"version\":true,\"query\":{\"query\":\"beat.name:\\\"CHANGEME_HOSTNAME\\\"\",\"language\":\"kuery\"}}"}},

			old: "CHANGEME_HOSTNAME",
			new: "hostname.local",
			expected: common.MapStr{
				"kibanaSavedObjectMeta": map[string]interface{}{
					"searchSourceJSON": "{\"filter\":[],\"highlightAll\":true,\"version\":true,\"query\":{\"query\":\"beat.name:\\\"hostname.local\\\"\",\"language\":\"kuery\"}}"}},
		},
	}

	for _, test := range tests {
		result, err := ReplaceStringInDashboard(test.old, test.new, test.content)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, result)
	}
}

func TestReplaceIndexInDashboardObject(t *testing.T) {
	tests := []struct {
		dashboard common.MapStr
		pattern   string
		expected  common.MapStr
	}{
		{
			common.MapStr{"objects": []interface{}{map[string]interface{}{
				"attributes": map[string]interface{}{
					"kibanaSavedObjectMeta": map[string]interface{}{
						"searchSourceJSON": "{\"index\":\"metricbeat-*\"}",
					},
				}}}},
			"otherindex-*",
			common.MapStr{"objects": []interface{}{map[string]interface{}{
				"attributes": map[string]interface{}{
					"kibanaSavedObjectMeta": map[string]interface{}{
						"searchSourceJSON": "{\"index\":\"otherindex-*\"}",
					},
				}}}},
		},
		{
			common.MapStr{"objects": []interface{}{map[string]interface{}{
				"attributes": map[string]interface{}{
					"kibanaSavedObjectMeta": map[string]interface{}{},
					"visState":              "{\"params\":{\"index_pattern\":\"metricbeat-*\"}}",
				}}}},
			"otherindex-*",
			common.MapStr{"objects": []interface{}{map[string]interface{}{
				"attributes": map[string]interface{}{
					"kibanaSavedObjectMeta": map[string]interface{}{},
					"visState":              "{\"params\":{\"index_pattern\":\"otherindex-*\"}}",
				}}}},
		},
	}

	for _, test := range tests {
		result := ReplaceIndexInDashboardObject(test.pattern, test.dashboard)
		assert.Equal(t, test.expected, result)
	}
}
