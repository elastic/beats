// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package encoding

import (
	"reflect"
	"testing"
)

func TestEncodingFlagHas(t *testing.T) {
	tests := []struct {
		flag     EncodingFlag
		option   EncodingFlag
		expected bool
	}{
		{EncodingFlagParseUnexported, EncodingFlagParseUnexported, true},
		{EncodingFlagParseUnexported, EncodingFlagUseNumbersZeroValues, false},
		{EncodingFlagUseNumbersZeroValues, EncodingFlagUseNumbersZeroValues, true},
		{EncodingFlagParseUnexported | EncodingFlagUseNumbersZeroValues, EncodingFlagParseUnexported, true},
		{EncodingFlagParseUnexported | EncodingFlagUseNumbersZeroValues, EncodingFlagUseNumbersZeroValues, true},
	}

	for _, test := range tests {
		result := test.flag.has(test.option)
		if result != test.expected {
			t.Errorf("has(%v) = %v; expected %v", test.option, result, test.expected)
		}
	}
}

func TestMarshalToMapWithFlags(t *testing.T) {
	tests := []struct {
		input    any
		flags    EncodingFlag
		expected map[string]string
		err      bool
	}{
		{
			input:    nil,
			flags:    0,
			expected: nil,
			err:      true,
		},
		{
			input: &struct {
				Name string `osquery:"name"`
			}{Name: "test"},
			flags:    0,
			expected: map[string]string{"name": "test"},
			err:      false,
		},
		{
			input:    map[string]any{"key1": "value1", "key2": "value2", "key3": 1},
			flags:    0,
			expected: map[string]string{"key1": "value1", "key2": "value2", "key3": "1"},
			err:      false,
		},
		{
			input: &struct {
				age int `osquery:"Age"`
			}{age: 30},
			flags:    EncodingFlagParseUnexported,
			expected: map[string]string{"Age": "30"},
			err:      false,
		},
		{
			input: &struct {
				HiddenField int `osquery:"-"`
			}{HiddenField: 42},
			flags:    0,
			expected: map[string]string{},
			err:      false,
		},
		{
			input: &struct {
				InvalidType map[int]string
			}{InvalidType: map[int]string{1: "value"}},
			flags:    0,
			expected: map[string]string{"InvalidType": "map[1:value]"},
			err:      false,
		},
		{
			input: &struct {
				ZeroVal int
			}{ZeroVal: 0},
			flags:    0,
			expected: map[string]string{"ZeroVal": ""},
			err:      false,
		},
		{
			input: &struct {
				ZeroVal int
			}{ZeroVal: 0},
			flags:    EncodingFlagUseNumbersZeroValues,
			expected: map[string]string{"ZeroVal": "0"},
			err:      false,
		},
		// Test bool type
		{
			input: &struct {
				IsActive bool
			}{IsActive: true},
			flags:    0,
			expected: map[string]string{"IsActive": "1"},
			err:      false,
		},
		{
			input: &struct {
				IsActive bool
			}{IsActive: false},
			flags:    0,
			expected: map[string]string{"IsActive": "0"},
			err:      false,
		},
		// Test uint type
		{
			input: &struct {
				Count uint
			}{Count: 42},
			flags:    0,
			expected: map[string]string{"Count": "42"},
			err:      false,
		},
		{
			input: &struct {
				Count uint
			}{Count: 0},
			flags:    0,
			expected: map[string]string{"Count": ""},
			err:      false,
		},
		{
			input: &struct {
				Count uint
			}{Count: 0},
			flags:    EncodingFlagUseNumbersZeroValues,
			expected: map[string]string{"Count": "0"},
			err:      false,
		},
		// Test float type
		{
			input: &struct {
				Price float64
			}{Price: 99.99},
			flags:    0,
			expected: map[string]string{"Price": "99.99"},
			err:      false,
		},
		{
			input: &struct {
				Price float32
			}{Price: 12.5},
			flags:    0,
			expected: map[string]string{"Price": "12.5"},
			err:      false,
		},
		// Test non-pointer struct
		{
			input: struct {
				Name string
			}{Name: "test"},
			flags:    0,
			expected: map[string]string{"Name": "test"},
			err:      false,
		},
		// Test pointer maps
		{
			input: &map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			flags:    0,
			expected: map[string]string{"key1": "value1", "key2": "value2"},
			err:      false,
		},
		// Test pointer fields
		{
			input: &struct {
				StrPtr *string
				IntPtr *int
			}{
				StrPtr: stringPtr("hello"),
				IntPtr: intPtr(123),
			},
			flags:    0,
			expected: map[string]string{"StrPtr": "hello", "IntPtr": "123"},
			err:      false,
		},
		// Test nil pointer fields
		{
			input: &struct {
				StrPtr *string
				IntPtr *int
			}{
				StrPtr: nil,
				IntPtr: nil,
			},
			flags:    0,
			expected: map[string]string{"StrPtr": "", "IntPtr": ""},
			err:      false,
		},
	}

	for _, test := range tests {
		result, err := MarshalToMapWithFlags(test.input, test.flags)
		if (err != nil) != test.err {
			t.Errorf("MarshalToMapWithFlags(%v, %v) error = %v; expected error = %v", test.input, test.flags, err, test.err)
			continue
		}
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("MarshalToMapWithFlags(%v, %v) = %v; expected %v", test.input, test.flags, result, test.expected)
		}
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
