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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestDecodeBase64(t *testing.T) {
	testCases := []struct {
		desc        string
		config      common.MapStr
		input       *beat.Event
		errExpected bool
		expected    common.MapStr
	}{
		{
			desc: "bad format",
			config: common.MapStr{
				"field": "field1",
			},
			input: &beat.Event{
				Fields: common.MapStr{
					"field1": "bad data",
				},
			},
			errExpected: true,
			expected: common.MapStr{
				"field1": "bad data",
			},
		},
		{
			desc: "correct format",
			config: common.MapStr{
				"field": "field1",
			},
			input: &beat.Event{
				Fields: common.MapStr{
					"field1": "Y29ycmVjdCBkYXRh",
				},
			},
			expected: common.MapStr{
				"field1": "correct data",
			},
		},
		{
			desc: "empty data",
			config: common.MapStr{
				"field": "field1",
			},
			input: &beat.Event{
				Fields: common.MapStr{
					"field1": "",
				},
			},
			expected: common.MapStr{
				"field1": "",
			},
		},
		{
			desc: "empty target",
			config: common.MapStr{
				"field":  "field1",
				"target": "",
			},
			input: &beat.Event{
				Fields: common.MapStr{
					"field1": "Y29ycmVjdCBkYXRh",
				},
			},
			expected: common.MapStr{
				"field1": "Y29ycmVjdCBkYXRh",
			},
		},
		{
			desc: "correct target",
			config: common.MapStr{
				"field":  "field1",
				"target": "field2",
			},
			input: &beat.Event{
				Fields: common.MapStr{
					"field1": "Y29ycmVjdCBkYXRh",
				},
			},
			expected: common.MapStr{
				"field1": "Y29ycmVjdCBkYXRh",
				"field2": "correct data",
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			cfg, err := common.NewConfigFrom(test.config)
			require.NoError(t, err)

			processor, err := NewDecodeBase64Field(cfg)
			require.NoError(t, err)

			outputEvent, err := processor.Run(test.input)

			if test.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.expected, outputEvent.Fields)
		})
	}
}
