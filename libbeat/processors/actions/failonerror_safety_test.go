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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestDecodeBase64FailOnErrorSafety verifies that when FailOnError=true and
// decoding fails, the event fields are unchanged (proving clone skip is safe).
func TestDecodeBase64FailOnErrorSafety(t *testing.T) {
	tests := []struct {
		name  string
		input mapstr.M
	}{
		{
			name:  "invalid base64 data",
			input: mapstr.M{"field1": "not valid base64!!!"},
		},
		{
			name:  "missing source field",
			input: mapstr.M{"other": "value"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &decodeBase64Field{
				log: logptest.NewTestingLogger(t, processorName),
				config: base64Config{
					Field:       fromTo{From: "field1", To: "field2"},
					FailOnError: true,
				},
			}

			input := tc.input.Clone()
			event := &beat.Event{Fields: input}
			original := input.Clone()

			result, err := f.Run(event)
			require.Error(t, err)

			result.Fields.Delete("error")
			assert.Equal(t, original, result.Fields,
				"event fields must be unchanged after error (clone skip safety)")
		})
	}
}

// TestDecompressGzipFailOnErrorSafety verifies that when FailOnError=true and
// decompression fails, the event fields are unchanged.
func TestDecompressGzipFailOnErrorSafety(t *testing.T) {
	tests := []struct {
		name  string
		input mapstr.M
	}{
		{
			name:  "invalid gzip data",
			input: mapstr.M{"field1": "not gzip data"},
		},
		{
			name:  "missing source field",
			input: mapstr.M{"other": "value"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &decompressGzipField{
				log: logptest.NewTestingLogger(t, "decompress_gzip_field"),
				config: decompressGzipFieldConfig{
					Field:       fromTo{From: "field1", To: "field2"},
					FailOnError: true,
				},
			}

			input := tc.input.Clone()
			event := &beat.Event{Fields: input}
			original := input.Clone()

			result, err := f.Run(event)
			require.Error(t, err)

			result.Fields.Delete("error")
			assert.Equal(t, original, result.Fields,
				"event fields must be unchanged after error (clone skip safety)")
		})
	}
}

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

			result.Fields.Delete("error")
			assert.Equal(t, original, result.Fields,
				"event fields must be unchanged after error (clone skip safety)")
		})
	}
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

	result.Fields.Delete("error")
	assert.Equal(t, original, result.Fields,
		"event fields must be unchanged after error (clone skip safety)")
}
