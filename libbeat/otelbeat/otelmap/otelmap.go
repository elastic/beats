// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package otelmap provides utilities for converting between beats and otel map types.
package otelmap

import (
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// ToMapstr converts a [pcommon.Map] to a [mapstr.M].
func ToMapstr(m pcommon.Map) mapstr.M {
	return m.AsRaw()
}

// FromMapstr converts a [mapstr.M] to a [pcommon.Map].
func FromMapstr(m mapstr.M) pcommon.Map {
	out := pcommon.NewMap()
	for k, v := range m {
		switch x := v.(type) {
		case string:
			out.PutStr(k, x)
		case []string:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]string) {
				newVal := dest.AppendEmpty()
				newVal.SetStr(i)
			}
		case int:
			out.PutInt(k, int64(v.(int)))
		case []int:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case int8:
			out.PutInt(k, int64(v.(int8)))
		case []int8:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int8) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case int16:
			out.PutInt(k, int64(v.(int16)))
		case []int16:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int16) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case int32:
			out.PutInt(k, int64(v.(int32)))
		case []int32:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int32) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case int64:
			out.PutInt(k, v.(int64))
		case []int64:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int64) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(i)
			}
		case uint:
			out.PutInt(k, int64(v.(uint)))
		case []uint:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case uint8:
			out.PutInt(k, int64(v.(uint8)))
		case []uint8:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint8) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case uint16:
			out.PutInt(k, int64(v.(uint16)))
		case []uint16:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint16) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case uint32:
			out.PutInt(k, int64(v.(uint32)))
		case []uint32:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint32) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case uint64:
			out.PutInt(k, int64(v.(uint64)))
		case []uint64:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint64) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case float32:
			out.PutDouble(k, float64(v.(float32)))
		case []float32:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]float32) {
				newVal := dest.AppendEmpty()
				newVal.SetDouble(float64(i))
			}
		case float64:
			out.PutDouble(k, v.(float64))
		case []float64:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]float64) {
				newVal := dest.AppendEmpty()
				newVal.SetDouble(i)
			}
		case bool:
			out.PutBool(k, x)
		case []bool:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]bool) {
				newVal := dest.AppendEmpty()
				newVal.SetBool(i)
			}
		case mapstr.M:
			dest := out.PutEmptyMap(k)
			newMap := FromMapstr(x)
			newMap.CopyTo(dest)
		case []mapstr.M:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]mapstr.M) {
				newVal := dest.AppendEmpty()
				newMap := FromMapstr(i)
				newMap.CopyTo(newVal.SetEmptyMap())
			}
		case time.Time:
			out.PutStr(k, x.UTC().Format("2006-01-02T15:04:05.000Z"))
		case []time.Time:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]time.Time) {
				newVal := dest.AppendEmpty()
				newVal.SetStr(i.UTC().Format("2006-01-02T15:04:05.000Z"))
			}
		case []any:
			dest := out.PutEmptySlice(k)
			convertValue(v.([]interface{}), dest)
		default:
			out.PutStr(k, fmt.Sprintf("unknown type: %T", x))
		}
	}
	return out
}

// convertValue converts a slice of any[] to pcommon.Value
func convertValue(v []any, dest pcommon.Slice) {
	// Handling the most common types without reflect is a small perf win.
	for _, i := range v {
		newValue := dest.AppendEmpty()
		switch val := i.(type) {
		case bool:
			newValue.SetBool(val)
		case string:
			newValue.SetStr(val)
		case int:
			newValue.SetInt(int64(val))
		case int8:
			newValue.SetInt(int64(val))
		case int16:
			newValue.SetInt(int64(val))
		case int32:
			newValue.SetInt(int64(val))
		case int64:
			newValue.SetInt(val)
		case uint:
			newValue.SetInt(int64(val))
		case uint8:
			newValue.SetInt(int64(val))
		case uint16:
			newValue.SetInt(int64(val))
		case uint32:
			newValue.SetInt(int64(val))
		case uint64:
			newValue.SetInt(int64(val))
		case float32:
			newValue.SetDouble(float64(val))
		case float64:
			newValue.SetDouble(val)
		case time.Time:
			newValue.SetStr(val.UTC().Format("2006-01-02T15:04:05.000Z"))
		default:
			newValue.SetStr(fmt.Sprintf("unknown type: %T", val))
		}

	}
}
