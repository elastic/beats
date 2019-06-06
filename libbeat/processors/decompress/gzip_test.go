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

package gzip

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestDecompressGzipField(t *testing.T) {
	tests := map[string]struct {
		config   common.MapStr
		input    beat.Event
		expected beat.Event
		fail     bool
	}{
		"self target": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "message",
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": "hello world",
				},
			},
		},

		"alternative target": {
			config: common.MapStr{
				"fields": common.MapStr{
					"my": common.MapStr{
						"field": "message",
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"my": common.MapStr{
						"field": string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					},
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"my.field": string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"message":  "hello world",
				},
			},
		},

		"non existing field": {
			config: common.MapStr{
				"fields": common.MapStr{
					"field": "my.field",
				},
			},
			fail: true,
		},

		"ignore missing": {
			config: common.MapStr{
				"fields": common.MapStr{
					"my_field": "my_field",
				},

				"ignore_missing": true,
			},
		},

		"overwrite keys failure": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "existing_field",
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message":        string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"existing_field": 42,
				},
			},
			fail: true,
		},

		"overwrite keys": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "existing_field",
				},
				"overwrite_keys": true,
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message":        string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"existing_field": 42,
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message":        string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"existing_field": "hello world",
				},
			},
		},

		"bad type": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "message",
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": 42,
				},
			},
			fail: true,
		},

		"multiple fields": {
			config: common.MapStr{
				"fields": common.MapStr{
					"a": "a_uncompress",
					"b": "b_uncompress",
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"a": string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"b": string([]byte{31, 139, 8, 0, 20, 84, 233, 92, 0, 3, 203, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 49, 2, 0, 179, 131, 10, 135, 12, 0, 0, 0}),
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"a":            string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"b":            string([]byte{31, 139, 8, 0, 20, 84, 233, 92, 0, 3, 203, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 49, 2, 0, 179, 131, 10, 135, 12, 0, 0, 0}),
					"a_uncompress": "hello world",
					"b_uncompress": "hello world2",
				},
			},
		},

		"multiple fields failure": {
			config: common.MapStr{
				"fields": common.MapStr{
					"a": "a.decompressed",
					"b": "b.decompressed",
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"a": string([]byte{139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"b": string([]byte{139, 8, 0, 20, 84, 233, 92, 0, 3, 203, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 49, 2, 0, 179, 131, 10, 135, 12, 0, 0, 0}),
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"a": string([]byte{139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"b": string([]byte{139, 8, 0, 20, 84, 233, 92, 0, 3, 203, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 49, 2, 0, 179, 131, 10, 135, 12, 0, 0, 0}),
				},
			},
			fail: true,
		},

		"ignore errors": {
			config: common.MapStr{
				"fields": common.MapStr{
					"a": "a",
					"b": "b",
					"c": "a.b",
				},
				"fail_on_error": false,
			},
			input: beat.Event{
				Fields: common.MapStr{
					"a": string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"b": string([]byte{31, 139, 8, 0, 20, 84, 233, 92, 0, 3, 203, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 49, 2, 0, 179, 131, 10, 135, 12, 0, 0, 0}),
					"c": string([]byte{139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"a": "hello world",
					"b": "hello world2",
					"c": string([]byte{139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
				},
			},
		},

		"restore on errors": {
			config: common.MapStr{
				"fields": common.MapStr{
					"a": "a",
					"b": "b",
					"c": "a.b",
				},
				"fail_on_error": true,
			},
			input: beat.Event{
				Fields: common.MapStr{
					"a": string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"b": string([]byte{31, 139, 8, 0, 20, 84, 233, 92, 0, 3, 203, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 49, 2, 0, 179, 131, 10, 135, 12, 0, 0, 0}),
					"c": string([]byte{139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"a": string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
					"b": string([]byte{31, 139, 8, 0, 20, 84, 233, 92, 0, 3, 203, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 49, 2, 0, 179, 131, 10, 135, 12, 0, 0, 0}),
					"c": string([]byte{139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 72, 205, 201, 201, 87, 40, 207, 47, 202, 73, 1, 0, 0, 0, 255, 255, 1, 0, 0, 255, 255, 133, 17, 74, 13, 11, 0, 0, 0}),
				},
			},
			fail: true,
		},
	}

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			processor, err := NewDecompressGzipFields(common.MustNewConfigFrom(tt.config))
			if err != nil {
				t.Fatal(err)
			}
			result, err := processor.Run(&tt.input)
			if tt.expected.Fields != nil {
				assert.Equal(t, tt.expected.Fields.Flatten(), result.Fields.Flatten())
				assert.Equal(t, tt.expected.Meta.Flatten(), result.Meta.Flatten())
				assert.Equal(t, tt.expected.Timestamp, result.Timestamp)
			}
			if tt.fail {
				assert.Error(t, err)
				t.Log("got expected error", err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestDecompressGzipField_String(t *testing.T) {
	p, err := NewDecompressGzipFields(common.MustNewConfigFrom(common.MapStr{
		"fields": common.MapStr{
			"a": "decompress.a",
			"b": "decompress.b",
		},
		"ignore_missing": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "decompress_gzip_fields={\"Fields\":{\"a\":\"decompress.a\",\"b\":\"decompress.b\"},\"IgnoreMissing\":true,\"OverwriteKeys\":false,\"FailOnError\":true}", p.String())
}
