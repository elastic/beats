// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package encoding

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type EncodingFlag int

const (
	// EncodingFlagUseNumbersZeroValues forces numeric zero values to be rendered as "0"
	// instead of empty strings. By default, zero values for int, uint, and float types
	// are converted to empty strings, but this flag preserves them as "0".
	EncodingFlagUseNumbersZeroValues EncodingFlag = 1 << iota

	DefaultTimeFormat = time.RFC3339
	DefaultTimezone   = "UTC"
)

func (f EncodingFlag) has(option EncodingFlag) bool {
	return f&option != 0
}

// MarshalToMap converts a struct, a single-level map (like map[string]string
// or map[string]any), or a pointer to these, into a map[string]string.
// It prioritizes the "osquery" tag for struct fields.
func MarshalToMap(in any) (map[string]string, error) {
	return MarshalToMapWithFlags(in, 0)
}

func MarshalToMapWithFlags(in any, flags EncodingFlag) (map[string]string, error) {
	if in == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}
	result := make(map[string]string)

	v := reflect.ValueOf(in)
	t := reflect.TypeOf(in)

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, fmt.Errorf("input pointer is nil")
		}
		v = v.Elem()
		t = t.Elem()
	}

	if v.Kind() == reflect.Map {
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map keys must be strings, got %s", t.Key().Kind())
		}

		for _, k := range v.MapKeys() {
			key := k.String()
			fieldValue := v.MapIndex(k)

			value, err := convertValueToStringWithTag(fieldValue, flags, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to convert field %s: %w", key, err)
			}
			result[key] = value
		}
		return result, nil
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("unsupported type: %s, must be a struct, map, or pointer to one of them", v.Kind())
	}

	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := t.Field(i)

		if !fieldType.IsExported() {
			continue
		}

		key := fieldType.Tag.Get("osquery")
		switch key {
		case "-":
			continue
		case "":
			key = fieldType.Name
		}

		value, err := convertValueToStringWithTag(fieldValue, flags, &fieldType.Tag)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", key, err)
		}

		result[key] = value
	}

	return result, nil
}

// convertValueToStringWithTag converts a reflect.Value to a string, handling pointers,
// booleans, integers, unsigned integers, floats, time.Time, and unsupported types.
// It also handles the EncodingFlagUseNumbersZeroValues flag and the tag format and tz attributes.
func convertValueToStringWithTag(fieldValue reflect.Value, flag EncodingFlag, tag *reflect.StructTag) (string, error) {
	// Handle pointers first
	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			return "", nil
		}
		return convertValueToStringWithTag(fieldValue.Elem(), flag, tag)
	}

	switch fieldValue.Kind() {
	case reflect.String:
		return fieldValue.String(), nil

	case reflect.Bool:
		// osquery often expects boolean values as "0" or "1"
		if fieldValue.Bool() {
			return "1", nil
		}
		return "0", nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// osquery often expects empty string for 0 values unless necessary
		val := fieldValue.Int()
		if !flag.has(EncodingFlagUseNumbersZeroValues) && val == 0 {
			return "", nil
		}
		return strconv.FormatInt(val, 10), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val := fieldValue.Uint()
		if !flag.has(EncodingFlagUseNumbersZeroValues) && val == 0 {
			return "", nil
		}
		return strconv.FormatUint(val, 10), nil

	case reflect.Float32:
		val := fieldValue.Float()
		if !flag.has(EncodingFlagUseNumbersZeroValues) && val == 0 {
			return "", nil
		}
		// Use -1 for precision to format the smallest number of digits necessary
		return strconv.FormatFloat(val, 'f', -1, 32), nil
	case reflect.Float64:
		val := fieldValue.Float()
		if !flag.has(EncodingFlagUseNumbersZeroValues) && val == 0 {
			return "", nil
		}
		return strconv.FormatFloat(val, 'f', -1, 64), nil

	case reflect.Struct:
		// Handle time.Time type
		switch fieldValue.Type() {
		case reflect.TypeOf(time.Time{}):
			return formatTimeWithTagFormat(fieldValue, flag, tag)
		default:
			return "", fmt.Errorf("unsupported struct type: %s", fieldValue.Type())
		}

	// Default: use Sprintf for unsupported types
	default:
		if fieldValue.CanInterface() {
			return fmt.Sprintf("%v", fieldValue.Interface()), nil
		}
		return "", fmt.Errorf("unsupported type (%s)", fieldValue.Kind())
	}
}

// formatTimeWithTagFormat formats a time.Time value with the specified format
// and timezone conversion if specified in the tag.
func formatTimeWithTagFormat(fieldValue reflect.Value, flag EncodingFlag, tag *reflect.StructTag) (string, error) {
	// Check if the value is zero and the flag is not set to use numbers zero values
	if !flag.has(EncodingFlagUseNumbersZeroValues) && fieldValue.IsZero() {
		return "", nil
	}

	t, ok := fieldValue.Interface().(time.Time)
	if !ok {
		return "", fmt.Errorf("expected time.Time value but got %v", fieldValue.Type())
	}

	// If no tag is specified, use the default format
	if tag == nil {
		return t.Format(DefaultTimeFormat), nil
	}

	// Handle timezone conversion if specified in tag
	if tz, ok := tag.Lookup("tz"); ok {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return "", fmt.Errorf("invalid timezone %s: %w", tz, err)
		}
		t = t.In(loc)
	} else {
		loc, err := time.LoadLocation(DefaultTimezone)
		if err != nil {
			return "", fmt.Errorf("invalid default timezone %s: %w", DefaultTimezone, err)
		}
		t = t.In(loc)
	}

	var result string
	if timeFormat, ok := tag.Lookup("format"); ok {
		switch timeFormat {
		case "unix":
			result = strconv.FormatInt(t.Unix(), 10)
		case "unixnano":
			result = strconv.FormatInt(t.UnixNano(), 10)
		case "unixmilli":
			result = strconv.FormatInt(t.UnixMilli(), 10)
		case "unixmicro":
			result = strconv.FormatInt(t.UnixMicro(), 10)
		case "rfc3339":
			result = t.Format(time.RFC3339)
		case "rfc3339nano":
			result = t.Format(time.RFC3339Nano)
		case "rfc822":
			result = t.Format(time.RFC822)
		case "rfc822z":
			result = t.Format(time.RFC822Z)
		case "rfc850":
			result = t.Format(time.RFC850)
		case "rfc1123":
			result = t.Format(time.RFC1123)
		case "rfc1123z":
			result = t.Format(time.RFC1123Z)
		case "kitchen":
			result = t.Format(time.Stamp)
		case "stampmilli":
			result = t.Format(time.StampMilli)
		case "stampmicro":
			result = t.Format(time.StampMicro)
		case "stampnano":
			result = t.Format(time.StampNano)
		default:
			return "", fmt.Errorf("unsupported time format: %s", timeFormat)
		}
	} else {
		result = t.Format(time.RFC3339)
	}

	return result, nil
}
