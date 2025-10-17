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
		{EncodingFlagParseUnexported, EncodingFlagUseNumbers0Values, false},
		{EncodingFlagUseNumbers0Values, EncodingFlagUseNumbers0Values, true},
		{EncodingFlagParseUnexported | EncodingFlagUseNumbers0Values, EncodingFlagParseUnexported, true},
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
			flags:    EncodingFlagUseNumbers0Values,
			expected: map[string]string{"ZeroVal": "0"},
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
