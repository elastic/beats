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

package registrar

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/input/file"
)

func TestRegistrarRead(t *testing.T) {
	type testCase struct {
		input    string
		expected []file.State
	}

	zone := time.FixedZone("+0000", 0)

	cases := map[string]testCase{
		"ok registry with one entry": testCase{
			input: `[
				{
				  "type": "log",
				  "source": "test.log",
				  "offset": 10,
				  "timestamp": "2018-07-16T10:45:01+00:00",
				  "ttl": -1,
				  "meta": null
				}
			]`,
			expected: []file.State{
				{
					Type:      "log",
					Source:    "test.log",
					Timestamp: time.Date(2018, time.July, 16, 10, 45, 01, 0, zone),
					Offset:    10,
					TTL:       -2, // loader always resets states
				},
			},
		},

		"load config without meta": testCase{
			input: `[
				{
				  "type": "log",
				  "source": "test.log",
				  "offset": 10,
				  "timestamp": "2018-07-16T10:45:01+00:00",
				  "ttl": -1
				}
			]`,
			expected: []file.State{
				{
					Type:      "log",
					Source:    "test.log",
					Timestamp: time.Date(2018, time.July, 16, 10, 45, 01, 0, zone),
					Offset:    10,
					TTL:       -2, // loader always resets states
				},
			},
		},

		"load config with empty meta": testCase{
			input: `[
				{
				  "type": "log",
				  "source": "test.log",
				  "offset": 10,
				  "timestamp": "2018-07-16T10:45:01+00:00",
				  "ttl": -1,
					"meta": {}
				}
			]`,
			expected: []file.State{
				{
					Type:      "log",
					Source:    "test.log",
					Timestamp: time.Date(2018, time.July, 16, 10, 45, 01, 0, zone),
					Offset:    10,
					TTL:       -2, // loader always resets states
				},
			},
		},

		"requires merge without meta-data": testCase{
			input: `[
				{
				  "type": "log",
				  "source": "test.log",
				  "offset": 100,
				  "timestamp": "2018-07-16T10:45:01+00:00",
				  "ttl": -1,
				  "meta": {}
				},
				{
				  "type": "log",
				  "source": "test.log",
				  "offset": 10,
				  "timestamp": "2018-07-16T10:45:10+00:00",
				  "ttl": -1,
				  "meta": null
				}
			]`,
			expected: []file.State{
				{
					Type:      "log",
					Source:    "test.log",
					Timestamp: time.Date(2018, time.July, 16, 10, 45, 10, 0, zone),
					Offset:    100,
					TTL:       -2, // loader always resets states
					Meta:      nil,
				},
			},
		},
	}

	matchState := func(t *testing.T, i int, expected, actual file.State) {
		check := func(name string, a, b interface{}) {
			if !reflect.DeepEqual(a, b) {
				t.Errorf("State %v: %v mismatch (expected=%v, actual=%v)", i, name, a, b)
			}
		}

		check("id", expected.ID(), actual.ID())
		check("source", expected.Source, actual.Source)
		check("offset", expected.Offset, actual.Offset)
		check("ttl", expected.TTL, actual.TTL)
		check("meta", expected.Meta, actual.Meta)
		check("type", expected.Type, actual.Type)

		if t1, t2 := expected.Timestamp, actual.Timestamp; !t1.Equal(t2) {
			t.Errorf("State %v: timestamp mismatch (expected=%v, actual=%v)", i, t1, t2)
		}
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			in := strings.NewReader(test.input)

			states, err := readStatesFrom(in)
			if !assert.NoError(t, err) {
				return
			}

			actual := sortedStates(states)
			expected := sortedStates(test.expected)
			if len(actual) != len(expected) {
				t.Errorf("expected %v state, but registrar did load %v states",
					len(expected), len(actual))
				return
			}

			for i := range expected {
				matchState(t, i, expected[i], actual[i])
			}
		})
	}
}

func sortedStates(states []file.State) []file.State {
	tmp := make([]file.State, len(states))
	copy(tmp, states)
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].ID() < tmp[j].ID()
	})
	return tmp
}
