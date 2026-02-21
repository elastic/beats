// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

// Value represents a cursor value that can be serialized and passed to SQL drivers
type Value struct {
	Type      string `json:"type"`                // "integer", "timestamp", "date", "float", or "decimal"
	Raw       string `json:"raw"`                 // String representation for persistence
	Timestamp int64  `json:"timestamp,omitempty"` // Unix nanoseconds (timestamp only)
}

// ParseValue creates a Value from a string representation.
// This is used to parse the default value from config and stored state.
func ParseValue(raw, valueType string) (*Value, error) {
	v := &Value{Type: valueType, Raw: raw}

	switch valueType {
	case CursorTypeInteger:
		if _, err := strconv.ParseInt(raw, 10, 64); err != nil {
			return nil, fmt.Errorf("invalid integer: %w", err)
		}

	case CursorTypeTimestamp:
		t, err := parseTimestampString(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp: %w", err)
		}
		v.Timestamp = t.UnixNano()
		v.Raw = t.Format(time.RFC3339Nano) // Normalize format

	case CursorTypeDate:
		d, err := parseDateString(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid date: %w", err)
		}
		v.Raw = d.Format("2006-01-02") // Normalize format

	case CursorTypeFloat:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float: %w", err)
		}
		// Reject special IEEE 754 values that would break comparisons and query semantics.
		if math.IsNaN(f) {
			return nil, errors.New("value is NaN")
		}
		if math.IsInf(f, 0) {
			return nil, fmt.Errorf("value is infinite: %f", f)
		}
		// Normalize: re-format to ensure consistent representation
		v.Raw = strconv.FormatFloat(f, 'g', -1, 64)

	case CursorTypeDecimal:
		d, err := decimal.NewFromString(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid decimal: %w", err)
		}
		// Normalize: use decimal library's canonical string representation
		v.Raw = d.String()

	default:
		return nil, fmt.Errorf("unsupported cursor type: %s", valueType)
	}

	return v, nil
}

// FromDatabaseValue creates a Value from a database result.
// This handles the various types that SQL drivers may return.
func FromDatabaseValue(dbVal interface{}, valueType string) (*Value, error) {
	if dbVal == nil {
		return nil, errors.New("column value is NULL")
	}

	switch valueType {
	case CursorTypeInteger:
		return parseIntegerFromDB(dbVal)
	case CursorTypeTimestamp:
		return parseTimestampFromDB(dbVal)
	case CursorTypeDate:
		return parseDateFromDB(dbVal)
	case CursorTypeFloat:
		return parseFloatFromDB(dbVal)
	case CursorTypeDecimal:
		return parseDecimalFromDB(dbVal)
	default:
		return nil, fmt.Errorf("unsupported cursor type: %s", valueType)
	}
}

// ToDriverArg converts the Value to a type suitable for database/sql Query.
// The returned value can be passed directly to db.QueryContext().
func (v *Value) ToDriverArg() interface{} {
	switch v.Type {
	case CursorTypeInteger:
		i, _ := strconv.ParseInt(v.Raw, 10, 64)
		return i
	case CursorTypeTimestamp:
		return time.Unix(0, v.Timestamp).UTC()
	case CursorTypeDate:
		return v.Raw
	case CursorTypeFloat:
		f, _ := strconv.ParseFloat(v.Raw, 64)
		return f
	case CursorTypeDecimal:
		// Most SQL drivers accept string for DECIMAL/NUMERIC bind parameters.
		// If a driver doesn't, the user can cast in SQL: CAST(:cursor AS DECIMAL(10,2))
		return v.Raw
	default:
		return v.Raw
	}
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other.
// Returns an error if the types don't match.
func (v *Value) Compare(other *Value) (int, error) {
	if v.Type != other.Type {
		return 0, fmt.Errorf("cannot compare %s with %s", v.Type, other.Type)
	}

	switch v.Type {
	case CursorTypeInteger:
		a, _ := strconv.ParseInt(v.Raw, 10, 64)
		b, _ := strconv.ParseInt(other.Raw, 10, 64)
		return compareInt64(a, b), nil

	case CursorTypeTimestamp:
		return compareInt64(v.Timestamp, other.Timestamp), nil

	case CursorTypeDate:
		if v.Raw < other.Raw {
			return -1, nil
		} else if v.Raw > other.Raw {
			return 1, nil
		}
		return 0, nil

	case CursorTypeFloat:
		a, _ := strconv.ParseFloat(v.Raw, 64)
		b, _ := strconv.ParseFloat(other.Raw, 64)
		return compareFloat64(a, b), nil

	case CursorTypeDecimal:
		a, errA := decimal.NewFromString(v.Raw)
		b, errB := decimal.NewFromString(other.Raw)
		if errA != nil || errB != nil {
			return 0, fmt.Errorf("failed to parse decimal for comparison: a=%q b=%q", v.Raw, other.Raw)
		}
		return a.Cmp(b), nil
	}

	return 0, fmt.Errorf("unsupported type: %s", v.Type)
}

// String returns the string representation of the cursor value.
func (v *Value) String() string {
	return v.Raw
}

func compareInt64(a, b int64) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

func compareFloat64(a, b float64) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// --- Integer parsing ---

func parseIntegerFromDB(dbVal interface{}) (*Value, error) {
	var intVal int64

	switch v := dbVal.(type) {
	case int:
		intVal = int64(v)
	case int32:
		intVal = int64(v)
	case int64:
		intVal = v
	case uint:
		if uint64(v) > math.MaxInt64 {
			return nil, fmt.Errorf("uint overflow: %d exceeds max int64", v)
		}
		intVal = int64(uint64(v)) //nolint:gosec // overflow checked above
	case uint32:
		intVal = int64(v)
	case uint64:
		if v > math.MaxInt64 {
			return nil, fmt.Errorf("uint64 overflow: %d exceeds max int64", v)
		}
		intVal = int64(v)
	case float32:
		// Some drivers may return float32 for certain numeric types
		if v > float32(math.MaxInt64) || v < float32(math.MinInt64) {
			return nil, fmt.Errorf("float32 overflow: %f exceeds int64 range", v)
		}
		intVal = int64(v)
	case float64:
		if v > float64(math.MaxInt64) || v < float64(math.MinInt64) {
			return nil, fmt.Errorf("float64 overflow: %f exceeds int64 range", v)
		}
		intVal = int64(v)
	case []byte:
		parsed, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse []byte as integer: %w", err)
		}
		intVal = parsed
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse string as integer: %w", err)
		}
		intVal = parsed
	default:
		return nil, fmt.Errorf("unsupported integer type: %T", dbVal)
	}

	return &Value{
		Type: CursorTypeInteger,
		Raw:  strconv.FormatInt(intVal, 10),
	}, nil
}

// --- Float parsing ---

func parseFloatFromDB(dbVal interface{}) (*Value, error) {
	var floatVal float64

	switch v := dbVal.(type) {
	case float32:
		floatVal = float64(v)
	case float64:
		floatVal = v
	case int:
		floatVal = float64(v)
	case int32:
		floatVal = float64(v)
	case int64:
		floatVal = float64(v)
	case []byte:
		parsed, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse []byte as float: %w", err)
		}
		floatVal = parsed
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse string as float: %w", err)
		}
		floatVal = parsed
	default:
		return nil, fmt.Errorf("unsupported float type: %T", dbVal)
	}

	if math.IsNaN(floatVal) {
		return nil, errors.New("value is NaN")
	}
	if math.IsInf(floatVal, 0) {
		return nil, fmt.Errorf("value is infinite: %f", floatVal)
	}

	return &Value{
		Type: CursorTypeFloat,
		Raw:  strconv.FormatFloat(floatVal, 'g', -1, 64),
	}, nil
}

// --- Decimal parsing ---

func parseDecimalFromDB(dbVal interface{}) (*Value, error) {
	var d decimal.Decimal

	switch v := dbVal.(type) {
	case float32:
		// Convert via string to avoid float64 precision loss:
		// decimal.NewFromFloat32 preserves the float32 representation.
		d = decimal.NewFromFloat32(v)
	case float64:
		// Use string conversion for better precision preservation:
		// float64 -> string -> decimal avoids double rounding.
		d = decimal.NewFromFloat(v)
	case []byte:
		parsed, err := decimal.NewFromString(string(v))
		if err != nil {
			return nil, fmt.Errorf("cannot parse []byte as decimal: %w", err)
		}
		d = parsed
	case string:
		parsed, err := decimal.NewFromString(v)
		if err != nil {
			return nil, fmt.Errorf("cannot parse string as decimal: %w", err)
		}
		d = parsed
	case int:
		d = decimal.NewFromInt(int64(v))
	case int32:
		d = decimal.NewFromInt32(v)
	case int64:
		d = decimal.NewFromInt(v)
	default:
		return nil, fmt.Errorf("unsupported decimal type: %T", dbVal)
	}

	return &Value{
		Type: CursorTypeDecimal,
		Raw:  d.String(),
	}, nil
}

// --- Timestamp parsing ---

// timestampFormats lists the supported timestamp formats in order of preference.
// RFC3339Nano is first as it's the canonical format we use for storage.
var timestampFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05.999999",
	"2006-01-02 15:04:05.999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func parseTimestampString(s string) (time.Time, error) {
	for _, format := range timestampFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp: %s", s)
}

func parseTimestampFromDB(dbVal interface{}) (*Value, error) {
	var t time.Time

	switch v := dbVal.(type) {
	case time.Time:
		t = v.UTC()
	case *time.Time:
		if v == nil {
			return nil, errors.New("column value is NULL")
		}
		t = v.UTC()
	case []byte:
		// MySQL returns timestamps as []byte
		parsed, err := parseTimestampString(string(v))
		if err != nil {
			return nil, fmt.Errorf("cannot parse []byte as timestamp: %w", err)
		}
		t = parsed
	case string:
		parsed, err := parseTimestampString(v)
		if err != nil {
			return nil, fmt.Errorf("cannot parse string as timestamp: %w", err)
		}
		t = parsed
	default:
		return nil, fmt.Errorf("unsupported timestamp type: %T", dbVal)
	}

	return &Value{
		Type:      CursorTypeTimestamp,
		Raw:       t.Format(time.RFC3339Nano),
		Timestamp: t.UnixNano(),
	}, nil
}

// --- Date parsing ---

func parseDateString(s string) (time.Time, error) {
	// Try the canonical date format first
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	// Fall back to timestamp parsing (extracts date portion)
	if t, err := parseTimestampString(s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %s", s)
}

func parseDateFromDB(dbVal interface{}) (*Value, error) {
	var d time.Time

	switch v := dbVal.(type) {
	case time.Time:
		d = v.UTC()
	case *time.Time:
		if v == nil {
			return nil, errors.New("column value is NULL")
		}
		d = v.UTC()
	case []byte:
		parsed, err := parseDateString(string(v))
		if err != nil {
			return nil, fmt.Errorf("cannot parse []byte as date: %w", err)
		}
		d = parsed
	case string:
		parsed, err := parseDateString(v)
		if err != nil {
			return nil, fmt.Errorf("cannot parse string as date: %w", err)
		}
		d = parsed
	default:
		return nil, fmt.Errorf("unsupported date type: %T", dbVal)
	}

	return &Value{
		Type: CursorTypeDate,
		Raw:  d.Format("2006-01-02"),
	}, nil
}
