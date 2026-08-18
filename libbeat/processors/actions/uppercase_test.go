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
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNewUpperCaseProcessor(t *testing.T) {
	c := conf.MustNewConfigFrom(
		mapstr.M{
			"fields":         []string{"field1", "type", "field2", "type.value.key", "typeKey"}, // "type" is our mandatory field
			"ignore_missing": true,
			"fail_on_error":  false,
		},
	)

	procInt, err := NewUpperCaseProcessor(c, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)

	processor, ok := procInt.(*alterFieldProcessor)
	assert.True(t, ok)
	assert.Equal(t, []string{"field1", "field2", "typeKey"}, processor.Fields) // we discard both "type" and "type.value.key" as mandatory fields
	assert.True(t, processor.IgnoreMissing)
	assert.False(t, processor.FailOnError)
}

func TestUpperCaseProcessorRun(t *testing.T) {
	tests := []struct {
		Name          string
		Fields        []string
		Values        []string
		IgnoreMissing bool
		FailOnError   bool
		FullPath      bool
		Input         mapstr.M
		Output        mapstr.M
		Error         bool
	}{
		{
			Name:          "Uppercase Fields",
			Fields:        []string{"a.b.c", "Field1"},
			Values:        []string{"Field3"},
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
				"FIELD1": mapstr.M{"Field2": "Value"}, // FIELD1 is uppercased
				"Field3": "VALUE",                     // VALUE is uppercased
				"A": mapstr.M{
					"B": mapstr.M{
						"C": "D",
					},
				},
			},
			Error: false,
		},
		{
			Name:          "Uppercase Fields when full_path is false", // searches only the most nested key 'case insensitively'
			Fields:        []string{"a.B.c"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      false,
			Input: mapstr.M{
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"c": "D",
					},
				},
			},
			Output: mapstr.M{
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"C": "D", // only c is uppercased
					},
				},
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
		{
			Name:          "Fail if value is not a string",
			Values:        []string{"Field1"},
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
				"error":  mapstr.M{"message": "value of key \"Field1\" is not a string"},
			},
			Error: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			p := &alterFieldProcessor{
				Fields:         test.Fields,
				Values:         test.Values,
				IgnoreMissing:  test.IgnoreMissing,
				FailOnError:    test.FailOnError,
				AlterFullField: test.FullPath,
				alterFunc:      upperCase,
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
			alterFunc:      upperCase,
		}

		_, err := p.Run(&beat.Event{Fields: Input})
		require.Error(t, err)
		assert.ErrorIs(t, err, mapstr.ErrKeyCollision)

	})
}
