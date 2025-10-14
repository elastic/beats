// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package utilities

import (
	"testing"
	"www.velocidex.com/golang/regparser"
)

func TestMakeTrimmedString(t *testing.T) {
	tests := []struct {
		name     string
		input    regparser.ValueData
		expected string
	}{
		{
			name:     "Test case 1",
			input:    regparser.ValueData{String: "Test string 1\x00\x00", Type: regparser.REG_SZ},
			expected: "Test string 1",
		},
		{
			name:     "Test case 2",
			input:    regparser.ValueData{String: "Test string 2\x00\x00", Type: regparser.REG_SZ},
			expected: "Test string 2",
		},
		{
			name:     "Test case 3",
			input:    regparser.ValueData{Uint64: 12345, Type: regparser.REG_DWORD},
			expected: "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function being tested with the test input
			output := MakeTrimmedString(&tt.input)

			// Check the output against the expected result
			if output != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, output)
			}
		})
	}
}
