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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestReplaceRun(t *testing.T) {
	var tests = []struct {
		description   string
		Fields        []replaceConfig
		IgnoreMissing bool
		FailOnError   bool
		Input         common.MapStr
		Output        common.MapStr
		error         bool
	}{
		{
			description: "simple field replacing",
			Fields: []replaceConfig{
				{
					Field:       "f",
					Pattern:     regexp.MustCompile(`a`),
					Replacement: "b",
				},
			},
			Input: common.MapStr{
				"f": "abc",
			},
			Output: common.MapStr{
				"f": "bbc",
			},
			error:         false,
			IgnoreMissing: false,
			FailOnError:   true,
		},
		{
			description: "Add one more hierarchy to event",
			Fields: []replaceConfig{
				{
					Field:       "f.b",
					Pattern:     regexp.MustCompile(`a`),
					Replacement: "b",
				},
			},
			Input: common.MapStr{
				"f": common.MapStr{
					"b": "abc",
				},
			},
			Output: common.MapStr{
				"f": common.MapStr{
					"b": "bbc",
				},
			},
			error:         false,
			IgnoreMissing: false,
			FailOnError:   true,
		},
		{
			description: "replace two fields at the same time.",
			Fields: []replaceConfig{
				{
					Field:       "f",
					Pattern:     regexp.MustCompile(`a.*c`),
					Replacement: "cab",
				},
				{
					Field:       "g",
					Pattern:     regexp.MustCompile(`ef`),
					Replacement: "oor",
				},
			},
			Input: common.MapStr{
				"f": "abbbc",
				"g": "def",
			},
			Output: common.MapStr{
				"f": "cab",
				"g": "door",
			},
			error:         false,
			IgnoreMissing: false,
			FailOnError:   true,
		},
		{
			description: "test missing fields",
			Fields: []replaceConfig{
				{
					Field:       "f",
					Pattern:     regexp.MustCompile(`abc`),
					Replacement: "xyz",
				},
				{
					Field:       "g",
					Pattern:     regexp.MustCompile(`def`),
					Replacement: "",
				},
			},
			Input: common.MapStr{
				"m": "abc",
				"n": "def",
			},
			Output: common.MapStr{
				"m": "abc",
				"n": "def",
				"error": common.MapStr{
					"message": "Failed to replace fields in processor: could not fetch value for key: f, Error: key not found",
				},
			},
			error:         true,
			IgnoreMissing: false,
			FailOnError:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			f := &replaceString{
				config: replaceStringConfig{
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
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.True(t, reflect.DeepEqual(newEvent.Fields, test.Output))
		})
	}
}

func TestReplaceField(t *testing.T) {
	var tests = []struct {
		Field         string
		Pattern       *regexp.Regexp
		Replacement   string
		ignoreMissing bool
		failOnError   bool
		Input         common.MapStr
		Output        common.MapStr
		error         bool
		description   string
	}{
		{
			description: "replace part of field value with another string",
			Field:       "f",
			Pattern:     regexp.MustCompile(`a`),
			Replacement: "b",
			Input: common.MapStr{
				"f": "abc",
			},
			Output: common.MapStr{
				"f": "bbc",
			},
			error:         false,
			failOnError:   true,
			ignoreMissing: false,
		},
		{
			description: "Add hierarchy to event and replace",
			Field:       "f.b",
			Pattern:     regexp.MustCompile(`a`),
			Replacement: "b",
			Input: common.MapStr{
				"f": common.MapStr{
					"b": "abc",
				},
			},
			Output: common.MapStr{
				"f": common.MapStr{
					"b": "bbc",
				},
			},
			error:         false,
			ignoreMissing: false,
			failOnError:   true,
		},
		{
			description: "try replacing value of missing fields in event",
			Field:       "f",
			Pattern:     regexp.MustCompile(`abc`),
			Replacement: "xyz",
			Input: common.MapStr{
				"m": "abc",
				"n": "def",
			},
			Output: common.MapStr{
				"m": "abc",
				"n": "def",
			},
			error:         true,
			ignoreMissing: false,
			failOnError:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			f := &replaceString{
				config: replaceStringConfig{
					IgnoreMissing: test.ignoreMissing,
					FailOnError:   test.failOnError,
				},
			}

			err := f.replaceField(test.Field, test.Pattern, test.Replacement, &beat.Event{Fields: test.Input})
			if err != nil {
				assert.Equal(t, test.error, true)
			}

			assert.True(t, reflect.DeepEqual(test.Input, test.Output))
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		event := &beat.Event{
			Meta: common.MapStr{
				"f": "abc",
			},
		}

		expectedMeta := common.MapStr{
			"f": "bbc",
		}

		f := &replaceString{
			config: replaceStringConfig{
				Fields: []replaceConfig{
					{
						Field:       "@metadata.f",
						Pattern:     regexp.MustCompile(`a`),
						Replacement: "b",
					},
				},
			},
		}

		newEvent, err := f.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, expectedMeta, newEvent.Meta)
		assert.Equal(t, event.Fields, newEvent.Fields)
	})
}
