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

package extract_array

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestExtractArrayProcessor_String(t *testing.T) {
	p, err := New(conf.MustNewConfigFrom(mapstr.M{
		"field": "csv",
		"mappings": mapstr.M{
			"source.ip":         0,
			"network.transport": 2,
			"destination.ip":    99,
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "extract_array={field=csv, mappings=[{0 source.ip} {2 network.transport} {99 destination.ip}]}", p.String())
}

func TestExtractArrayProcessor_Run(t *testing.T) {
	tests := map[string]struct {
		config   mapstr.M
		input    beat.Event
		expected beat.Event
		fail     bool
		afterFn  func(e *beat.Event)
	}{
		"sample": {
			config: mapstr.M{
				"field": "array",
				"mappings": mapstr.M{
					"dest.one": 1,
					"dest.two": 2,
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{"zero", 1, mapstr.M{"two": 2}},
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"array":    []interface{}{"zero", 1, mapstr.M{"two": 2}},
					"dest.one": 1,
					"dest.two": mapstr.M{"two": 2},
				},
			},
		},

		"modified elements": {
			config: mapstr.M{
				"field": "array",
				"mappings": mapstr.M{
					"dest.one": 1,
					"dest.two": 2,
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{"zero", 1, mapstr.M{"two": 2}},
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"array":    []interface{}{"zero", 1, mapstr.M{"two": 2}},
					"dest.one": 1,
					"dest.two": mapstr.M{"two": 3},
				},
			},
			afterFn: func(e *beat.Event) {
				e.PutValue("dest.two.two", 3)
			},
		},

		"modified array": {
			config: mapstr.M{
				"field": "array",
				"mappings": mapstr.M{
					"dest.one": 1,
					"dest.two": 2,
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{"zero", 1, []interface{}{"a", "b"}},
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"array":    []interface{}{"zero", 1, []interface{}{"a", "b"}},
					"dest.one": 1,
					"dest.two": []interface{}{"a", "c"},
				},
			},
			afterFn: func(e *beat.Event) {
				val, _ := e.GetValue("dest.two")
				val.([]interface{})[1] = "c"
			},
		},

		"out of range mapping": {
			config: mapstr.M{
				"field": "array",
				"mappings": mapstr.M{
					"source.ip":      0,
					"destination.ip": 999,
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{"127.0.0.1"},
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{"127.0.0.1"},
				},
			},
			fail: true,
		},

		"ignore errors": {
			config: mapstr.M{
				"field": "array",
				"mappings": mapstr.M{
					"a":   0,
					"b.c": 1,
				},
				"fail_on_error": false,
			},
			input: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{3.14, 9000.0},
					"b":     true,
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{3.14, 9000.0},
					"a":     3.14,
					"b":     true,
				},
			},
		},

		"multicopy": {
			config: mapstr.M{
				"field": "array",
				"mappings": mapstr.M{
					"a": 1,
					"b": 1,
					"c": 1,
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{0, 42},
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{0, 42},
					"a":     42,
					"b":     42,
					"c":     42,
				},
			},
		},

		"omit_empty": {
			config: mapstr.M{
				"field": "array",
				"mappings": mapstr.M{
					"a": 0,
					"b": 1,
					"c": 2,
					"d": 3,
					"e": 4,
				},
				"omit_empty": true,
			},
			input: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{0, "", []interface{}(nil), make(map[string]string), 0.0},
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{0, "", []interface{}(nil), make(map[string]string), 0.0},
					"a":     0,
					"e":     0.0,
				},
			},
		},

		"nil values": {
			config: mapstr.M{
				"field": "array",
				"mappings": mapstr.M{
					"a": 0,
					"b": 1,
					"c": 2,
					"d": 3,
					"e": 4,
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{nil, "", []interface{}(nil), map[string]string(nil), (*int)(nil)},
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"array": []interface{}{nil, "", []interface{}(nil), map[string]string(nil), (*int)(nil)},
					"a":     nil,
					"b":     "",
					"c":     []interface{}{},
					"d":     map[string]string(nil),
					"e":     (*int)(nil),
				},
			},
		},
	}
	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(tt.config)
			processor, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			result, err := processor.Run(&tt.input)
			if tt.afterFn != nil {
				tt.afterFn(result)
			}
			if tt.fail {
				assert.Error(t, err)
				t.Log("got expected error", err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.Fields.Flatten(), result.Fields.Flatten())
			assert.Equal(t, tt.expected.Timestamp, result.Timestamp)
			t.Log(result)
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {

		config := mapstr.M{
			"field": "@metadata.array",
			"mappings": mapstr.M{
				"@metadata.first":  1,
				"@metadata.second": 2,
			},
		}

		event := &beat.Event{
			Meta: mapstr.M{
				"array": []interface{}{"zero", 1, mapstr.M{"two": 2}},
			},
		}

		expMeta := mapstr.M{
			"array":  []interface{}{"zero", 1, mapstr.M{"two": 2}},
			"first":  1,
			"second": mapstr.M{"two": 2},
		}

		cfg := conf.MustNewConfigFrom(config)
		processor, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		result, err := processor.Run(event)
		assert.NoError(t, err)

		assert.Equal(t, expMeta, result.Meta)
		assert.Equal(t, event.Fields, result.Fields)
	})
}
