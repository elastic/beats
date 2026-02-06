// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		valueType  string
		wantErr    bool
		wantRaw    string
		wantTsNano int64
	}{
		// Integer tests
		{
			name:      "integer - zero",
			raw:       "0",
			valueType: CursorTypeInteger,
			wantErr:   false,
			wantRaw:   "0",
		},
		{
			name:      "integer - positive",
			raw:       "12345",
			valueType: CursorTypeInteger,
			wantErr:   false,
			wantRaw:   "12345",
		},
		{
			name:      "integer - negative",
			raw:       "-12345",
			valueType: CursorTypeInteger,
			wantErr:   false,
			wantRaw:   "-12345",
		},
		{
			name:      "integer - max int64",
			raw:       "9223372036854775807",
			valueType: CursorTypeInteger,
			wantErr:   false,
			wantRaw:   "9223372036854775807",
		},
		{
			name:      "integer - invalid",
			raw:       "not-a-number",
			valueType: CursorTypeInteger,
			wantErr:   true,
		},
		{
			name:      "integer - float string",
			raw:       "123.45",
			valueType: CursorTypeInteger,
			wantErr:   true,
		},

		// Timestamp tests
		{
			name:       "timestamp - RFC3339",
			raw:        "2024-01-15T10:30:00Z",
			valueType:  CursorTypeTimestamp,
			wantErr:    false,
			wantRaw:    "2024-01-15T10:30:00Z",
			wantTsNano: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
		},
		{
			name:       "timestamp - RFC3339Nano",
			raw:        "2024-01-15T10:30:00.123456789Z",
			valueType:  CursorTypeTimestamp,
			wantErr:    false,
			wantRaw:    "2024-01-15T10:30:00.123456789Z",
			wantTsNano: time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC).UnixNano(),
		},
		{
			name:       "timestamp - space format",
			raw:        "2024-01-15 10:30:00",
			valueType:  CursorTypeTimestamp,
			wantErr:    false,
			wantRaw:    "2024-01-15T10:30:00Z",
			wantTsNano: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
		},
		{
			name:      "timestamp - invalid",
			raw:       "not-a-timestamp",
			valueType: CursorTypeTimestamp,
			wantErr:   true,
		},

		// Date tests
		{
			name:      "date - valid",
			raw:       "2024-01-15",
			valueType: CursorTypeDate,
			wantErr:   false,
			wantRaw:   "2024-01-15",
		},
		{
			name:      "date - from timestamp",
			raw:       "2024-01-15T10:30:00Z",
			valueType: CursorTypeDate,
			wantErr:   false,
			wantRaw:   "2024-01-15",
		},
		{
			name:      "date - invalid",
			raw:       "not-a-date",
			valueType: CursorTypeDate,
			wantErr:   true,
		},

		// Unsupported type
		{
			name:      "unsupported type",
			raw:       "value",
			valueType: "unsupported",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ParseValue(tt.raw, tt.valueType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, tt.valueType, val.Type)
			if tt.valueType == CursorTypeTimestamp {
				assert.Equal(t, tt.wantTsNano, val.Timestamp)
			}
		})
	}
}

func TestFromDatabaseValue_Integer(t *testing.T) {
	tests := []struct {
		name    string
		dbVal   interface{}
		wantRaw string
		wantErr bool
	}{
		{
			name:    "int",
			dbVal:   int(123),
			wantRaw: "123",
		},
		{
			name:    "int32",
			dbVal:   int32(123),
			wantRaw: "123",
		},
		{
			name:    "int64",
			dbVal:   int64(123),
			wantRaw: "123",
		},
		{
			name:    "uint",
			dbVal:   uint(123),
			wantRaw: "123",
		},
		{
			name:    "uint32",
			dbVal:   uint32(123),
			wantRaw: "123",
		},
		{
			name:    "uint64",
			dbVal:   uint64(123),
			wantRaw: "123",
		},
		{
			name:    "uint64 overflow",
			dbVal:   uint64(math.MaxUint64),
			wantErr: true,
		},
		{
			name:    "uint overflow (on 64-bit systems)",
			dbVal:   uint(math.MaxUint64),
			wantErr: true,
		},
		{
			name:    "float32",
			dbVal:   float32(123),
			wantRaw: "123",
		},
		{
			name:    "float32 overflow",
			dbVal:   float32(math.MaxFloat32),
			wantErr: true,
		},
		{
			name:    "float64",
			dbVal:   float64(123),
			wantRaw: "123",
		},
		{
			name:    "float64 overflow",
			dbVal:   float64(math.MaxFloat64),
			wantErr: true,
		},
		{
			name:    "[]byte",
			dbVal:   []byte("123"),
			wantRaw: "123",
		},
		{
			name:    "[]byte invalid",
			dbVal:   []byte("not-a-number"),
			wantErr: true,
		},
		{
			name:    "string",
			dbVal:   "123",
			wantRaw: "123",
		},
		{
			name:    "string invalid",
			dbVal:   "not-a-number",
			wantErr: true,
		},
		{
			name:    "nil",
			dbVal:   nil,
			wantErr: true,
		},
		{
			name:    "unsupported type",
			dbVal:   struct{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := FromDatabaseValue(tt.dbVal, CursorTypeInteger)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, CursorTypeInteger, val.Type)
		})
	}
}

func TestFromDatabaseValue_Timestamp(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name       string
		dbVal      interface{}
		wantRaw    string
		wantTsNano int64
		wantErr    bool
	}{
		{
			name:       "time.Time",
			dbVal:      baseTime,
			wantRaw:    "2024-01-15T10:30:00Z",
			wantTsNano: baseTime.UnixNano(),
		},
		{
			name:       "*time.Time",
			dbVal:      &baseTime,
			wantRaw:    "2024-01-15T10:30:00Z",
			wantTsNano: baseTime.UnixNano(),
		},
		{
			name:    "*time.Time nil",
			dbVal:   (*time.Time)(nil),
			wantErr: true,
		},
		{
			name:       "[]byte",
			dbVal:      []byte("2024-01-15 10:30:00"),
			wantRaw:    "2024-01-15T10:30:00Z",
			wantTsNano: baseTime.UnixNano(),
		},
		{
			name:    "[]byte invalid",
			dbVal:   []byte("invalid"),
			wantErr: true,
		},
		{
			name:       "string",
			dbVal:      "2024-01-15 10:30:00",
			wantRaw:    "2024-01-15T10:30:00Z",
			wantTsNano: baseTime.UnixNano(),
		},
		{
			name:    "string invalid",
			dbVal:   "invalid",
			wantErr: true,
		},
		{
			name:    "nil",
			dbVal:   nil,
			wantErr: true,
		},
		{
			name:    "unsupported type",
			dbVal:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := FromDatabaseValue(tt.dbVal, CursorTypeTimestamp)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, tt.wantTsNano, val.Timestamp)
			assert.Equal(t, CursorTypeTimestamp, val.Type)
		})
	}
}

func TestFromDatabaseValue_Date(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		dbVal   interface{}
		wantRaw string
		wantErr bool
	}{
		{
			name:    "time.Time",
			dbVal:   baseTime,
			wantRaw: "2024-01-15",
		},
		{
			name:    "*time.Time",
			dbVal:   &baseTime,
			wantRaw: "2024-01-15",
		},
		{
			name:    "*time.Time nil",
			dbVal:   (*time.Time)(nil),
			wantErr: true,
		},
		{
			name:    "[]byte",
			dbVal:   []byte("2024-01-15"),
			wantRaw: "2024-01-15",
		},
		{
			name:    "[]byte invalid",
			dbVal:   []byte("invalid"),
			wantErr: true,
		},
		{
			name:    "string",
			dbVal:   "2024-01-15",
			wantRaw: "2024-01-15",
		},
		{
			name:    "string invalid",
			dbVal:   "invalid",
			wantErr: true,
		},
		{
			name:    "nil",
			dbVal:   nil,
			wantErr: true,
		},
		{
			name:    "unsupported type",
			dbVal:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := FromDatabaseValue(tt.dbVal, CursorTypeDate)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, CursorTypeDate, val.Type)
		})
	}
}

func TestValueToDriverArg(t *testing.T) {
	tests := []struct {
		name      string
		value     *Value
		wantType  string
		wantValue interface{}
	}{
		{
			name: "integer",
			value: &Value{
				Type: CursorTypeInteger,
				Raw:  "12345",
			},
			wantType:  "int64",
			wantValue: int64(12345),
		},
		{
			name: "timestamp",
			value: &Value{
				Type:      CursorTypeTimestamp,
				Raw:       "2024-01-15T10:30:00Z",
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
			},
			wantType:  "time.Time",
			wantValue: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name: "date",
			value: &Value{
				Type: CursorTypeDate,
				Raw:  "2024-01-15",
			},
			wantType:  "string",
			wantValue: "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.value.ToDriverArg()
			assert.IsType(t, tt.wantValue, result)
			assert.Equal(t, tt.wantValue, result)
		})
	}
}

func TestValueCompare(t *testing.T) {
	tests := []struct {
		name    string
		v1      *Value
		v2      *Value
		want    int
		wantErr bool
	}{
		// Integer comparisons
		{
			name:    "integer equal",
			v1:      &Value{Type: CursorTypeInteger, Raw: "100"},
			v2:      &Value{Type: CursorTypeInteger, Raw: "100"},
			want:    0,
			wantErr: false,
		},
		{
			name:    "integer less than",
			v1:      &Value{Type: CursorTypeInteger, Raw: "100"},
			v2:      &Value{Type: CursorTypeInteger, Raw: "200"},
			want:    -1,
			wantErr: false,
		},
		{
			name:    "integer greater than",
			v1:      &Value{Type: CursorTypeInteger, Raw: "200"},
			v2:      &Value{Type: CursorTypeInteger, Raw: "100"},
			want:    1,
			wantErr: false,
		},

		// Timestamp comparisons
		{
			name: "timestamp equal",
			v1: &Value{
				Type:      CursorTypeTimestamp,
				Raw:       "2024-01-15T10:30:00Z",
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
			},
			v2: &Value{
				Type:      CursorTypeTimestamp,
				Raw:       "2024-01-15T10:30:00Z",
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "timestamp less than",
			v1: &Value{
				Type:      CursorTypeTimestamp,
				Raw:       "2024-01-15T10:30:00Z",
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
			},
			v2: &Value{
				Type:      CursorTypeTimestamp,
				Raw:       "2024-01-15T11:30:00Z",
				Timestamp: time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC).UnixNano(),
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "timestamp greater than",
			v1: &Value{
				Type:      CursorTypeTimestamp,
				Raw:       "2024-01-15T11:30:00Z",
				Timestamp: time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC).UnixNano(),
			},
			v2: &Value{
				Type:      CursorTypeTimestamp,
				Raw:       "2024-01-15T10:30:00Z",
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
			},
			want:    1,
			wantErr: false,
		},

		// Date comparisons
		{
			name:    "date equal",
			v1:      &Value{Type: CursorTypeDate, Raw: "2024-01-15"},
			v2:      &Value{Type: CursorTypeDate, Raw: "2024-01-15"},
			want:    0,
			wantErr: false,
		},
		{
			name:    "date less than",
			v1:      &Value{Type: CursorTypeDate, Raw: "2024-01-15"},
			v2:      &Value{Type: CursorTypeDate, Raw: "2024-01-16"},
			want:    -1,
			wantErr: false,
		},
		{
			name:    "date greater than",
			v1:      &Value{Type: CursorTypeDate, Raw: "2024-01-16"},
			v2:      &Value{Type: CursorTypeDate, Raw: "2024-01-15"},
			want:    1,
			wantErr: false,
		},

		// Type mismatch
		{
			name:    "type mismatch",
			v1:      &Value{Type: CursorTypeInteger, Raw: "100"},
			v2:      &Value{Type: CursorTypeDate, Raw: "2024-01-15"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.v1.Compare(tt.v2)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestValueString(t *testing.T) {
	val := &Value{Type: CursorTypeInteger, Raw: "12345"}
	assert.Equal(t, "12345", val.String())
}

func TestFromDatabaseValue_UnsupportedType(t *testing.T) {
	_, err := FromDatabaseValue("some_value", "unsupported")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported cursor type")
}

func TestFromDatabaseValue_NilValue(t *testing.T) {
	_, err := FromDatabaseValue(nil, CursorTypeInteger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NULL")
}

func TestValueToDriverArg_UnsupportedType(t *testing.T) {
	val := &Value{Type: "unknown", Raw: "hello"}
	result := val.ToDriverArg()
	assert.Equal(t, "hello", result)
}

func TestValueCompare_UnsupportedType(t *testing.T) {
	v1 := &Value{Type: "unknown", Raw: "a"}
	v2 := &Value{Type: "unknown", Raw: "b"}
	_, err := v1.Compare(v2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestFromDatabaseValue_Integer_NegativeValues(t *testing.T) {
	// Test negative int values
	val, err := FromDatabaseValue(int(-42), CursorTypeInteger)
	require.NoError(t, err)
	assert.Equal(t, "-42", val.Raw)

	// Test negative int64
	val, err = FromDatabaseValue(int64(-9999), CursorTypeInteger)
	require.NoError(t, err)
	assert.Equal(t, "-9999", val.Raw)

	// Test negative float64
	val, err = FromDatabaseValue(float64(-100), CursorTypeInteger)
	require.NoError(t, err)
	assert.Equal(t, "-100", val.Raw)

	// Test negative string
	val, err = FromDatabaseValue("-777", CursorTypeInteger)
	require.NoError(t, err)
	assert.Equal(t, "-777", val.Raw)
}

func TestFromDatabaseValue_Timestamp_WithTimezone(t *testing.T) {
	// Timestamp with timezone offset should be converted to UTC
	loc := time.FixedZone("EST", -5*60*60)
	eastern := time.Date(2024, 6, 15, 10, 0, 0, 0, loc)

	val, err := FromDatabaseValue(eastern, CursorTypeTimestamp)
	require.NoError(t, err)
	// Should be converted to UTC (15:00)
	assert.Equal(t, "2024-06-15T15:00:00Z", val.Raw)
}

func TestFromDatabaseValue_Date_FromTimestamp(t *testing.T) {
	// Date from a time.Time with time portion should extract just the date
	ts := time.Date(2024, 6, 15, 23, 59, 59, 0, time.UTC)
	val, err := FromDatabaseValue(ts, CursorTypeDate)
	require.NoError(t, err)
	assert.Equal(t, "2024-06-15", val.Raw)
}

func TestFromDatabaseValue_Date_StringTimestamp(t *testing.T) {
	// Date from a string that contains a full timestamp
	val, err := FromDatabaseValue("2024-06-15T10:30:00Z", CursorTypeDate)
	require.NoError(t, err)
	assert.Equal(t, "2024-06-15", val.Raw)
}

// ============================================================================
// Float type tests
// ============================================================================

func TestParseValue_Float(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantRaw string
		wantErr bool
	}{
		{"zero", "0.0", "0", false},
		{"positive", "3.14159", "3.14159", false},
		{"negative", "-2.718", "-2.718", false},
		{"integer-like", "42", "42", false},
		{"scientific", "1.5e10", "1.5e+10", false},
		{"very small", "0.000001", "1e-06", false},
		{"nan", "NaN", "", true},
		{"pos inf", "+Inf", "", true},
		{"neg inf", "-Inf", "", true},
		{"invalid", "not-a-float", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ParseValue(tt.raw, CursorTypeFloat)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, CursorTypeFloat, val.Type)
		})
	}
}

func TestFromDatabaseValue_Float(t *testing.T) {
	tests := []struct {
		name    string
		dbVal   interface{}
		wantRaw string
		wantErr bool
	}{
		{"float32", float32(3.14), "3.140000104904175", false},
		{"float64", float64(3.14159), "3.14159", false},
		{"int", int(42), "42", false},
		{"int32", int32(42), "42", false},
		{"int64", int64(42), "42", false},
		{"[]byte", []byte("99.95"), "99.95", false},
		{"string", "99.95", "99.95", false},
		{"[]byte invalid", []byte("abc"), "", true},
		{"string invalid", "abc", "", true},
		{"nil", nil, "", true},
		{"unsupported", struct{}{}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := FromDatabaseValue(tt.dbVal, CursorTypeFloat)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, CursorTypeFloat, val.Type)
		})
	}
}

func TestFromDatabaseValue_Float_NaN(t *testing.T) {
	_, err := FromDatabaseValue(math.NaN(), CursorTypeFloat)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NaN")
}

func TestFromDatabaseValue_Float_Inf(t *testing.T) {
	_, err := FromDatabaseValue(math.Inf(1), CursorTypeFloat)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "infinite")
}

func TestValueCompare_Float(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal", "3.14", "3.14", 0},
		{"less", "2.0", "3.0", -1},
		{"greater", "3.0", "2.0", 1},
		{"negative", "-1.5", "1.5", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1 := &Value{Type: CursorTypeFloat, Raw: tt.a}
			v2 := &Value{Type: CursorTypeFloat, Raw: tt.b}
			cmp, err := v1.Compare(v2)
			require.NoError(t, err)
			assert.Equal(t, tt.want, cmp)
		})
	}
}

func TestValueToDriverArg_Float(t *testing.T) {
	val := &Value{Type: CursorTypeFloat, Raw: "3.14"}
	arg := val.ToDriverArg()
	f, ok := arg.(float64)
	require.True(t, ok)
	assert.InDelta(t, 3.14, f, 0.001)
}

// ============================================================================
// Decimal type tests
// ============================================================================

func TestParseValue_Decimal(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantRaw string
		wantErr bool
	}{
		{"zero", "0", "0", false},
		{"positive", "99.95", "99.95", false},
		{"negative", "-42.50", "-42.5", false},
		{"high precision", "123456789.123456789", "123456789.123456789", false},
		{"integer-like", "100", "100", false},
		{"invalid", "not-a-decimal", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ParseValue(tt.raw, CursorTypeDecimal)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, CursorTypeDecimal, val.Type)
		})
	}
}

func TestFromDatabaseValue_Decimal(t *testing.T) {
	tests := []struct {
		name    string
		dbVal   interface{}
		wantRaw string
		wantErr bool
	}{
		{"float64", float64(99.95), "99.95", false},
		{"float32", float32(3.14), "3.14", false},
		{"int", int(42), "42", false},
		{"int32", int32(42), "42", false},
		{"int64", int64(42), "42", false},
		{"[]byte", []byte("123.456"), "123.456", false},
		{"string", "123.456", "123.456", false},
		{"high precision string", "999999999999.999999999", "999999999999.999999999", false},
		{"[]byte invalid", []byte("abc"), "", true},
		{"string invalid", "abc", "", true},
		{"nil", nil, "", true},
		{"unsupported", struct{}{}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := FromDatabaseValue(tt.dbVal, CursorTypeDecimal)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, CursorTypeDecimal, val.Type)
		})
	}
}

func TestValueCompare_Decimal(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal", "99.95", "99.95", 0},
		{"less", "99.94", "99.95", -1},
		{"greater", "99.96", "99.95", 1},
		{"negative", "-1.5", "1.5", -1},
		{"high precision", "0.123456789", "0.123456788", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1 := &Value{Type: CursorTypeDecimal, Raw: tt.a}
			v2 := &Value{Type: CursorTypeDecimal, Raw: tt.b}
			cmp, err := v1.Compare(v2)
			require.NoError(t, err)
			assert.Equal(t, tt.want, cmp)
		})
	}
}

func TestValueToDriverArg_Decimal(t *testing.T) {
	val := &Value{Type: CursorTypeDecimal, Raw: "99.95"}
	arg := val.ToDriverArg()
	s, ok := arg.(string)
	require.True(t, ok)
	assert.Equal(t, "99.95", s)
}

func TestDecimalPrecisionRoundTrip(t *testing.T) {
	// Critical test: verify that decimal preserves exact values through
	// parse -> store -> load -> compare cycle
	original := "123456789.123456789012345"
	val, err := ParseValue(original, CursorTypeDecimal)
	require.NoError(t, err)

	// Re-parse from stored Raw
	val2, err := ParseValue(val.Raw, CursorTypeDecimal)
	require.NoError(t, err)

	cmp, err := val.Compare(val2)
	require.NoError(t, err)
	assert.Equal(t, 0, cmp, "decimal round-trip must preserve exact value")
}

// ============================================================================
// Timestamp coverage hardening
// ============================================================================

func TestTimestampNanosecondPrecisionRoundTrip(t *testing.T) {
	// Critical test: verify that nanosecond precision is preserved through
	// parse -> Raw -> re-parse -> compare -> ToDriverArg cycle.
	original := "2024-06-15T10:30:00.123456789Z"
	val, err := ParseValue(original, CursorTypeTimestamp)
	require.NoError(t, err)
	assert.Equal(t, original, val.Raw, "Raw must preserve original nanosecond string")

	// Re-parse from stored Raw (simulates store->load)
	val2, err := ParseValue(val.Raw, CursorTypeTimestamp)
	require.NoError(t, err)
	assert.Equal(t, val.Timestamp, val2.Timestamp, "Timestamp (UnixNano) must survive round-trip")

	// Compare must report equal
	cmp, err := val.Compare(val2)
	require.NoError(t, err)
	assert.Equal(t, 0, cmp, "round-tripped timestamp must compare equal")

	// ToDriverArg must produce a time.Time with the exact nanosecond
	arg := val2.ToDriverArg()
	tm, ok := arg.(time.Time)
	require.True(t, ok)
	assert.Equal(t, 123456789, tm.Nanosecond(), "ToDriverArg must preserve nanosecond component")
}

func TestTimestampCompare_NanosecondBoundaries(t *testing.T) {
	// Two timestamps differing only in the nanosecond component must compare correctly.
	ts1 := time.Date(2024, 6, 15, 10, 30, 0, 100, time.UTC) // 100 ns
	ts2 := time.Date(2024, 6, 15, 10, 30, 0, 200, time.UTC) // 200 ns

	v1, err := FromDatabaseValue(ts1, CursorTypeTimestamp)
	require.NoError(t, err)
	v2, err := FromDatabaseValue(ts2, CursorTypeTimestamp)
	require.NoError(t, err)

	cmp, err := v1.Compare(v2)
	require.NoError(t, err)
	assert.Equal(t, -1, cmp, "100ns should be less than 200ns")

	cmp, err = v2.Compare(v1)
	require.NoError(t, err)
	assert.Equal(t, 1, cmp, "200ns should be greater than 100ns")

	// Same nanosecond
	v3, err := FromDatabaseValue(ts1, CursorTypeTimestamp)
	require.NoError(t, err)
	cmp, err = v1.Compare(v3)
	require.NoError(t, err)
	assert.Equal(t, 0, cmp, "same nanosecond should compare equal")
}

func TestParseTimestamp_AllSupportedFormats(t *testing.T) {
	// Verify every format in timestampFormats is correctly parsed.
	// The canonical output is always RFC3339Nano in UTC.
	tests := []struct {
		name       string
		input      string
		wantRaw    string
		wantTsNano int64
	}{
		{
			name:       "RFC3339Nano",
			input:      "2024-06-15T10:30:00.123456789Z",
			wantRaw:    "2024-06-15T10:30:00.123456789Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 123456789, time.UTC).UnixNano(),
		},
		{
			name:       "RFC3339",
			input:      "2024-06-15T10:30:00Z",
			wantRaw:    "2024-06-15T10:30:00Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
		},
		{
			name:       "space 9-digit nanoseconds",
			input:      "2024-06-15 10:30:00.123456789",
			wantRaw:    "2024-06-15T10:30:00.123456789Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 123456789, time.UTC).UnixNano(),
		},
		{
			name:       "space 6-digit microseconds",
			input:      "2024-06-15 10:30:00.123456",
			wantRaw:    "2024-06-15T10:30:00.123456Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 123456000, time.UTC).UnixNano(),
		},
		{
			name:       "space 3-digit milliseconds",
			input:      "2024-06-15 10:30:00.123",
			wantRaw:    "2024-06-15T10:30:00.123Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 123000000, time.UTC).UnixNano(),
		},
		{
			name:       "space seconds only",
			input:      "2024-06-15 10:30:00",
			wantRaw:    "2024-06-15T10:30:00Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
		},
		{
			name:       "T separator no timezone",
			input:      "2024-06-15T10:30:00",
			wantRaw:    "2024-06-15T10:30:00Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
		},
		{
			name:       "date only",
			input:      "2024-06-15",
			wantRaw:    "2024-06-15T00:00:00Z",
			wantTsNano: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC).UnixNano(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ParseValue(tt.input, CursorTypeTimestamp)
			require.NoError(t, err, "format %q should be parseable", tt.input)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, tt.wantTsNano, val.Timestamp, "UnixNano mismatch for format %q", tt.input)
		})
	}
}

func TestFromDatabaseValue_Timestamp_SubsecondPrecision(t *testing.T) {
	// Database drivers may return time.Time with various precision levels.
	// Verify all are correctly captured.
	tests := []struct {
		name    string
		dbVal   time.Time
		wantNs  int
		wantRaw string
	}{
		{
			name:    "milliseconds",
			dbVal:   time.Date(2024, 6, 15, 10, 30, 0, 123000000, time.UTC),
			wantNs:  123000000,
			wantRaw: "2024-06-15T10:30:00.123Z",
		},
		{
			name:    "microseconds",
			dbVal:   time.Date(2024, 6, 15, 10, 30, 0, 123456000, time.UTC),
			wantNs:  123456000,
			wantRaw: "2024-06-15T10:30:00.123456Z",
		},
		{
			name:    "nanoseconds",
			dbVal:   time.Date(2024, 6, 15, 10, 30, 0, 123456789, time.UTC),
			wantNs:  123456789,
			wantRaw: "2024-06-15T10:30:00.123456789Z",
		},
		{
			name:    "no sub-second",
			dbVal:   time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
			wantNs:  0,
			wantRaw: "2024-06-15T10:30:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := FromDatabaseValue(tt.dbVal, CursorTypeTimestamp)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)

			// Verify nanosecond precision is preserved through ToDriverArg
			arg := val.ToDriverArg()
			tm, ok := arg.(time.Time)
			require.True(t, ok)
			assert.Equal(t, tt.wantNs, tm.Nanosecond(),
				"nanosecond component must be preserved through FromDatabaseValue -> ToDriverArg")
		})
	}
}

func TestFromDatabaseValue_Timestamp_ByteFormats(t *testing.T) {
	// MySQL returns timestamps as []byte in various formats.
	// Verify all supported formats are handled.
	tests := []struct {
		name       string
		dbVal      []byte
		wantRaw    string
		wantTsNano int64
	}{
		{
			name:       "MySQL DATETIME (no fractional)",
			dbVal:      []byte("2024-06-15 10:30:00"),
			wantRaw:    "2024-06-15T10:30:00Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC).UnixNano(),
		},
		{
			name:       "MySQL DATETIME(3) milliseconds",
			dbVal:      []byte("2024-06-15 10:30:00.123"),
			wantRaw:    "2024-06-15T10:30:00.123Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 123000000, time.UTC).UnixNano(),
		},
		{
			name:       "MySQL DATETIME(6) microseconds",
			dbVal:      []byte("2024-06-15 10:30:00.123456"),
			wantRaw:    "2024-06-15T10:30:00.123456Z",
			wantTsNano: time.Date(2024, 6, 15, 10, 30, 0, 123456000, time.UTC).UnixNano(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := FromDatabaseValue(tt.dbVal, CursorTypeTimestamp)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRaw, val.Raw)
			assert.Equal(t, tt.wantTsNano, val.Timestamp)
		})
	}
}
