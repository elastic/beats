// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"reflect"
	"strings"
)

func convertStructArrayToMapArray[T any](input []T) []map[string]any {
	result := make([]map[string]any, len(input))
	for i, v := range input {
		result[i] = convertObjectToMap(v)
	}
	return result
}

func convertObjectToMap[T any](input T) map[string]any {
	return structToMap(reflect.ValueOf(input))
}

func structToMap(val reflect.Value) map[string]any {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	result := make(map[string]any)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)

		// Only exported fields
		if !fieldVal.CanInterface() {
			continue
		}

		// Get JSON tag
		tagName := parseJSONTag(fieldType.Tag.Get("json"))

		if tagName == "-" || tagName == "" {
			continue
		}

		// Recursively convert nested structs/slices/maps
		result[tagName] = convertValue(fieldVal)
	}

	return result
}

func convertValue(val reflect.Value) any {
	if !val.IsValid() {
		return nil
	}

	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		return convertValue(val.Elem())

	case reflect.Struct:
		return structToMap(val)

	case reflect.Slice, reflect.Array:
		slice := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			slice[i] = convertValue(val.Index(i))
		}
		return slice

	case reflect.Map:
		m := make(map[any]any)
		for _, key := range val.MapKeys() {
			m[key.Interface()] = convertValue(val.MapIndex(key))
		}
		return m

	default:
		return val.Interface()
	}
}

func isZeroValue(v reflect.Value) bool {
	// Treat zero/nil values as "empty"
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		// Check all fields are zero
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func parseJSONTag(tag string) string {
	if tag == "" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}
