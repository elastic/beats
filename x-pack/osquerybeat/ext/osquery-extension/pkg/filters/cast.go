// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"reflect"
	"strconv"
)

func ToBool(input any) bool {
	v := reflect.ValueOf(input)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return false
	}

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0
	case reflect.String:
		s := v.String()
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i != 0
		}
		if u, err := strconv.ParseUint(s, 10, 64); err == nil {
			return u != 0
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f != 0
		}
	}
	return false
}

func ToInt64(input any) int64 {
	v := reflect.ValueOf(input)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return 0
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int64(v.Uint())
	case reflect.Float32, reflect.Float64:
		return int64(v.Float())
	case reflect.Bool:
		if v.Bool() {
			return 1
		}
		return 0
	case reflect.String:
		s := v.String()
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if u, err := strconv.ParseUint(s, 10, 64); err == nil {
			return int64(u)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f)
		}
	}
	return 0
}

func ToUint64(input any) uint64 {
	v := reflect.ValueOf(input)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return 0
	}

	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := v.Int()
		if i < 0 {
			return 0
		}
		return uint64(i)
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if f < 0 {
			return 0
		}
		return uint64(f)
	case reflect.Bool:
		if v.Bool() {
			return 1
		}
		return 0
	case reflect.String:
		s := v.String()
		if u, err := strconv.ParseUint(s, 10, 64); err == nil {
			return u
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			if i < 0 {
				return 0
			}
			return uint64(i)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			if f < 0 {
				return 0
			}
			return uint64(f)
		}
	}
	return 0
}

func ToFloat64(input any) float64 {
	v := reflect.ValueOf(input)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return 0
	}

	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(v.Uint())
	case reflect.Bool:
		if v.Bool() {
			return 1
		}
		return 0
	case reflect.String:
		s := v.String()
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return float64(i)
		}
		if u, err := strconv.ParseUint(s, 10, 64); err == nil {
			return float64(u)
		}
	}
	return 0
}