// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"reflect"
	"strconv"
)

// resolveValue resolves the value of an input interface{} to a reflect.Value.
func resolveValue(input any) (value reflect.Value, ok bool) {
	v := reflect.ValueOf(input)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return reflect.Value{}, false
	}
	return v, true
}

// ToBool converts an input value to a boolean.
func ToBool(input any) (result bool, ok bool) {
	v, ok := resolveValue(input)
	if !ok {
		return false, false
	}

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0, true
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0, true
	case reflect.String:
		s := v.String()
		if b, err := strconv.ParseBool(s); err == nil {
			return b, true
		}
		return false, false
	default:
		return false, false
	}
}

// ToInt64 converts an input value to an int64.
func ToInt64(input any) (result int64, ok bool) {
	v, ok := resolveValue(input)
	if !ok {
		return 0, false
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), true
	case reflect.Float32, reflect.Float64:
		return int64(v.Float()), true
	case reflect.String:
		s := v.String()
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// ToFloat64 converts an input value to a float64.
func ToFloat64(input any) (result float64, ok bool) {
	v, ok := resolveValue(input)
	if !ok {
		return 0, false
	}

	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.String:
		s := v.String()
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}
