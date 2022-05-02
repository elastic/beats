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

package decode_csv_fields

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestDecodeCSVField(t *testing.T) {
	tests := map[string]struct {
		config   mapstr.M
		input    beat.Event
		expected beat.Event
		fail     bool
	}{
		"self target": {
			config: mapstr.M{
				"fields": mapstr.M{
					"message": "message",
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"message": []string{"17", "192.168.33.1", "8.8.8.8"},
				},
			},
		},

		"alternative target": {
			config: mapstr.M{
				"fields": mapstr.M{
					"my": mapstr.M{
						"field": "message",
					},
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"my": mapstr.M{
						"field": "17,192.168.33.1,8.8.8.8",
					},
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"my.field": "17,192.168.33.1,8.8.8.8",
					"message":  []string{"17", "192.168.33.1", "8.8.8.8"},
				},
			},
		},

		"non existing field": {
			config: mapstr.M{
				"fields": mapstr.M{
					"field": "my.field",
				},
			},
			fail: true,
		},

		"ignore missing": {
			config: mapstr.M{
				"fields": mapstr.M{
					"my_field": "my_field",
				},

				"ignore_missing": true,
			},
		},

		"overwrite keys failure": {
			config: mapstr.M{
				"fields": mapstr.M{
					"message": "existing_field",
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"message":        `"hello ""world"""`,
					"existing_field": 42,
				},
			},
			fail: true,
		},

		"overwrite keys": {
			config: mapstr.M{
				"fields": mapstr.M{
					"message": "existing_field",
				},
				"overwrite_keys": true,
			},
			input: beat.Event{
				Fields: mapstr.M{
					"message":        `"hello ""world"""`,
					"existing_field": 42,
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"message":        `"hello ""world"""`,
					"existing_field": []string{`hello "world"`},
				},
			},
		},

		"custom separator": {
			config: mapstr.M{
				"fields": mapstr.M{
					"message": "message",
				},
				"separator": ";",
			},
			input: beat.Event{
				Fields: mapstr.M{
					"message": "1.5;false;hello world;3",
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"message": []string{"1.5", "false", "hello world", "3"},
				},
			},
		},

		"trim leading space": {
			config: mapstr.M{
				"fields": mapstr.M{
					"message": "message",
				},
				"trim_leading_space": true,
			},
			input: beat.Event{
				Fields: mapstr.M{
					"message": " Here's,   some,   extra ,whitespace",
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"message": []string{"Here's", "some", "extra ", "whitespace"},
				},
			},
		},

		"tab separator": {
			config: mapstr.M{
				"fields": mapstr.M{
					"message": "message",
				},
				"separator":      "\t",
				"overwrite_keys": true,
			},
			input: beat.Event{
				Fields: mapstr.M{
					"message": "Tab\tin\tASCII\thas\tthe\t\"decimal\tcharacter\tcode\"\t9",
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"message": []string{"Tab", "in", "ASCII", "has", "the", "decimal\tcharacter\tcode", "9"},
				},
			},
		},

		"unicode separator": {
			config: mapstr.M{
				"fields": mapstr.M{
					"message": "message",
				},
				"separator":      "🍺",
				"overwrite_keys": true,
			},
			input: beat.Event{
				Fields: mapstr.M{
					"message": `🐢🍺🌔🐈🍺🍺🐥🐲`,
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"message": []string{"🐢", "🌔🐈", "", "🐥🐲"},
				},
			},
		},

		"bad type": {
			config: mapstr.M{
				"fields": mapstr.M{
					"message": "message",
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"message": 42,
				},
			},
			fail: true,
		},

		"multiple fields": {
			config: mapstr.M{
				"fields": mapstr.M{
					"a": "a_csv",
					"b": "b_csv",
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"a": "1,2",
					"b": "hello,world",
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"a":     "1,2",
					"b":     "hello,world",
					"a_csv": []string{"1", "2"},
					"b_csv": []string{"hello", "world"},
				},
			},
		},

		"multiple fields failure": {
			config: mapstr.M{
				"fields": mapstr.M{
					"a": "a.csv",
					"b": "b.csv",
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"a": "1,2",
					"b": "hello,world",
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"a": "1,2",
					"b": "hello,world",
				},
			},
			fail: true,
		},

		"ignore errors": {
			config: mapstr.M{
				"fields": mapstr.M{
					"a": "a",
					"b": "b",
					"c": "a.b",
				},
				"fail_on_error": false,
			},
			input: beat.Event{
				Fields: mapstr.M{
					"a": "1,2",
					"b": "hello,world",
					"c": ":)",
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"a": []string{"1", "2"},
					"b": []string{"hello", "world"},
					"c": ":)",
				},
			},
		},

		"restore on errors": {
			config: mapstr.M{
				"fields": mapstr.M{
					"a": "a",
					"b": "b",
					"c": "a.b",
				},
				"fail_on_error": true,
			},
			input: beat.Event{
				Fields: mapstr.M{
					"a": "1,2",
					"b": "hello,world",
					"c": ":)",
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"a": "1,2",
					"b": "hello,world",
					"c": ":)",
				},
			},
			fail: true,
		},
	}

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			processor, err := NewDecodeCSVField(cfg.MustNewConfigFrom(tt.config))
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

	t.Run("supports metadata as a target", func(t *testing.T) {
		config := mapstr.M{
			"fields": mapstr.M{
				"@metadata": mapstr.M{
					"field": "@metadata.message",
				},
			},
		}

		event := &beat.Event{
			Meta: mapstr.M{
				"field": "17,192.168.33.1,8.8.8.8",
			},
			Fields: mapstr.M{},
		}
		expMeta := mapstr.M{
			"field":   "17,192.168.33.1,8.8.8.8",
			"message": []string{"17", "192.168.33.1", "8.8.8.8"},
		}

		processor, err := NewDecodeCSVField(cfg.MustNewConfigFrom(config))
		if err != nil {
			t.Fatal(err)
		}
		result, err := processor.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, result.Meta)
		assert.Equal(t, event.Fields, result.Fields)
	})

}

func TestDecodeCSVField_String(t *testing.T) {
	p, err := NewDecodeCSVField(cfg.MustNewConfigFrom(mapstr.M{
		"fields": mapstr.M{
			"a": "csv.a",
			"b": "csv.b",
		},
		"separator":      "#",
		"ignore_missing": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "decode_csv_field={\"Fields\":{\"a\":\"csv.a\",\"b\":\"csv.b\"},\"IgnoreMissing\":true,\"TrimLeadingSpace\":false,\"OverwriteKeys\":false,\"FailOnError\":true,\"Separator\":\"#\"}", p.String())
}
