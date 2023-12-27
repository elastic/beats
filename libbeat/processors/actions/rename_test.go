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
	"reflect"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestRenameRun(t *testing.T) {
	log := logp.NewLogger("rename_test")
	var tests = []struct {
		description   string
		Fields        []fromTo
		IgnoreMissing bool
		FailOnError   bool
		Input         mapstr.M
		Output        mapstr.M
		error         bool
	}{
		{
			description: "simple field renaming",
			Fields: []fromTo{
				{
					From: "a",
					To:   "b",
				},
			},
			Input: mapstr.M{
				"a": "c",
			},
			Output: mapstr.M{
				"b": "c",
			},
			IgnoreMissing: false,
			FailOnError:   true,
			error:         false,
		},
		{
			description: "Add one more hierarchy to event",
			Fields: []fromTo{
				{
					From: "a.b",
					To:   "a.b.c",
				},
			},
			Input: mapstr.M{
				"a.b": 1,
			},
			Output: mapstr.M{
				"a": mapstr.M{
					"b": mapstr.M{
						"c": 1,
					},
				},
			},
			IgnoreMissing: false,
			FailOnError:   true,
			error:         false,
		},
		{
			description: "overwrites an existing field which is not allowed",
			Fields: []fromTo{
				{
					From: "a",
					To:   "b",
				},
			},
			Input: mapstr.M{
				"a": 2,
				"b": "q",
			},
			Output: mapstr.M{
				"a": 2,
				"b": "q",
				"error": mapstr.M{
					"message": "Failed to rename fields in processor: target field b already exists, drop or rename this field first",
				},
			},
			error:         true,
			FailOnError:   true,
			IgnoreMissing: false,
		},
		{
			description: "overwrites existing field but renames it first, order matters",
			Fields: []fromTo{
				{
					From: "b",
					To:   "c",
				},
				{
					From: "a",
					To:   "b",
				},
			},
			Input: mapstr.M{
				"a": 2,
				"b": "q",
			},
			Output: mapstr.M{
				"b": 2,
				"c": "q",
			},
			error:         false,
			FailOnError:   true,
			IgnoreMissing: false,
		},
		{
			description: "take an invalid ES event with key / object conflict and convert it to a valid event",
			Fields: []fromTo{
				{
					From: "a",
					To:   "a.value",
				},
			},
			Input: mapstr.M{
				"a":   5,
				"a.b": 6,
			},
			Output: mapstr.M{
				"a.b": 6,
				"a": mapstr.M{
					"value": 5,
				},
			},
			error:         false,
			FailOnError:   true,
			IgnoreMissing: false,
		},
		{
			description: "renames two fields into the same namespace. order matters as a is first key and then object",
			Fields: []fromTo{
				{
					From: "a",
					To:   "a.value",
				},
				{
					From: "c",
					To:   "a.c",
				},
			},
			Input: mapstr.M{
				"a": 7,
				"c": 8,
			},
			Output: mapstr.M{
				"a": mapstr.M{
					"value": 7,
					"c":     8,
				},
			},
			error:         false,
			IgnoreMissing: false,
			FailOnError:   true,
		},
		{
			description: "rename two fields into the same name space. this fails because a is already a key, renaming of a needs to happen first",
			Fields: []fromTo{
				{
					From: "c",
					To:   "a.c",
				},
				{
					From: "a",
					To:   "a.value",
				},
			},
			Input: mapstr.M{
				"a": 9,
				"c": 10,
			},
			Output: mapstr.M{
				"a": 9,
				"c": 10,
				"error": mapstr.M{
					"message": "Failed to rename fields in processor: could not put value: a.c: 10, expected map but type is int",
				},
			},
			error:         true,
			IgnoreMissing: false,
			FailOnError:   true,
		},
		{
			description: "renames conflicting keys. partially works because fail_on_error is false",
			Fields: []fromTo{
				{
					From: "c",
					To:   "a.c",
				},
				{
					From: "a",
					To:   "a.value",
				},
			},
			Input: mapstr.M{
				"a": 9,
				"c": 10,
			},
			Output: mapstr.M{
				"a": mapstr.M{
					"value": 9,
				},
			},
			error:         false,
			IgnoreMissing: false,
			FailOnError:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			f := &renameFields{
				config: renameFieldsConfig{
					Fields:        test.Fields,
					IgnoreMissing: test.IgnoreMissing,
					FailOnError:   test.FailOnError,
				},
				logger: log,
			}
			event := &beat.Event{
				Fields: test.Input,
			}

			newEvent, err := f.Run(event)
			if !test.error {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.True(t, reflect.DeepEqual(newEvent.Fields, test.Output))
		})
	}
}

func TestRenameField(t *testing.T) {
	var tests = []struct {
		From          string
		To            string
		ignoreMissing bool
		failOnError   bool
		Input         mapstr.M
		Output        mapstr.M
		error         bool
		description   string
	}{
		{
			description: "simple rename of field",
			From:        "a",
			To:          "c",
			Input: mapstr.M{
				"a": "b",
			},
			Output: mapstr.M{
				"c": "b",
			},
			error:         false,
			failOnError:   true,
			ignoreMissing: false,
		},
		{
			description: "Add hierarchy to event",
			From:        "a.b",
			To:          "a.b.c",
			Input: mapstr.M{
				"a.b": 1,
			},
			Output: mapstr.M{
				"a": mapstr.M{
					"b": mapstr.M{
						"c": 1,
					},
				},
			},
			error:         false,
			failOnError:   true,
			ignoreMissing: false,
		},
		{
			description: "overwrite an existing field that should lead to an error",
			From:        "a",
			To:          "b",
			Input: mapstr.M{
				"a": 2,
				"b": "q",
			},
			Output: mapstr.M{
				"a": 2,
				"b": "q",
			},
			error:         true,
			failOnError:   true,
			ignoreMissing: false,
		},
		{
			description: "resolve dotted event conflict",
			From:        "a",
			To:          "a.value",
			Input: mapstr.M{
				"a":   5,
				"a.b": 6,
			},
			Output: mapstr.M{
				"a.b": 6,
				"a": mapstr.M{
					"value": 5,
				},
			},
			error:         false,
			failOnError:   true,
			ignoreMissing: false,
		},
		{
			description: "try to rename no existing field with failOnError true",
			From:        "a",
			To:          "b",
			Input: mapstr.M{
				"c": 5,
			},
			Output: mapstr.M{
				"c": 5,
			},
			failOnError:   true,
			ignoreMissing: false,
			error:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			f := &renameFields{
				config: renameFieldsConfig{
					IgnoreMissing: test.ignoreMissing,
					FailOnError:   test.failOnError,
				},
			}

			err := f.renameField(test.From, test.To, &beat.Event{Fields: test.Input})
			if err != nil {
				assert.Equal(t, test.error, true)
			}

			assert.True(t, reflect.DeepEqual(test.Input, test.Output))
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		event := &beat.Event{
			Meta: mapstr.M{
				"a": "c",
			},
		}

		expMeta := mapstr.M{
			"b": "c",
		}

		f := &renameFields{
			config: renameFieldsConfig{
				Fields: []fromTo{
					{
						From: "@metadata.a",
						To:   "@metadata.b",
					},
				},
			},
		}

		newEvent, err := f.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, newEvent.Meta)
		assert.Equal(t, event.Fields, newEvent.Fields)
	})
}
