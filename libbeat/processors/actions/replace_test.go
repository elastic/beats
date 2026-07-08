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

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestBadConfig(t *testing.T) {
	var cases = []struct {
		name        string
		cfg         replaceStringConfig
		shouldError bool
	}{
		{
			name:        "field-only",
			cfg:         replaceStringConfig{Fields: []replaceConfig{{Field: "message"}}},
			shouldError: true,
		},
		{
			name:        "no-regex",
			cfg:         replaceStringConfig{Fields: []replaceConfig{{Field: "message", Replacement: ptr("new_message")}}},
			shouldError: true,
		},
		{
			name:        "no-replacement",
			cfg:         replaceStringConfig{Fields: []replaceConfig{{Field: "message", Pattern: regexp.MustCompile(`message`)}}},
			shouldError: true,
		},
		{
			name: "valid-then-invalid",
			cfg: replaceStringConfig{Fields: []replaceConfig{
				{Field: "message", Pattern: regexp.MustCompile(`message`), Replacement: ptr("new_message")},
				{Field: "message", Pattern: regexp.MustCompile(`message`)},
			},
			},
			shouldError: true,
		},
		{
			name:        "no-error",
			cfg:         replaceStringConfig{Fields: []replaceConfig{{Field: "message", Replacement: ptr("new_message"), Pattern: regexp.MustCompile(`message`)}}},
			shouldError: false,
		},
		{
			name:        "no-error zero string",
			cfg:         replaceStringConfig{Fields: []replaceConfig{{Field: "message", Replacement: ptr(""), Pattern: regexp.MustCompile(`message`)}}},
			shouldError: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(testCase.cfg)
			require.NoError(t, err)
			unpacked := replaceStringConfig{}
			err = cfg.Unpack(&unpacked)
			if testCase.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

		})
	}

}

func TestReplaceRun(t *testing.T) {
	var tests = []struct {
		description   string
		Fields        []replaceConfig
		IgnoreMissing bool
		FailOnError   bool
		Input         mapstr.M
		Output        mapstr.M
		error         bool
	}{
		{
			description: "simple field replacing",
			Fields: []replaceConfig{
				{
					Field:       "f",
					Pattern:     regexp.MustCompile(`a`),
					Replacement: ptr("b"),
				},
			},
			Input: mapstr.M{
				"f": "abc",
			},
			Output: mapstr.M{
				"f": "bbc",
			},
			error:         false,
			IgnoreMissing: false,
			FailOnError:   true,
		},
		{
			description: "replace with zero",
			Fields: []replaceConfig{
				{
					Field:       "f",
					Pattern:     regexp.MustCompile(`a`),
					Replacement: ptr(""),
				},
			},
			Input: mapstr.M{
				"f": "abc",
			},
			Output: mapstr.M{
				"f": "bc",
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
					Replacement: ptr("b"),
				},
			},
			Input: mapstr.M{
				"f": mapstr.M{
					"b": "abc",
				},
			},
			Output: mapstr.M{
				"f": mapstr.M{
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
					Replacement: ptr("cab"),
				},
				{
					Field:       "g",
					Pattern:     regexp.MustCompile(`ef`),
					Replacement: ptr("oor"),
				},
			},
			Input: mapstr.M{
				"f": "abbbc",
				"g": "def",
			},
			Output: mapstr.M{
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
					Replacement: ptr("xyz"),
				},
				{
					Field:       "g",
					Pattern:     regexp.MustCompile(`def`),
					Replacement: nil,
				},
			},
			Input: mapstr.M{
				"m": "abc",
				"n": "def",
			},
			Output: mapstr.M{
				"m": "abc",
				"n": "def",
				"error": mapstr.M{
					"message": "Failed to replace fields in processor: could not fetch value for key: f, Error: key not found",
				},
			},
			error:         true,
			IgnoreMissing: false,
			FailOnError:   true,
		},
		{
			description: "non-string value: nil",
			Fields: []replaceConfig{
				{
					Field:       "f",
					Pattern:     regexp.MustCompile(`.*`),
					Replacement: ptr("b"),
				},
			},
			Input: mapstr.M{
				"f": nil,
			},
			Output: mapstr.M{
				"f": nil,
				"error": mapstr.M{
					"message": "Failed to replace fields in processor: key 'f' expected type string, but got <nil> with value '<nil>'",
				},
			},
			error:         true,
			IgnoreMissing: false,
			FailOnError:   true,
		},
		{
			description: "non-string value: float64",
			Fields: []replaceConfig{
				{
					Field:       "f",
					Pattern:     regexp.MustCompile(`.*`),
					Replacement: ptr("b"),
				},
			},
			Input: mapstr.M{
				"f": 123.45,
			},
			Output: mapstr.M{
				"f": 123.45,
				"error": mapstr.M{
					"message": "Failed to replace fields in processor: key 'f' expected type string, but got float64 with value '123.45'",
				},
			},
			error:         true,
			IgnoreMissing: false,
			FailOnError:   true,
		},
		{
			description: "non-string value: integer",
			Fields: []replaceConfig{
				{
					Field:       "f",
					Pattern:     regexp.MustCompile(`.*`),
					Replacement: ptr("b"),
				},
			},
			Input: mapstr.M{
				"f": 123,
			},
			Output: mapstr.M{
				"f": 123,
				"error": mapstr.M{
					"message": "Failed to replace fields in processor: key 'f' expected type string, but got int with value '123'",
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
				log: logptest.NewTestingLogger(t, "replace"),
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

			assert.Equal(t, newEvent.Fields, test.Output)
		})
	}
}

func ptr[T any](v T) *T { return &v }

// TestReplaceFailOnErrorSafety verifies that when FailOnError=true and the
// field is missing, the event fields are unchanged.
func TestReplaceFailOnErrorSafety(t *testing.T) {
	tests := []struct {
		name  string
		input mapstr.M
	}{
		{
			name:  "missing field",
			input: mapstr.M{"other": "value"},
		},
		{
			name:  "non-string field value",
			input: mapstr.M{"f": 42},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &replaceString{
				log: logptest.NewTestingLogger(t, "replace"),
				config: replaceStringConfig{
					Fields: []replaceConfig{
						{
							Field:       "f",
							Pattern:     regexp.MustCompile(`abc`),
							Replacement: ptr("xyz"),
						},
					},
					FailOnError: true,
				},
			}

			input := tc.input.Clone()
			event := &beat.Event{Fields: input}
			original := input.Clone()

			result, err := f.Run(event)
			require.Error(t, err)
			assert.Same(t, event, result)

			result.Fields.Delete("error")
			assert.Equal(t, original, result.Fields,
				"event fields must be unchanged after error (clone skip safety)")
		})
	}
}

// TestReplaceMultiFieldBackup verifies that when multiple fields are configured
// and a later field fails, the changes from earlier fields are rolled back via backup.
func TestReplaceMultiFieldBackup(t *testing.T) {
	f := &replaceString{
		log: logptest.NewTestingLogger(t, "replace"),
		config: replaceStringConfig{
			Fields: []replaceConfig{
				{
					Field:       "f1",
					Pattern:     regexp.MustCompile(`a`),
					Replacement: ptr("X"),
				},
				{
					Field:       "f2",
					Pattern:     regexp.MustCompile(`a`),
					Replacement: ptr("X"),
				},
			},
			FailOnError: true,
		},
	}

	// f1 is a string (replaces "abc" → "Xbc" on first iteration),
	// f2 is an int (will fail the string type check on second iteration).
	input := mapstr.M{"f1": "abc", "f2": 42}
	event := &beat.Event{Fields: input.Clone()}
	original := input.Clone()

	result, err := f.Run(event)
	require.Error(t, err)

	// Multi-field: backup was taken and restored; result is the backup pointer.
	// f1 change must be undone.
	result.Fields.Delete("error")
	assert.Equal(t, original, result.Fields,
		"first field's replacement must be rolled back when second field fails")
}

func TestReplaceField(t *testing.T) {
	var tests = []struct {
		Field         string
		Pattern       *regexp.Regexp
		Replacement   string
		ignoreMissing bool
		failOnError   bool
		Input         mapstr.M
		Output        mapstr.M
		error         bool
		description   string
	}{
		{
			description: "replace part of field value with another string",
			Field:       "f",
			Pattern:     regexp.MustCompile(`a`),
			Replacement: "b",
			Input: mapstr.M{
				"f": "abc",
			},
			Output: mapstr.M{
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
			Input: mapstr.M{
				"f": mapstr.M{
					"b": "abc",
				},
			},
			Output: mapstr.M{
				"f": mapstr.M{
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
			Input: mapstr.M{
				"m": "abc",
				"n": "def",
			},
			Output: mapstr.M{
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
				assert.True(t, test.error)
			}

			assert.True(t, reflect.DeepEqual(test.Input, test.Output))
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		event := &beat.Event{
			Meta: mapstr.M{
				"f": "abc",
			},
		}

		expectedMeta := mapstr.M{
			"f": "bbc",
		}

		f := &replaceString{
			config: replaceStringConfig{
				Fields: []replaceConfig{
					{
						Field:       "@metadata.f",
						Pattern:     regexp.MustCompile(`a`),
						Replacement: ptr("b"),
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
