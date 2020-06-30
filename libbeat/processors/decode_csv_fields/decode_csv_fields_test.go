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
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestDecodeCSVField(t *testing.T) {
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
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": []string{"17", "192.168.33.1", "8.8.8.8"},
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
						"field": "17,192.168.33.1,8.8.8.8",
					},
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"my.field": "17,192.168.33.1,8.8.8.8",
					"message":  []string{"17", "192.168.33.1", "8.8.8.8"},
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
					"message":        `"hello ""world"""`,
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
				"fields": common.MapStr{
					"message": "message",
				},
				"separator": ";",
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "1.5;false;hello world;3",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": []string{"1.5", "false", "hello world", "3"},
				},
			},
		},

		"trim leading space": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "message",
				},
				"trim_leading_space": true,
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": " Here's,   some,   extra ,whitespace",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": []string{"Here's", "some", "extra ", "whitespace"},
				},
			},
		},

		"tab separator": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "message",
				},
				"separator":      "\t",
				"overwrite_keys": true,
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "Tab\tin\tASCII\thas\tthe\t\"decimal\tcharacter\tcode\"\t9",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": []string{"Tab", "in", "ASCII", "has", "the", "decimal\tcharacter\tcode", "9"},
				},
			},
		},

		"unicode separator": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "message",
				},
				"separator":      "🍺",
				"overwrite_keys": true,
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": `🐢🍺🌔🐈🍺🍺🐥🐲`,
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": []string{"🐢", "🌔🐈", "", "🐥🐲"},
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
					"a": "a_csv",
					"b": "b_csv",
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"a": "1,2",
					"b": "hello,world",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"a":     "1,2",
					"b":     "hello,world",
					"a_csv": []string{"1", "2"},
					"b_csv": []string{"hello", "world"},
				},
			},
		},

		"multiple fields failure": {
			config: common.MapStr{
				"fields": common.MapStr{
					"a": "a.csv",
					"b": "b.csv",
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"a": "1,2",
					"b": "hello,world",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"a": "1,2",
					"b": "hello,world",
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
					"a": "1,2",
					"b": "hello,world",
					"c": ":)",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"a": []string{"1", "2"},
					"b": []string{"hello", "world"},
					"c": ":)",
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
					"a": "1,2",
					"b": "hello,world",
					"c": ":)",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"a": "1,2",
					"b": "hello,world",
					"c": ":)",
				},
			},
			fail: true,
		},

		"header given as string": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "csv",
				},
				"headers": common.MapStr{
					"message": common.MapStr{
						"string": "col1,col2,col3",
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"csv":     map[string]string{"col1": "17", "col2": "192.168.33.1", "col3": "8.8.8.8"},
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
		},

		"header for each field given as string": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message1": "message1",
					"message2": "message2",
				},
				"headers": common.MapStr{
					"message1": common.MapStr{
						"string": "col1,col2,col3",
					},
					"message2": common.MapStr{
						"string": "col4,col5,col6",
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message1": "hello,world,!!!",
					"message2": "good,morning,!!!",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message1": map[string]string{"col1": "hello", "col2": "world", "col3": "!!!"},
					"message2": map[string]string{"col4": "good", "col5": "morning", "col6": "!!!"},
				},
			},
		},

		"header for one field given as string": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message1": "message1",
					"message2": "message2",
				},
				"headers": common.MapStr{
					"message2": common.MapStr{
						"string": "col4,col5,col6",
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message1": "hello,world,!!!",
					"message2": "good,morning,!!!",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message1": []string{"hello", "world", "!!!"},
					"message2": map[string]string{"col4": "good", "col5": "morning", "col6": "!!!"},
				},
			},
		},

		"header given as string but have different separator": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "csv",
				},
				"headers": common.MapStr{
					"message": common.MapStr{
						"string": "col1;col2;col3",
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
			fail: true,
		},

		"header given as string but have different length": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "csv",
				},
				"headers": common.MapStr{
					"message": common.MapStr{
						"string": "col1,col2",
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": "17,192.168.33.1,8.8.8.8",
				},
			},
			fail: true,
		},

		"header given in conf file": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "message",
				},
				"headers": common.MapStr{
					"message": common.MapStr{
						"file": common.MapStr{
							"path": "header_test",
						},
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "hello,world",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": map[string]string{"col1_1": "hello", "col1_2": "world"},
				},
			},
		},

		"header given in conf file and the file doesn't exits": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "csv",
				},
				"headers": common.MapStr{
					"message": common.MapStr{
						"file": common.MapStr{
							"path": "file_dont_exists",
						},
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "hello,world",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": "hello,world",
				},
			},
			fail: true,
		},

		"header given in conf file with a offset": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "csv",
				},
				"headers": common.MapStr{
					"message": common.MapStr{
						"offset": 3,
						"file": common.MapStr{
							"path": "header_test",
						},
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "hello,world",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": "hello,world",
					"csv":     map[string]string{"col3_1": "hello", "col3_2": "world"},
				},
			},
		},

		"header given in conf file with too large offset": {
			config: common.MapStr{
				"fields": common.MapStr{
					"message": "csv",
				},
				"headers": common.MapStr{
					"message": common.MapStr{
						"offset": 5,
						"file": common.MapStr{
							"path": "header_test",
						},
					},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"message": "hello,world",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"message": "hello,world",
				},
			},
			fail: true,
		},
	}

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			processor, err := NewDecodeCSVField(common.MustNewConfigFrom(tt.config))
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

func TestDecodeCSVField_String(t *testing.T) {
	p, err := NewDecodeCSVField(common.MustNewConfigFrom(common.MapStr{
		"fields": common.MapStr{
			"a": "csv.a",
			"b": "csv.b",
		},
		"separator":      "#",
		"ignore_missing": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "decode_csv_field={\"Fields\":{\"a\":\"csv.a\",\"b\":\"csv.b\"},\"IgnoreMissing\":true,\"TrimLeadingSpace\":false,\"OverwriteKeys\":false,\"FailOnError\":true,\"Separator\":\"#\",\"Headers\":null}", p.String())
}
