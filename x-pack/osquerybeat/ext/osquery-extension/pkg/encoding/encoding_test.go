// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package encoding

import (
	"reflect"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
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

func TestGenerateColumnDefinitions(t *testing.T) {
	tests := []struct {
		name          string
		input         any
		expectedCols  []table.ColumnDefinition
		expectedError bool
	}{
		{
			name: "basic struct with all supported types",
			input: struct {
				Name      string  `osquery:"name"`
				Count     int64   `osquery:"count"`
				Active    bool    `osquery:"active"`
				Score     float64 `osquery:"score"`
				SmallNum  int     `osquery:"small_num"`
				BigNum    uint64  `osquery:"big_num"`
				Precision float32 `osquery:"precision"`
			}{},
			expectedCols: []table.ColumnDefinition{
				table.TextColumn("name"),
				table.BigIntColumn("count"),
				table.IntegerColumn("active"),
				table.DoubleColumn("score"),
				table.IntegerColumn("small_num"),
				table.BigIntColumn("big_num"),
				table.DoubleColumn("precision"),
			},
			expectedError: false,
		},
		{
			name: "struct with skipped fields",
			input: struct {
				Included string `osquery:"included"`
				Skipped  string `osquery:"-"`
				NoTag    string
			}{},
			expectedCols: []table.ColumnDefinition{
				table.TextColumn("included"),
				table.TextColumn("NoTag"), // Fields without osquery tag use field name
			},
			expectedError: false,
		},
		{
			name: "struct with pointer fields",
			input: struct {
				Name  *string `osquery:"name"`
				Count *int64  `osquery:"count"`
			}{},
			expectedCols: []table.ColumnDefinition{
				table.TextColumn("name"),
				table.BigIntColumn("count"),
			},
			expectedError: false,
		},
		{
			name:          "nil input",
			input:         nil,
			expectedCols:  nil,
			expectedError: true,
		},
		{
			name:          "non-struct input",
			input:         "string",
			expectedCols:  nil,
			expectedError: true,
		},
		{
			name: "pointer to struct",
			input: &struct {
				Field string `osquery:"field"`
			}{},
			expectedCols: []table.ColumnDefinition{
				table.TextColumn("field"),
			},
			expectedError: false,
		},
		{
			name: "all integer types",
			input: struct {
				I   int    `osquery:"i"`
				I8  int8   `osquery:"i8"`
				I16 int16  `osquery:"i16"`
				I32 int32  `osquery:"i32"`
				I64 int64  `osquery:"i64"`
				U   uint   `osquery:"u"`
				U8  uint8  `osquery:"u8"`
				U16 uint16 `osquery:"u16"`
				U32 uint32 `osquery:"u32"`
				U64 uint64 `osquery:"u64"`
			}{},
			expectedCols: []table.ColumnDefinition{
				table.IntegerColumn("i"),
				table.IntegerColumn("i8"),
				table.IntegerColumn("i16"),
				table.IntegerColumn("i32"),
				table.BigIntColumn("i64"),
				table.IntegerColumn("u"),
				table.IntegerColumn("u8"),
				table.IntegerColumn("u16"),
				table.IntegerColumn("u32"),
				table.BigIntColumn("u64"),
			},
			expectedError: false,
		},
		{
			name: "fields without osquery tag use field name",
			input: struct {
				CustomName  string `osquery:"custom_name"`
				DefaultName string
			}{},
			expectedCols: []table.ColumnDefinition{
				table.TextColumn("custom_name"),
				table.TextColumn("DefaultName"), // No tag, uses field name
			},
			expectedError: false,
		},
		{
			name: "mixed exported and unexported fields",
			input: struct {
				Public  string `osquery:"public"`
				private string `osquery:"private"` // will be skipped
				Another string `osquery:"another"`
			}{},
			expectedCols: []table.ColumnDefinition{
				table.TextColumn("public"),
				table.TextColumn("another"),
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, err := GenerateColumnDefinitions(tt.input)

			if (err != nil) != tt.expectedError {
				t.Errorf("GenerateColumnDefinitions() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if !tt.expectedError {
				if len(cols) != len(tt.expectedCols) {
					t.Errorf("GenerateColumnDefinitions() returned %d columns, expected %d", len(cols), len(tt.expectedCols))
					return
				}

				for i := range cols {
					if cols[i].Name != tt.expectedCols[i].Name {
						t.Errorf("Column %d: name = %s, expected %s", i, cols[i].Name, tt.expectedCols[i].Name)
					}
					if cols[i].Type != tt.expectedCols[i].Type {
						t.Errorf("Column %d (%s): type = %s, expected %s", i, cols[i].Name, cols[i].Type, tt.expectedCols[i].Type)
					}
				}
			}
		})
	}
}
