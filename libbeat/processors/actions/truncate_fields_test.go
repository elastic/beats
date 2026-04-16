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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestTruncateFields(t *testing.T) {
	log := logptest.NewTestingLogger(t, "truncate_fields_test")
	var tests = map[string]struct {
		MaxBytes     int
		MaxChars     int
		Input        mapstr.M
		Output       mapstr.M
		ShouldError  bool
		TruncateFunc truncater
	}{
		"truncate bytes of too long string line": {
			MaxBytes: 3,
			Input: mapstr.M{
				"message": "too long line",
			},
			Output: mapstr.M{
				"message": "too",
				"log": mapstr.M{
					"flags": []string{"truncated"},
				},
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"truncate bytes of too long byte line": {
			MaxBytes: 3,
			Input: mapstr.M{
				"message": []byte("too long line"),
			},
			Output: mapstr.M{
				"message": []byte("too"),
				"log": mapstr.M{
					"flags": []string{"truncated"},
				},
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"do not truncate short string line": {
			MaxBytes: 15,
			Input: mapstr.M{
				"message": "shorter line",
			},
			Output: mapstr.M{
				"message": "shorter line",
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"do not truncate short byte line": {
			MaxBytes: 15,
			Input: mapstr.M{
				"message": []byte("shorter line"),
			},
			Output: mapstr.M{
				"message": []byte("shorter line"),
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"try to truncate integer and get error": {
			MaxBytes: 5,
			Input: mapstr.M{
				"message": 42,
			},
			Output: mapstr.M{
				"message": 42,
			},
			ShouldError:  true,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"do not truncate characters of short byte line": {
			MaxChars: 6,
			Input: mapstr.M{
				"message": []byte("ez jó"), // this is good (hungarian)
			},
			Output: mapstr.M{
				"message": []byte("ez jó"), // this is good (hungarian)
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateCharacters,
		},
		"do not truncate bytes of short byte line with multibyte runes": {
			MaxBytes: 6,
			Input: mapstr.M{
				"message": []byte("ez jó"), // this is good (hungarian)
			},
			Output: mapstr.M{
				"message": []byte("ez jó"), // this is good (hungarian)
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"truncate characters of too long byte line": {
			MaxChars: 10,
			Input: mapstr.M{
				"message": []byte("ez egy túl hosszú sor"), // this is a too long line (hungarian)
			},
			Output: mapstr.M{
				"message": []byte("ez egy túl"), // this is a too (hungarian)
				"log": mapstr.M{
					"flags": []string{"truncated"},
				},
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateCharacters,
		},
		"truncate bytes of too long byte line with multibyte runes": {
			MaxBytes: 10,
			Input: mapstr.M{
				"message": []byte("ez egy túl hosszú sor"), // this is a too long line (hungarian)
			},
			Output: mapstr.M{
				"message": []byte("ez egy tú"), // this is a "to" (hungarian)
				"log": mapstr.M{
					"flags": []string{"truncated"},
				},
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := truncateFields{
				config: truncateFieldsConfig{
					Fields:      []string{"message"},
					MaxBytes:    test.MaxBytes,
					MaxChars:    test.MaxChars,
					FailOnError: true,
				},
				truncate: test.TruncateFunc,
				logger:   log,
			}

			event := &beat.Event{
				Fields: test.Input,
			}

			newEvent, err := p.Run(event)
			if test.ShouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, test.Output, newEvent.Fields)
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		p := truncateFields{
			config: truncateFieldsConfig{
				Fields:      []string{"@metadata.message"},
				MaxBytes:    3,
				FailOnError: true,
			},
			truncate: (*truncateFields).truncateBytes,
			logger:   log,
		}

		event := &beat.Event{
			Meta: mapstr.M{
				"message": "too long line",
			},
			Fields: mapstr.M{},
		}

		expFields := mapstr.M{
			"log": mapstr.M{
				"flags": []string{"truncated"},
			},
		}

		expMeta := mapstr.M{
			"message": "too",
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)

		assert.Equal(t, expFields, newEvent.Fields)
		assert.Equal(t, expMeta, newEvent.Meta)
	})
}

// TestTruncateFieldsFailOnErrorSafety verifies that when FailOnError=true and
// the field has a non-truncatable type, the event fields are unchanged.
func TestTruncateFieldsFailOnErrorSafety(t *testing.T) {
	f := &truncateFields{
		config: truncateFieldsConfig{
			Fields:      []string{"message"},
			MaxBytes:    5,
			FailOnError: true,
		},
		truncate: (*truncateFields).truncateBytes,
		logger:   logptest.NewTestingLogger(t, "truncate_fields"),
	}

	input := mapstr.M{"message": 42}
	event := &beat.Event{Fields: input.Clone()}
	original := input.Clone()

	result, err := f.Run(event)
	require.Error(t, err)
	assert.Same(t, event, result)

	assert.Equal(t, original, result.Fields,
		"event fields must be unchanged after error (clone skip safety)")
}

// TestTruncateFieldsMultiFieldBackup verifies that when multiple fields are
// configured and truncation of a later field fails, earlier truncations are
// rolled back via backup.
func TestTruncateFieldsMultiFieldBackup(t *testing.T) {
	f := &truncateFields{
		config: truncateFieldsConfig{
			Fields:      []string{"f1", "f2"},
			MaxBytes:    3,
			FailOnError: true,
		},
		truncate: (*truncateFields).truncateBytes,
		logger:   logptest.NewTestingLogger(t, "truncate_fields"),
	}

	// f1 is a long string (will be truncated on first iteration),
	// f2 is an int (will fail the type check on second iteration).
	input := mapstr.M{"f1": "hello", "f2": 42}
	event := &beat.Event{Fields: input.Clone()}
	original := input.Clone()

	result, err := f.Run(event)
	require.Error(t, err)

	// Multi-field: backup was taken and restored; f1 truncation must be undone.
	assert.Equal(t, original, result.Fields,
		"first field's truncation must be rolled back when second field fails")
}
