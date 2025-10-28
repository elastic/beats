// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package encoding

import (
	"reflect"
	"testing"
	"time"

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

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
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
			name:     "nil input",
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
			name:     "map input",
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
			}{Time: time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)},
			flags:    0,
			expected: map[string]string{"time": "2023-06-15T14:30:00Z"},
			err:      false,
		},
		{
			name: "time.Time",
			input: &struct {
				Time time.Time `osquery:"time" format:"unix"`
			}{Time: time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)},
			flags:    0,
			expected: map[string]string{"time": "1686839400"},
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

func tagPtr(tag string) *reflect.StructTag {
	st := reflect.StructTag(tag)
	return &st
}

func Test_formatTimeWithTagFormat(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		fieldValue reflect.Value
		flag       EncodingFlag
		tag        *reflect.StructTag
		want       string
		wantErr    bool
	}{
		{
			name:       "RFC3339 format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc3339"`),
			want:       "2023-06-15T14:30:00Z",
			wantErr:    false,
		},
		{
			name:       "RFC3339Nano format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 123456789, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc3339nano"`),
			want:       "2023-06-15T14:30:00.123456789Z",
			wantErr:    false,
		},
		{
			name:       "Unix timestamp format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"unix"`),
			want:       "1686839400",
			wantErr:    false,
		},
		{
			name:       "RFC822 format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc822"`),
			want:       "15 Jun 23 14:30 UTC",
			wantErr:    false,
		},
		{
			name:       "RFC822Z format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc822z"`),
			want:       "15 Jun 23 14:30 +0000",
			wantErr:    false,
		},
		{
			name:       "RFC850 format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc850"`),
			want:       "Thursday, 15-Jun-23 14:30:00 UTC",
			wantErr:    false,
		},
		{
			name:       "RFC1123 format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc1123"`),
			want:       "Thu, 15 Jun 2023 14:30:00 UTC",
			wantErr:    false,
		},
		{
			name:       "RFC1123Z format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc1123z"`),
			want:       "Thu, 15 Jun 2023 14:30:00 +0000",
			wantErr:    false,
		},
		{
			name:       "Kitchen format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"kitchen"`),
			want:       "Jun 15 14:30:00",
			wantErr:    false,
		},
		{
			name:       "StampMilli format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 123000000, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"stampmilli"`),
			want:       "Jun 15 14:30:00.123",
			wantErr:    false,
		},
		{
			name:       "StampMicro format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 123456000, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"stampmicro"`),
			want:       "Jun 15 14:30:00.123456",
			wantErr:    false,
		},
		{
			name:       "StampNano format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 123456789, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"stampnano"`),
			want:       "Jun 15 14:30:00.123456789",
			wantErr:    false,
		},
		{
			name:       "Invalid format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"invalid"`),
			want:       "",
			wantErr:    true,
		},
		{
			name:       "With timezone",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc3339" tz:"America/New_York"`),
			want:       "2023-06-15T10:30:00-04:00",
			wantErr:    false,
		},
		{
			name:       "Invalid timezone",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc3339" tz:"Invalid/Timezone"`),
			want:       "",
			wantErr:    true,
		},
		{
			name:       "Zero time with default flags",
			fieldValue: reflect.ValueOf(time.Time{}),
			flag:       0,
			tag:        tagPtr(`format:"rfc3339"`),
			want:       "",
			wantErr:    false,
		},
		{
			name:       "Zero time with UseNumbersZeroValues flag",
			fieldValue: reflect.ValueOf(time.Time{}),
			flag:       EncodingFlagUseNumbersZeroValues,
			tag:        tagPtr(`format:"rfc3339"`),
			want:       "0001-01-01T00:00:00Z",
			wantErr:    false,
		},
		{
			name:       "With timezone Asia/Tokyo",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc3339" tz:"Asia/Tokyo"`),
			want:       "2023-06-15T23:30:00+09:00",
			wantErr:    false,
		},
		{
			name:       "With timezone Europe/London",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc3339" tz:"Europe/London"`),
			want:       "2023-06-15T15:30:00+01:00",
			wantErr:    false,
		},
		{
			name:       "With timezone Australia/Sydney",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"rfc3339" tz:"Australia/Sydney"`),
			want:       "2023-06-16T00:30:00+10:00",
			wantErr:    false,
		},
		{
			name:       "Unix milliseconds format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"unixmilli"`),
			want:       "1686839400000",
			wantErr:    false,
		},
		{
			name:       "Unix nanoseconds format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"unixnano"`),
			want:       "1686839400000000000",
			wantErr:    false,
		},
		{
			name:       "Unix microseconds format",
			fieldValue: reflect.ValueOf(time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)),
			flag:       0,
			tag:        tagPtr(`format:"unixmicro"`),
			want:       "1686839400000000",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := formatTimeWithTagFormat(tt.fieldValue, tt.flag, tt.tag)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Name: %s, formatTimeWithTagFormat() failed: %v", tt.name, gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatalf("Name: %s, formatTimeWithTagFormat() succeeded unexpectedly", tt.name)
			}
			// TODO: update the condition below to compare got with tt.want.
			if got != tt.want {
				t.Errorf("Name: %s, formatTimeWithTagFormat() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
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
