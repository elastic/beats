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

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestCopyFieldsSingleFieldNoClone verifies that a single-field copy
// produces identical results without cloning.
func TestCopyFieldsSingleFieldNoClone(t *testing.T) {
	tests := []struct {
		name   string
		from   string
		to     string
		input  mapstr.M
		output mapstr.M
		err    bool
	}{
		{
			name:  "happy path: message to event.original",
			from:  "message",
			to:    "event.original",
			input: mapstr.M{"message": "hello world"},
			output: mapstr.M{
				"message": "hello world",
				"event":   mapstr.M{"original": "hello world"},
			},
		},
		{
			name:   "source missing",
			from:   "nonexistent",
			to:     "dest",
			input:  mapstr.M{"message": "hello"},
			output: mapstr.M{"message": "hello"},
			err:    true,
		},
		{
			name:   "target already exists",
			from:   "message",
			to:     "existing",
			input:  mapstr.M{"message": "hello", "existing": "already here"},
			output: mapstr.M{"message": "hello", "existing": "already here"},
			err:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := conf.NewConfigFrom(map[string]interface{}{
				"fields": []map[string]interface{}{
					{"from": tc.from, "to": tc.to},
				},
				"fail_on_error": true,
			})
			require.NoError(t, err)

			p, err := NewCopyFields(c, logp.L())
			require.NoError(t, err)

			event := &beat.Event{Fields: tc.input.Clone()}
			result, err := p.Run(event)

			if tc.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			result.Fields.Delete("error")
			assert.Equal(t, tc.output, result.Fields,
				"single-field copy must produce same result without clone")
		})
	}
}
