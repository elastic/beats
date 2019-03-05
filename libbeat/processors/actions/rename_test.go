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

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestRenameRun(t *testing.T) {
	var tests = []struct {
		description   string
		Fields        []fromTo
		IgnoreMissing bool
		FailOnError   bool
		Input         common.MapStr
		Output        common.MapStr
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
			Input: common.MapStr{
				"a": "c",
			},
			Output: common.MapStr{
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
			Input: common.MapStr{
				"a.b": 1,
			},
			Output: common.MapStr{
				"a": common.MapStr{
					"b": common.MapStr{
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
			Input: common.MapStr{
				"a": 2,
				"b": "q",
			},
			Output: common.MapStr{
				"a": 2,
				"b": "q",
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
			Input: common.MapStr{
				"a": 2,
				"b": "q",
			},
			Output: common.MapStr{
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
			Input: common.MapStr{
				"a":   5,
				"a.b": 6,
			},
			Output: common.MapStr{
				"a.b": 6,
				"a": common.MapStr{
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
			Input: common.MapStr{
				"a": 7,
				"c": 8,
			},
			Output: common.MapStr{
				"a": common.MapStr{
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
			Input: common.MapStr{
				"a": 9,
				"c": 10,
			},
			Output: common.MapStr{
				"a": 9,
				"c": 10,
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
			Input: common.MapStr{
				"a": 9,
				"c": 10,
			},
			Output: common.MapStr{
				"a": common.MapStr{
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
			}
			event := &beat.Event{
				Fields: test.Input,
			}

			newEvent, err := f.Run(event)
			if !test.error {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
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
		Input         common.MapStr
		Output        common.MapStr
		error         bool
		description   string
	}{
		{
			description: "simple rename of field",
			From:        "a",
			To:          "c",
			Input: common.MapStr{
				"a": "b",
			},
			Output: common.MapStr{
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
			Input: common.MapStr{
				"a.b": 1,
			},
			Output: common.MapStr{
				"a": common.MapStr{
					"b": common.MapStr{
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
			Input: common.MapStr{
				"a": 2,
				"b": "q",
			},
			Output: common.MapStr{
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
			Input: common.MapStr{
				"a":   5,
				"a.b": 6,
			},
			Output: common.MapStr{
				"a.b": 6,
				"a": common.MapStr{
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
			Input: common.MapStr{
				"c": 5,
			},
			Output: common.MapStr{
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

			err := f.renameField(test.From, test.To, test.Input)
			if err != nil {
				assert.Equal(t, test.error, true)
			}

			assert.True(t, reflect.DeepEqual(test.Input, test.Output))
		})
	}
}
