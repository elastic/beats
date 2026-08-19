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

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/libbeat/internal/testutil"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNewLowerCaseProcessor(t *testing.T) {
	c := conf.MustNewConfigFrom(
		mapstr.M{
			"fields":         []string{"field1", "type", "field2", "type.value.key", "typeKey"}, // "type" is our mandatory field
			"ignore_missing": true,
			"fail_on_error":  false,
		},
	)

	procInt, err := NewLowerCaseProcessor(c, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)

	processor, ok := procInt.(*alterFieldProcessor)
	assert.True(t, ok)
	assert.Equal(t, []string{"field1", "field2", "typeKey"}, processor.Fields) // we discard both "type" and "type.value.key" as mandatory fields
	assert.True(t, processor.IgnoreMissing)
	assert.False(t, processor.FailOnError)
}

func TestLowerCaseProcessorRun(t *testing.T) {
	tests := []struct {
		Name          string
		Fields        []string
		IgnoreMissing bool
		FailOnError   bool
		FullPath      bool
		Input         mapstr.M
		Output        mapstr.M
		Error         bool
	}{
		{
			Name:          "Lowercase Fields",
			Fields:        []string{"a.b.c", "Field1"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      true,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"C": "D",
					},
				},
			},
			Output: mapstr.M{
				"field1": mapstr.M{"Field2": "Value"}, // field1 is lowercased
				"Field3": "Value",
				"a": mapstr.M{
					"b": mapstr.M{
						"c": "D",
					},
				},
			},
			Error: false,
		},
		{
			Name:          "Lowercase Fields when full_path is false", // searches only the most nested key 'case insensitively'
			Fields:        []string{"a.B.c"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      false,
			Input: mapstr.M{
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"C": "D",
					},
				},
			},
			Output: mapstr.M{
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"c": "D", // only c is lowercased
					},
				},
			},

			Error: false,
		},
		{
			Name:          "Revert to original map on error",
			Fields:        []string{"Field1", "abcbd"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      true,
			Input: mapstr.M{
				"Field1": "value1",
				"ab":     "first",
			},
			Output: mapstr.M{
				"Field1": "value1",
				"ab":     "first",
				"error":  mapstr.M{"message": "could not fetch value for key: abcbd, Error: key not found"},
			},
			Error: true,
		},
		{
			Name:          "Ignore Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: true,
			FailOnError:   true,
			FullPath:      true,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Output: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Error: false,
		},
		{
			Name:          "Do Not Fail On Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: false,
			FailOnError:   false,
			FullPath:      true,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Output: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Error: false,
		},
		{
			Name:          "Fail On Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      true,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Output: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
				"error":  mapstr.M{"message": "could not fetch value for key: Field4, Error: key not found"},
			},
			Error: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			p := &alterFieldProcessor{
				Fields:         test.Fields,
				IgnoreMissing:  test.IgnoreMissing,
				FailOnError:    test.FailOnError,
				AlterFullField: test.FullPath,
				alterFunc:      lowerCase,
			}

			event, err := p.Run(&beat.Event{Fields: test.Input})

			if !test.Error {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			assert.Equal(t, test.Output, event.Fields)
		})
	}

	t.Run("test key collison", func(t *testing.T) {
		Input :=
			mapstr.M{
				"ab": "first",
				"Ab": "second",
			}

		p := &alterFieldProcessor{
			Fields:         []string{"ab"},
			IgnoreMissing:  false,
			FailOnError:    true,
			AlterFullField: true,
			alterFunc:      lowerCase,
		}

		_, err := p.Run(&beat.Event{Fields: Input})
		require.Error(t, err)
		assert.ErrorIs(t, err, mapstr.ErrKeyCollision)

	})
}

func TestLowerCaseProcessorValues(t *testing.T) {
	tests := []struct {
		Name          string
		Values        []string
		IgnoreMissing bool
		FailOnError   bool
		FullPath      bool
		Input         mapstr.M
		Output        mapstr.M
		Error         bool
	}{
		{
			Name:          "Lowercase Values",
			Values:        []string{"a.b.c"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      true,
			Input: mapstr.M{
				"a": mapstr.M{
					"b": mapstr.M{
						"c": "D",
					},
				},
			},
			Output: mapstr.M{
				"a": mapstr.M{
					"b": mapstr.M{
						"c": "d", // d is lowercased
					},
				},
			},
			Error: false,
		},
		{
			Name:          "Fail if given path to value is not a string",
			Values:        []string{"a.B"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      true,
			Input: mapstr.M{
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"C": "D",
					},
				},
			},
			Output: mapstr.M{
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"C": "D",
					},
				},
				"error": mapstr.M{"message": "value of key \"a.B\" is not a string"},
			},

			Error: true,
		},
		{
			Name:          "Fail On Missing Key Error",
			Values:        []string{"a.B.c"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      true,
			Input: mapstr.M{
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"C": "D",
					},
				},
			},
			Output: mapstr.M{
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"C": "D",
					},
				},
				"error": mapstr.M{"message": "could not fetch value for key: a.B.c, Error: key not found"},
			},

			Error: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			p := &alterFieldProcessor{
				Values:         test.Values,
				IgnoreMissing:  test.IgnoreMissing,
				FailOnError:    test.FailOnError,
				AlterFullField: test.FullPath,
				alterFunc:      lowerCase,
			}

			event, err := p.Run(&beat.Event{Fields: test.Input})

			if !test.Error {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			assert.Equal(t, test.Output, event.Fields)
		})
	}
}
func BenchmarkLowerCaseProcessorRun(b *testing.B) {
	tests := []struct {
		Name   string
		Events []beat.Event
	}{
		{
			Name:   "5000 events with 5 fields on each level with 3 level depth",
			Events: testutil.GenerateEvents(5000, 5, 3),
		},
		{
			Name:   "500 events with 50 fields on each level with 5 level depth",
			Events: testutil.GenerateEvents(500, 50, 3),
		},

		// Add more test cases as needed for benchmarking
	}

	for _, tt := range tests {
		b.Run(tt.Name, func(b *testing.B) {
			p := &alterFieldProcessor{
				Fields:         []string{"level1field1.level2field1.level3field1"},
				alterFunc:      lowerCase,
				AlterFullField: true,
				IgnoreMissing:  false,
				FailOnError:    true,
			}
			for i := 0; i < b.N; i++ {
				//Run the function with the input
				for _, e := range tt.Events {
					ev := e
					_, err := p.Run(&ev)
					require.NoError(b, err)
				}

			}
		})
	}
}
