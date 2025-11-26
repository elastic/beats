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

//go:build !requirefips

package translate_ldap_attribute

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGUIDToBytes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []byte
		expectError bool
	}{
		{
			name:  "GUID with curly braces and hyphens",
			input: "{7fb125ee-ceaf-48ff-8385-32c516ab10ed}",
			// Expected byte order after Microsoft GUID conversion:
			// Original hex: 7fb125ee-ceaf-48ff-8385-32c516ab10ed
			// After swap: ee25b17f-afce-ff48-8385-32c516ab10ed
			expected:    []byte{0xee, 0x25, 0xb1, 0x7f, 0xaf, 0xce, 0xff, 0x48, 0x83, 0x85, 0x32, 0xc5, 0x16, 0xab, 0x10, 0xed},
			expectError: false,
		},
		{
			name:        "GUID with hyphens",
			input:       "7fb125ee-ceaf-48ff-8385-32c516ab10ed",
			expected:    []byte{0xee, 0x25, 0xb1, 0x7f, 0xaf, 0xce, 0xff, 0x48, 0x83, 0x85, 0x32, 0xc5, 0x16, 0xab, 0x10, 0xed},
			expectError: false,
		},
		{
			name:        "GUID without hyphens",
			input:       "7fb125eeceaf48ff838532c516ab10ed",
			expected:    []byte{0xee, 0x25, 0xb1, 0x7f, 0xaf, 0xce, 0xff, 0x48, 0x83, 0x85, 0x32, 0xc5, 0x16, 0xab, 0x10, 0xed},
			expectError: false,
		},
		{
			name:        "Another valid GUID",
			input:       "{a1b2c3d4-e5f6-0718-9293-a4b5c6d7e8f9}",
			expected:    []byte{0xd4, 0xc3, 0xb2, 0xa1, 0xf6, 0xe5, 0x18, 0x07, 0x92, 0x93, 0xa4, 0xb5, 0xc6, 0xd7, 0xe8, 0xf9},
			expectError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Invalid length",
			input:       "7fb125ee-ceaf-48ff-8385",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Invalid hex characters",
			input:       "7fb125ee-ceaf-48ff-8385-32c516ab10xz",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Too long",
			input:       "7fb125ee-ceaf-48ff-8385-32c516ab10ed-extra",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := guidToBytes(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result, "Expected: %s, Got: %s",
					hex.EncodeToString(tt.expected),
					hex.EncodeToString(result))
			}
		})
	}
}

func TestEscapeBinaryForLDAP(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Simple binary data",
			input:    []byte{0x7f, 0xb1, 0x25, 0xee},
			expected: "\\7f\\b1\\25\\ee",
		},
		{
			name:     "GUID binary",
			input:    []byte{0xee, 0x25, 0xb1, 0x7f, 0xaf, 0xce, 0xff, 0x48, 0x83, 0x85, 0x32, 0xc5, 0x16, 0xab, 0x10, 0xed},
			expected: "\\ee\\25\\b1\\7f\\af\\ce\\ff\\48\\83\\85\\32\\c5\\16\\ab\\10\\ed",
		},
		{
			name:     "Empty byte array",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "Single byte",
			input:    []byte{0x00},
			expected: "\\00",
		},
		{
			name:     "High value bytes",
			input:    []byte{0xff, 0xfe, 0xfd},
			expected: "\\ff\\fe\\fd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeBinaryForLDAP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
