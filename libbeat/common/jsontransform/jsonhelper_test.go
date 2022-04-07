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

package jsontransform

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
)

func TestWriteJSONKeys(t *testing.T) {
	now := time.Now()
	now = now.Round(time.Second)

	eventTimestamp := time.Date(2020, 01, 01, 01, 01, 00, 0, time.UTC)
	eventMetadata := common.MapStr{
		"foo": "bar",
		"baz": common.MapStr{
			"qux": 17,
		},
	}
	eventFields := common.MapStr{
		"top_a": 23,
		"top_b": common.MapStr{
			"inner_c": "see",
			"inner_d": "dee",
		},
	}

	tests := map[string]struct {
		keys              map[string]interface{}
		expandKeys        bool
		overwriteKeys     bool
		expectedMetadata  common.MapStr
		expectedTimestamp time.Time
		expectedFields    common.MapStr
	}{
		"overwrite_true": {
			overwriteKeys: true,
			keys: map[string]interface{}{
				"@metadata": map[string]interface{}{
					"foo": "NEW_bar",
					"baz": map[string]interface{}{
						"qux":   "NEW_qux",
						"durrr": "COMPLETELY_NEW",
					},
				},
				"@timestamp": now.Format(time.RFC3339),
				"top_b": map[string]interface{}{
					"inner_d": "NEW_dee",
					"inner_e": "COMPLETELY_NEW_e",
				},
				"top_c": "COMPLETELY_NEW_c",
			},
			expectedMetadata: common.MapStr{
				"foo": "NEW_bar",
				"baz": common.MapStr{
					"qux":   "NEW_qux",
					"durrr": "COMPLETELY_NEW",
				},
			},
			expectedTimestamp: now,
			expectedFields: common.MapStr{
				"top_a": 23,
				"top_b": common.MapStr{
					"inner_c": "see",
					"inner_d": "NEW_dee",
					"inner_e": "COMPLETELY_NEW_e",
				},
				"top_c": "COMPLETELY_NEW_c",
			},
		},
		"overwrite_true_ISO8601": {
			overwriteKeys: true,
			keys: map[string]interface{}{
				"@metadata": map[string]interface{}{
					"foo": "NEW_bar",
					"baz": map[string]interface{}{
						"qux":   "NEW_qux",
						"durrr": "COMPLETELY_NEW",
					},
				},
				"@timestamp": now.Format(iso8601),
				"top_b": map[string]interface{}{
					"inner_d": "NEW_dee",
					"inner_e": "COMPLETELY_NEW_e",
				},
				"top_c": "COMPLETELY_NEW_c",
			},
			expectedMetadata: common.MapStr{
				"foo": "NEW_bar",
				"baz": common.MapStr{
					"qux":   "NEW_qux",
					"durrr": "COMPLETELY_NEW",
				},
			},
			expectedTimestamp: now,
			expectedFields: common.MapStr{
				"top_a": 23,
				"top_b": common.MapStr{
					"inner_c": "see",
					"inner_d": "NEW_dee",
					"inner_e": "COMPLETELY_NEW_e",
				},
				"top_c": "COMPLETELY_NEW_c",
			},
		},
		"overwrite_false": {
			overwriteKeys: false,
			keys: map[string]interface{}{
				"@metadata": map[string]interface{}{
					"foo": "NEW_bar",
					"baz": map[string]interface{}{
						"qux":   "NEW_qux",
						"durrr": "COMPLETELY_NEW",
					},
				},
				"@timestamp": now.Format(time.RFC3339),
				"top_b": map[string]interface{}{
					"inner_d": "NEW_dee",
					"inner_e": "COMPLETELY_NEW_e",
				},
				"top_c": "COMPLETELY_NEW_c",
			},
			expectedMetadata:  eventMetadata.Clone(),
			expectedTimestamp: eventTimestamp,
			expectedFields: common.MapStr{
				"top_a": 23,
				"top_b": common.MapStr{
					"inner_c": "see",
					"inner_d": "dee",
					"inner_e": "COMPLETELY_NEW_e",
				},
				"top_c": "COMPLETELY_NEW_c",
			},
		},
		"expand_true": {
			expandKeys:    true,
			overwriteKeys: true,
			keys: map[string]interface{}{
				"top_b": map[string]interface{}{
					"inner_d.inner_e": "COMPLETELY_NEW_e",
				},
			},
			expectedMetadata:  eventMetadata.Clone(),
			expectedTimestamp: eventTimestamp,
			expectedFields: common.MapStr{
				"top_a": 23,
				"top_b": common.MapStr{
					"inner_c": "see",
					"inner_d": common.MapStr{
						"inner_e": "COMPLETELY_NEW_e",
					},
				},
			},
		},
		"expand_false": {
			expandKeys:    false,
			overwriteKeys: true,
			keys: map[string]interface{}{
				"top_b": map[string]interface{}{
					"inner_d.inner_e": "COMPLETELY_NEW_e",
				},
			},
			expectedMetadata:  eventMetadata.Clone(),
			expectedTimestamp: eventTimestamp,
			expectedFields: common.MapStr{
				"top_a": 23,
				"top_b": common.MapStr{
					"inner_c":         "see",
					"inner_d":         "dee",
					"inner_d.inner_e": "COMPLETELY_NEW_e",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			event := &beat.Event{
				Timestamp: eventTimestamp,
				Meta:      eventMetadata.Clone(),
				Fields:    eventFields.Clone(),
			}

			WriteJSONKeys(event, test.keys, test.expandKeys, test.overwriteKeys, false)
			require.Equal(t, test.expectedMetadata, event.Meta)
			require.Equal(t, test.expectedTimestamp.UnixNano(), event.Timestamp.UnixNano())
			require.Equal(t, test.expectedFields, event.Fields)
		})
	}
}
