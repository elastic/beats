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

package urldecode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestURLDecodeFailOnErrorSafety verifies that when FailOnError=true and
// decoding fails, the event fields are unchanged (proving clone skip is safe).
func TestURLDecodeFailOnErrorSafety(t *testing.T) {
	tests := []struct {
		name  string
		input mapstr.M
	}{
		{
			name:  "invalid percent encoding",
			input: mapstr.M{"field1": "Hello G%ünter"},
		},
		{
			name:  "missing source field",
			input: mapstr.M{"other": "value"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &urlDecode{
				log: logptest.NewTestingLogger(t, "urldecode"),
				config: urlDecodeConfig{
					Fields: []fromTo{{
						From: "field1", To: "field2",
					}},
					FailOnError: true,
				},
			}

			input := tc.input.Clone()
			event := &beat.Event{Fields: input}
			original := input.Clone()

			result, err := f.Run(event)
			require.Error(t, err)

			result.Fields.Delete("error")
			assert.Equal(t, original, result.Fields,
				"event fields must be unchanged after error (clone skip safety)")
		})
	}
}
