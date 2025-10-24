// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package encoding

import (
	"reflect"
	"testing"
	"time"
)

func TestEncodingFlagHas(t *testing.T) {
	tests := []struct {
		flag     EncodingFlag
		option   EncodingFlag
		expected bool
	}{
		{EncodingFlagUseNumbersZeroValues, EncodingFlagUseNumbersZeroValues, true},
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
		name     string
		input    any
		flags    EncodingFlag
		expected map[string]string
		err      bool
	}{
		{
			name:    "nil input",
			input:    nil,
			flags:    0,
			expected: nil,
			err:      true,
		},
		{
			name: "struct with osquery tag",
			input: &struct {
				Name string `osquery:"name"`
			}{Name: "test"},
			flags:    0,
			expected: map[string]string{"name": "test"},
			err:      false,
		},
		{
			name: "map input",
			input:    map[string]any{"key1": "value1", "key2": "value2", "key3": 1},
			flags:    0,
			expected: map[string]string{"key1": "value1", "key2": "value2", "key3": "1"},
			err:      false,
		},
		{
			name: "struct with hidden field",
			input: &struct {
				HiddenField int `osquery:"-"`
			}{HiddenField: 42},
			flags:    0,
			expected: map[string]string{},
			err:      false,
		},
		{
			name: "invalid type",
			input: &struct {
				InvalidType map[int]string
			}{InvalidType: map[int]string{1: "value"}},
			flags:    0,
			expected: map[string]string{"InvalidType": "map[1:value]"},
			err:      false,
		},
		{
			name: "zero value int",
			input: &struct {
				ZeroVal int
			}{ZeroVal: 0},
			flags:    0,
			expected: map[string]string{"ZeroVal": ""},
			err:      false,
		},
		{
			name: "zero value int with flag",
			input: &struct {
				ZeroVal int
			}{ZeroVal: 0},
			flags:    EncodingFlagUseNumbersZeroValues,
			expected: map[string]string{"ZeroVal": "0"},
			err:      false,
		},
		// Test bool type
		{
			name: "bool type",
			input: &struct {
				IsActive bool
			}{IsActive: true},
			flags:    0,
			expected: map[string]string{"IsActive": "1"},
			err:      false,
		},
		{
			name: "bool type false",
			input: &struct {
				IsActive bool
			}{IsActive: false},	
			flags:    0,
			expected: map[string]string{"IsActive": "0"},
			err:      false,
		},
		// Test uint type
		{
			name: "uint type",
			input: &struct {
				Count uint
			}{Count: 42},
			flags:    0,
			expected: map[string]string{"Count": "42"},
			err:      false,
		},
		{
			name: "zero value uint",
			input: &struct {
				Count uint
			}{Count: 0},
			flags:    0,
			expected: map[string]string{"Count": ""},
			err:      false,
		},
		{
			name: "zero value uint with flag",
			input: &struct {
				Count uint
			}{Count: 0},
			flags:    EncodingFlagUseNumbersZeroValues,
			expected: map[string]string{"Count": "0"},
			err:      false,
		},
		// Test float type
		{
			name: "float64 type",
			input: &struct {
				Price float64
			}{Price: 99.99},
			flags:    0,
			expected: map[string]string{"Price": "99.99"},
			err:      false,
		},
		{
			name: "float32 type",
			input: &struct {
				Price float32
			}{Price: 12.5},
			flags:    0,
			expected: map[string]string{"Price": "12.5"},
			err:      false,
		},
		// Test non-pointer struct
		{
			name: "non-pointer struct",
			input: struct {
				Name string
			}{Name: "test"},
			flags:    0,
			expected: map[string]string{"Name": "test"},
			err:      false,
		},
		// Test pointer maps
		{
			name: "pointer map",
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
			name: "pointer fields",
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
			name: "nil pointer fields",
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
		// Test time.Time type
		{
			name: "time.Time type",
			input: &struct {
				Time time.Time `osquery:"time"`
			}{Time: time.Unix(1719158400, 0)},
			flags:    0,
			expected: map[string]string{"time": "1719158400"},
			err:      false,
		},
		{
			name: "time.Time type with custom tag",
			input: &struct {
				Time time.Time `osquery:"my_time"`
			}{Time: time.Unix(1719158400, 0)},
			flags:    0,
			expected: map[string]string{"my_time": "1719158400"},
			err:      false,
		},
		{
			name: "time.Time type with zero value and flag",
			input: &struct {
				Time time.Time `osquery:"time"`
			}{Time: time.Unix(0, 0)},
			flags:    EncodingFlagUseNumbersZeroValues,
			expected: map[string]string{"time": "0"},
			err:      false,
		},
		{
			name: "time.Time type with zero value and no flag",
			input: &struct {
				Time time.Time `osquery:"time"`
			}{Time: time.Unix(0, 0)},
			flags:    0,
			expected: map[string]string{"time": ""},
			err:      false,
		},
	}

	for _, test := range tests {
		result, err := MarshalToMapWithFlags(test.input, test.flags)
		if (err != nil) != test.err {
			t.Errorf("%s: MarshalToMapWithFlags(%v, %v) error = %v; expected error = %v", test.name, test.input, test.flags, err, test.err)
			continue
		}
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("%s: MarshalToMapWithFlags(%v, %v) = %v; expected %v", test.name, test.input, test.flags, result, test.expected)
			continue
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
