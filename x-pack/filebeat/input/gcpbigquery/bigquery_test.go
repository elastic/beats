package gcpbigquery

import (
	"math/big"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorState_Set_StringFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "valid string value",
			value:    "test_string",
			expected: `"test_string"`,
			wantErr:  false,
		},
		{
			name:     "empty string value",
			value:    "",
			expected: `""`,
			wantErr:  false,
		},
		{
			name:     "string with quotes",
			value:    `hello \"world\"`,
			expected: `"hello \"world\""`,
			wantErr:  false,
		},
		{
			name:    "invalid type - int64",
			value:   int64(123),
			wantErr: true,
		},
		{
			name:    "invalid type - nil",
			value:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.StringFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_IntegerFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "positive integer",
			value:    int64(123),
			expected: "123",
			wantErr:  false,
		},
		{
			name:     "negative integer",
			value:    int64(-456),
			expected: "-456",
			wantErr:  false,
		},
		{
			name:     "zero",
			value:    int64(0),
			expected: "0",
			wantErr:  false,
		},
		{
			name:     "max int64",
			value:    int64(9223372036854775807),
			expected: "9223372036854775807",
			wantErr:  false,
		},
		{
			name:    "invalid type - string",
			value:   "123",
			wantErr: true,
		},
		{
			name:    "invalid type - float64",
			value:   float64(123.45),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.IntegerFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_FloatFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "positive float",
			value:    float64(123.456),
			expected: "123.456000",
			wantErr:  false,
		},
		{
			name:     "negative float",
			value:    float64(-789.123),
			expected: "-789.123000",
			wantErr:  false,
		},
		{
			name:     "zero float",
			value:    float64(0.0),
			expected: "0.000000",
			wantErr:  false,
		},
		{
			name:     "scientific notation",
			value:    float64(1.23e-4),
			expected: "0.000123",
			wantErr:  false,
		},
		{
			name:    "invalid type - string",
			value:   "123.456",
			wantErr: true,
		},
		{
			name:    "invalid type - int64",
			value:   int64(123),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.FloatFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_BytesFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "valid bytes",
			value:    []byte("hello"),
			expected: `B"hello"`,
			wantErr:  false,
		},
		{
			name:     "empty bytes",
			value:    []byte(""),
			expected: `B""`,
			wantErr:  false,
		},
		{
			name:     "binary data",
			value:    []byte{0x00, 0x01, 0x02, 0xFF},
			expected: "B\"\x00\x01\x02\xff\"",
			wantErr:  false,
		},
		{
			name:    "invalid type - string",
			value:   "hello",
			wantErr: true,
		},
		{
			name:    "invalid type - nil",
			value:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.BytesFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_TimestampFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "valid timestamp",
			value:    time.Date(2023, 12, 25, 10, 30, 45, 123456000, time.UTC),
			expected: "TIMESTAMP '2023-12-25T10:30:45.123456Z'",
			wantErr:  false,
		},
		{
			name:     "timestamp with timezone - converts to UTC",
			value:    time.Date(2023, 12, 25, 10, 30, 45, 0, time.FixedZone("PST", -8*3600)),
			expected: "TIMESTAMP '2023-12-25T18:30:45Z'",
			wantErr:  false,
		},
		{
			name:     "epoch timestamp",
			value:    time.Unix(0, 0),
			expected: "TIMESTAMP '1970-01-01T00:00:00Z'",
			wantErr:  false,
		},
		{
			name:    "invalid type - string",
			value:   "2023-12-25T10:30:45Z",
			wantErr: true,
		},
		{
			name:    "invalid type - int64",
			value:   int64(1703505045),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.TimestampFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_DateFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "valid date",
			value:    civil.Date{Year: 2023, Month: 12, Day: 25},
			expected: "DATE '2023-12-25'",
			wantErr:  false,
		},
		{
			name:     "leap year date",
			value:    civil.Date{Year: 2020, Month: 2, Day: 29},
			expected: "DATE '2020-02-29'",
			wantErr:  false,
		},
		{
			name:     "first day of year",
			value:    civil.Date{Year: 2023, Month: 1, Day: 1},
			expected: "DATE '2023-01-01'",
			wantErr:  false,
		},
		{
			name:    "invalid type - time.Time",
			value:   time.Now(),
			wantErr: true,
		},
		{
			name:    "invalid type - string",
			value:   "2023-12-25",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.DateFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_TimeFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "valid time",
			value:    civil.Time{Hour: 10, Minute: 30, Second: 45, Nanosecond: 123456000},
			expected: "TIME '10:30:45.123456'",
			wantErr:  false,
		},
		{
			name:     "midnight",
			value:    civil.Time{Hour: 0, Minute: 0, Second: 0, Nanosecond: 0},
			expected: "TIME '00:00:00'",
			wantErr:  false,
		},
		{
			name:     "end of day",
			value:    civil.Time{Hour: 23, Minute: 59, Second: 59, Nanosecond: 999999999},
			expected: "TIME '23:59:59.1000000'",
			wantErr:  false,
		},
		{
			name:    "invalid type - time.Time",
			value:   time.Now(),
			wantErr: true,
		},
		{
			name:    "invalid type - string",
			value:   "10:30:45",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.TimeFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_DateTimeFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name: "valid datetime",
			value: civil.DateTime{
				Date: civil.Date{Year: 2023, Month: 12, Day: 25},
				Time: civil.Time{Hour: 10, Minute: 30, Second: 45, Nanosecond: 123456000},
			},
			expected: "DATETIME '2023-12-25 10:30:45.123456'",
			wantErr:  false,
		},
		{
			name: "datetime at midnight",
			value: civil.DateTime{
				Date: civil.Date{Year: 2023, Month: 1, Day: 1},
				Time: civil.Time{Hour: 0, Minute: 0, Second: 0, Nanosecond: 0},
			},
			expected: "DATETIME '2023-01-01 00:00:00'",
			wantErr:  false,
		},
		{
			name:    "invalid type - time.Time",
			value:   time.Now(),
			wantErr: true,
		},
		{
			name:    "invalid type - string",
			value:   "2023-12-25T10:30:45",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.DateTimeFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_NumericFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "valid numeric - integer",
			value:    big.NewRat(12345, 1),
			expected: "NUMERIC '12345.000000000'",
			wantErr:  false,
		},
		{
			name:     "valid numeric - decimal",
			value:    big.NewRat(12345, 100), // 123.45
			expected: "NUMERIC '123.450000000'",
			wantErr:  false,
		},
		{
			name:     "negative numeric",
			value:    big.NewRat(-789, 10), // -78.9
			expected: "NUMERIC '-78.900000000'",
			wantErr:  false,
		},
		{
			name:     "zero numeric",
			value:    big.NewRat(0, 1),
			expected: "NUMERIC '0.000000000'",
			wantErr:  false,
		},
		{
			name:     "small decimal",
			value:    big.NewRat(123, 1000), // 0.123
			expected: "NUMERIC '0.123000000'",
			wantErr:  false,
		},
		{
			name:    "invalid type - string",
			value:   "123.45",
			wantErr: true,
		},
		{
			name:    "invalid type - float64",
			value:   float64(123.45),
			wantErr: true,
		},
		{
			name:    "invalid type - int64",
			value:   int64(123),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.NumericFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_BigNumericFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "valid bignumeric - integer",
			value:    big.NewRat(123456789012345, 1),
			expected: "BIGNUMERIC '123456789012345.00000000000000000000000000000000000000'",
			wantErr:  false,
		},
		{
			name:     "valid bignumeric - decimal",
			value:    big.NewRat(123456789, 1000), // 123456.789
			expected: "BIGNUMERIC '123456.78900000000000000000000000000000000000'",
			wantErr:  false,
		},
		{
			name:     "negative bignumeric",
			value:    big.NewRat(-987654321, 100), // -9876543.21
			expected: "BIGNUMERIC '-9876543.21000000000000000000000000000000000000'",
			wantErr:  false,
		},
		{
			name:     "zero bignumeric",
			value:    big.NewRat(0, 1),
			expected: "BIGNUMERIC '0.00000000000000000000000000000000000000'",
			wantErr:  false,
		},
		{
			name:     "very large number",
			value:    new(big.Rat).SetFrac(big.NewInt(999999999999999999), big.NewInt(1)),
			expected: "BIGNUMERIC '999999999999999999.00000000000000000000000000000000000000'",
			wantErr:  false,
		},
		{
			name:    "invalid type - string",
			value:   "123.45",
			wantErr: true,
		},
		{
			name:    "invalid type - float64",
			value:   float64(123.45),
			wantErr: true,
		},
		{
			name:    "invalid type - int64",
			value:   int64(123),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: bigquery.BigNumericFieldType,
			}

			err := c.set(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, c.WhereVal)
			}
		})
	}
}

func TestCursorState_Set_UnsupportedFieldTypes(t *testing.T) {
	tests := []struct {
		name      string
		fieldType bigquery.FieldType
		value     interface{}
	}{
		{
			name:      "boolean field type",
			fieldType: bigquery.BooleanFieldType,
			value:     true,
		},
		{
			name:      "record field type",
			fieldType: bigquery.RecordFieldType,
			value:     map[string]interface{}{"key": "value"},
		},
		{
			name:      "geography field type",
			fieldType: bigquery.GeographyFieldType,
			value:     "POINT(-122.084 37.422)",
		},
		{
			name:      "interval field type",
			fieldType: bigquery.IntervalFieldType,
			value:     "1 YEAR 2 MONTH 3 DAY",
		},
		{
			name:      "json field type",
			fieldType: bigquery.JSONFieldType,
			value:     `{"key": "value"}`,
		},
		{
			name:      "range field type",
			fieldType: bigquery.RangeFieldType,
			value:     "[2023-01-01, 2023-12-31)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: tt.fieldType,
			}

			err := c.set(field, tt.value)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported cursor field type")
		})
	}
}

func TestCursorState_Set_TypeMismatchErrors(t *testing.T) {
	tests := []struct {
		name          string
		fieldType     bigquery.FieldType
		value         interface{}
		expectedError string
	}{
		{
			name:          "string field with int value",
			fieldType:     bigquery.StringFieldType,
			value:         123,
			expectedError: "expected string value for STRING field, got int",
		},
		{
			name:          "integer field with string value",
			fieldType:     bigquery.IntegerFieldType,
			value:         "123",
			expectedError: "expected int64 value for INTEGER field, got string",
		},
		{
			name:          "float field with string value",
			fieldType:     bigquery.FloatFieldType,
			value:         "123.45",
			expectedError: "expected float64 value for FLOAT field, got string",
		},
		{
			name:          "bytes field with string value",
			fieldType:     bigquery.BytesFieldType,
			value:         "hello",
			expectedError: "expected []byte value for BYTES field, got string",
		},
		{
			name:          "timestamp field with string value",
			fieldType:     bigquery.TimestampFieldType,
			value:         "2023-01-01T00:00:00Z",
			expectedError: "expected time.Time value for TIMESTAMP field, got string",
		},
		{
			name:          "date field with string value",
			fieldType:     bigquery.DateFieldType,
			value:         "2023-01-01",
			expectedError: "expected civil.Date value for DATE field, got string",
		},
		{
			name:          "time field with string value",
			fieldType:     bigquery.TimeFieldType,
			value:         "10:30:45",
			expectedError: "expected civil.Time value for TIME field, got string",
		},
		{
			name:          "datetime field with string value",
			fieldType:     bigquery.DateTimeFieldType,
			value:         "2023-01-01T10:30:45",
			expectedError: "expected civil.DateTime value for DATETIME field, got string",
		},
		{
			name:          "numeric field with float value",
			fieldType:     bigquery.NumericFieldType,
			value:         123.45,
			expectedError: "expected *big.Rat value for NUMERIC field, got float64",
		},
		{
			name:          "bignumeric field with float value",
			fieldType:     bigquery.BigNumericFieldType,
			value:         123.45,
			expectedError: "expected *big.Rat value for BIGNUMERIC field, got float64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursorState{}
			field := &bigquery.FieldSchema{
				Type: tt.fieldType,
			}

			err := c.set(field, tt.value)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestGetTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		field    *bigquery.FieldSchema
		value    bigquery.Value
		expected time.Time
		wantErr  bool
		errorMsg string
	}{
		{
			name: "valid timestamp field with time.Time value",
			field: &bigquery.FieldSchema{
				Type: bigquery.TimestampFieldType,
			},
			value:    time.Date(2023, 12, 25, 10, 30, 45, 123456000, time.UTC),
			expected: time.Date(2023, 12, 25, 10, 30, 45, 123456000, time.UTC),
			wantErr:  false,
		},
		{
			name: "valid timestamp field with timezone - preserves original",
			field: &bigquery.FieldSchema{
				Type: bigquery.TimestampFieldType,
			},
			value:    time.Date(2023, 12, 25, 10, 30, 45, 0, time.FixedZone("PST", -8*3600)),
			expected: time.Date(2023, 12, 25, 10, 30, 45, 0, time.FixedZone("PST", -8*3600)),
			wantErr:  false,
		},
		{
			name: "valid timestamp field with zero time",
			field: &bigquery.FieldSchema{
				Type: bigquery.TimestampFieldType,
			},
			value:    time.Unix(0, 0),
			expected: time.Unix(0, 0),
			wantErr:  false,
		},
		{
			name: "non-timestamp field type - string field",
			field: &bigquery.FieldSchema{
				Type: bigquery.StringFieldType,
			},
			value:    time.Now(),
			wantErr:  true,
			errorMsg: "timestamp_field is not of type TIMESTAMP",
		},
		{
			name: "non-timestamp field type - integer field",
			field: &bigquery.FieldSchema{
				Type: bigquery.IntegerFieldType,
			},
			value:    time.Now(),
			wantErr:  true,
			errorMsg: "timestamp_field is not of type TIMESTAMP",
		},
		{
			name: "non-timestamp field type - date field",
			field: &bigquery.FieldSchema{
				Type: bigquery.DateFieldType,
			},
			value:    time.Now(),
			wantErr:  true,
			errorMsg: "timestamp_field is not of type TIMESTAMP",
		},
		{
			name: "timestamp field with non-time.Time value - string",
			field: &bigquery.FieldSchema{
				Type: bigquery.TimestampFieldType,
			},
			value:    "2023-12-25T10:30:45Z",
			wantErr:  true,
			errorMsg: "timestamp_field is not time.Time",
		},
		{
			name: "timestamp field with non-time.Time value - int64",
			field: &bigquery.FieldSchema{
				Type: bigquery.TimestampFieldType,
			},
			value:    int64(1703505045),
			wantErr:  true,
			errorMsg: "timestamp_field is not time.Time",
		},
		{
			name: "timestamp field with non-time.Time value - nil",
			field: &bigquery.FieldSchema{
				Type: bigquery.TimestampFieldType,
			},
			value:    nil,
			wantErr:  true,
			errorMsg: "timestamp_field is not time.Time",
		},
		{
			name: "timestamp field with non-time.Time value - civil.Date",
			field: &bigquery.FieldSchema{
				Type: bigquery.TimestampFieldType,
			},
			value:    civil.Date{Year: 2023, Month: 12, Day: 25},
			wantErr:  true,
			errorMsg: "timestamp_field is not time.Time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getTimestamp(tt.field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.True(t, result.IsZero()) // Should return zero time on error
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExpandJSON(t *testing.T) {
	tests := []struct {
		name     string
		field    *bigquery.FieldSchema
		value    bigquery.Value
		expected interface{}
		wantErr  bool
		errorMsg string
	}{
		// Test that interface{} correctly handles all JSON types
		{
			name: "JSON field with JSON object - validates map[string]interface{} output",
			field: &bigquery.FieldSchema{
				Type: bigquery.JSONFieldType,
			},
			value:    `{"name": "John", "age": 30}`,
			expected: map[string]interface{}{"name": "John", "age": float64(30)},
			wantErr:  false,
		},
		{
			name: "JSON field with JSON array - validates []interface{} output",
			field: &bigquery.FieldSchema{
				Type: bigquery.JSONFieldType,
			},
			value:    `["apple", "banana", "cherry"]`,
			expected: []interface{}{"apple", "banana", "cherry"},
			wantErr:  false,
		},
		{
			name: "JSON field with JSON number - validates float64 output",
			field: &bigquery.FieldSchema{
				Type: bigquery.JSONFieldType,
			},
			value:    `42.5`,
			expected: float64(42.5),
			wantErr:  false,
		},
		{
			name: "JSON field with JSON string - validates string output",
			field: &bigquery.FieldSchema{
				Type: bigquery.JSONFieldType,
			},
			value:    `"hello world"`,
			expected: "hello world",
			wantErr:  false,
		},
		{
			name: "JSON field with JSON boolean - validates bool output",
			field: &bigquery.FieldSchema{
				Type: bigquery.JSONFieldType,
			},
			value:    `true`,
			expected: true,
			wantErr:  false,
		},
		{
			name: "JSON field with JSON null - validates nil output",
			field: &bigquery.FieldSchema{
				Type: bigquery.JSONFieldType,
			},
			value:    `null`,
			expected: nil,
			wantErr:  false,
		},
		// Test business logic - when NOT to parse JSON
		{
			name: "non-JSON field with JSON-like string - should return original",
			field: &bigquery.FieldSchema{
				Type: bigquery.StringFieldType,
			},
			value:    `{"looks": "like json"}`,
			expected: `{"looks": "like json"}`,
			wantErr:  false,
		},
		{
			name: "JSON field with non-string value - should return original",
			field: &bigquery.FieldSchema{
				Type: bigquery.JSONFieldType,
			},
			value:    int64(123),
			expected: int64(123),
			wantErr:  false,
		},
		{
			name: "JSON field with nil value - should return nil",
			field: &bigquery.FieldSchema{
				Type: bigquery.JSONFieldType,
			},
			value:    nil,
			expected: nil,
			wantErr:  false,
		},
		// Test error handling
		{
			name: "JSON field with invalid JSON - should return error",
			field: &bigquery.FieldSchema{
				Type: bigquery.JSONFieldType,
			},
			value:    `{"invalid": json}`,
			wantErr:  true,
			errorMsg: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandJSON(tt.field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
