// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package encoding

import (
	"fmt"
	"reflect"
	"strconv"
)

type EncodingFlag int

const (
	EncodingFlagParseUnexported EncodingFlag = 1 << iota
	EncodingFlagUseNumbers0Values
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
	result := make(map[string]string)

	if in == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

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

			value, err := convertValueToString(fieldValue, flags)
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

		if !flags.has(EncodingFlagParseUnexported) && !fieldType.IsExported() {
			continue
		}

		key := fieldType.Tag.Get("osquery")
		switch key {
		case "-":
			continue
		case "":
			key = fieldType.Name
		}

		value, err := convertValueToString(fieldValue, flags)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", key, err)
		}

		result[key] = value
	}

	return result, nil
}

func convertValueToString(fieldValue reflect.Value, flag EncodingFlag) (string, error) {
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
		if !flag.has(EncodingFlagUseNumbers0Values) && val == 0 {
			return "", nil
		}
		return strconv.FormatInt(val, 10), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val := fieldValue.Uint()
		if !flag.has(EncodingFlagUseNumbers0Values) && val == 0 {
			return "", nil
		}
		return strconv.FormatUint(val, 10), nil

	case reflect.Float32, reflect.Float64:
		// Use -1 for precision to format the smallest number of digits necessary
		return strconv.FormatFloat(fieldValue.Float(), 'f', -1, 64), nil

	// Default: use Sprintf for unsupported types
	default:
		if fieldValue.CanInterface() {
			return fmt.Sprintf("%v", fieldValue.Interface()), nil
		}
		return "", fmt.Errorf("unsupported type (%s)", fieldValue.Kind())
	}
}
