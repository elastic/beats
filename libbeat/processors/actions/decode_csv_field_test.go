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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestDecodeCSVField(t *testing.T) {
	tests := map[string]struct {
		config   common.MapStr
		input    beat.Event
		expected beat.Event
		fail     bool
	}{
		"default target": {
			config: common.MapStr{
				"field": "message",
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": "17,192.168.33.1,8.8.8.8",
					"csv":     []string{"17", "192.168.33.1", "8.8.8.8"},
				},
			},
		},
		"alternative target": {
			config: common.MapStr{
				"field":  "message",
				"target": "my.field",
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message":  "17,192.168.33.1,8.8.8.8",
					"my.field": []string{"17", "192.168.33.1", "8.8.8.8"},
				},
			},
		},
		"no field set": {
			fail: true,
		},
		"non existing field": {
			config: common.MapStr{
				"field": "my.field",
			},
			fail: true,
		},
		"ignore missing": {
			config: common.MapStr{
				"field":          "my_field",
				"ignore_missing": true,
			},
		},
		"overwrite keys failure": {
			config: common.MapStr{
				"field":  "message",
				"target": "existing_field",
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message":        `"hello ""world"""`,
					"existing_field": 42,
				},
			},
			fail: true,
		},

		"overwrite keys": {
			config: common.MapStr{
				"field":          "message",
				"target":         "existing_field",
				"overwrite_keys": true,
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message":        `"hello ""world"""`,
					"existing_field": 42,
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message":        `"hello ""world"""`,
					"existing_field": []string{`hello "world"`},
				},
			},
		},

		"custom separator": {
			config: common.MapStr{
				"field":     "message",
				"separator": ";",
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "1.5;false;hello world;3",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": "1.5;false;hello world;3",
					"csv":     []string{"1.5", "false", "hello world", "3"},
				},
			},
		},

		"trim leading space": {
			config: common.MapStr{
				"field":              "message",
				"trim_leading_space": true,
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": " Here's,   some,   extra ,whitespace",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": " Here's,   some,   extra ,whitespace",
					"csv":     []string{"Here's", "some", "extra ", "whitespace"},
				},
			},
		},
	}

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			processor, err := NewDecodeCSVField(common.MustNewConfigFrom(tt.config))
			if err != nil {
				t.Fatal(err)
			}
			result, err := processor.Run(&tt.input)
			if tt.fail {
				assert.Error(t, err)
				t.Log("got expected error", err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.Fields.Flatten(), result.Fields.Flatten())
			assert.Equal(t, tt.expected.Meta.Flatten(), result.Meta.Flatten())
			assert.Equal(t, tt.expected.Timestamp, result.Timestamp)
			t.Log(result)
		})
	}
}

func TestDecodeCSVField_String(t *testing.T) {
	p, err := NewDecodeCSVField(common.MustNewConfigFrom(common.MapStr{
		"field":          "source",
		"target":         "destination",
		"separator":      "#",
		"ignore_missing": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "decode_csv_field={field:source,target:destination,separator:'#',ignore_missing:true,trim_leading_space:false,overwrite_keys:false", p.String())
}
