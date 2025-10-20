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
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Allow ConvertNonPrimitive to be called recursively to handle nested maps of either type.
type mapstrOrMap interface {
	mapstr.M | map[string]any
}

// ToMapstr converts a [pcommon.Map] to a [mapstr.M].
func ToMapstr(m pcommon.Map) mapstr.M {
	return m.AsRaw()
}

// ConvertNonPrimitive handles the conversion of non-primitive data types to pcommon-primitive types.
// The conversion is performed in place.
// Notes:
//  1. Slices require special handling when converting a map[string]any to pcommon.Map.
//     The pcommon.Map expects all slices to be of type []any.
//     If you attempt to use other slice types (e.g., []string or []int),
//     pcommon.Map.FromRaw(...) will return an "invalid type" error.
//     To overcome this, we use "reflect" to transform []T into []any.
func ConvertNonPrimitive[T mapstrOrMap](src T, dst pcommon.Map) {
	dst.EnsureCapacity(len(src))
	for key, val := range src {
		switch x := val.(type) {
		case string:
			dst.PutStr(key, x)
		case bool:
			dst.PutBool(key, x)
		case int, int8, int16, int32, int64:
			dst.PutInt(key, reflect.ValueOf(x).Int())
		case uint, uint8, uint16, uint32, uint64:
			dst.PutInt(key, int64(reflect.ValueOf(x).Uint()))
		case float32, float64:
			dst.PutDouble(key, reflect.ValueOf(x).Convert(reflect.TypeOf(float64(0))).Float())
		case time.Time:
			dst.PutStr(key, x.UTC().Format("2006-01-02T15:04:05.000Z"))
		case common.Time:
			dst.PutStr(key, time.Time(x).UTC().Format("2006-01-02T15:04:05.000Z"))
		case mapstr.M:
			child := dst.PutEmptyMap(key)
			ConvertNonPrimitive(map[string]any(x), child)
		case map[string]any:
			child := dst.PutEmptyMap(key)
			ConvertNonPrimitive(x, child)
		case []mapstr.M:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, val := range x {
				child := s.AppendEmpty().SetEmptyMap()
				ConvertNonPrimitive(map[string]any(val), child)
			}
		case []map[string]any:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, val := range x {
				child := s.AppendEmpty().SetEmptyMap()
				ConvertNonPrimitive(val, child)
			}
		case []time.Time:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, i := range x {
				s.AppendEmpty().SetStr(i.UTC().Format("2006-01-02T15:04:05.000Z"))
			}
		case []common.Time:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, i := range x {
				s.AppendEmpty().SetStr(time.Time(i).UTC().Format("2006-01-02T15:04:05.000Z"))
			}
		case encoding.TextMarshaler:
			text, err := x.MarshalText()
			if err != nil {
				text = fmt.Appendf(nil, "error converting %T to string: %s", x, err)
			}
			dst.PutStr(key, string(text))
		// case []bool, []string, []float32, []float64, []int, []int8, []int16, []int32, []int64,
		// 	[]uint, []uint8, []uint16, []uint32, []uint64:
		// 	ref := reflect.ValueOf(x)
		// 	s := dst.PutEmptySlice(key)
		// 	s.EnsureCapacity(ref.Len())
		// 	for i := 0; i < ref.Len(); i++ {
		// 		s.AppendEmpty().FromRaw(ref.Index(i).Interface())
		// 	}
		case []string:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetStr(ele)
			}
		case []float32:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetDouble(float64(ele))
			}
		case []float64:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetDouble(ele)
			}
		case []int:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []int8:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []int16:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []int32:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []int64:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []uint:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []uint8:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []uint16:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []uint32:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []uint64:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetInt(int64(ele))
			}
		case []bool:
			s := dst.PutEmptySlice(key)
			s.EnsureCapacity(len(x))
			for _, ele := range x {
				s.AppendEmpty().SetBool(ele)
			}
		default:
			ref := reflect.ValueOf(x)
			if ref.Kind() == reflect.Struct {
				var im map[string]any
				err := marshalUnmarshal(x, &im)
				if err != nil {
					dst.PutStr(key, fmt.Sprintf("error encoding struct to map: %s", err))
					break
				}
				child := dst.PutEmptyMap(key)
				ConvertNonPrimitive(im, child)
				break
			}
			if ref.Kind() == reflect.Slice || ref.Kind() == reflect.Array {
				s := dst.PutEmptySlice(key)
				for i := 0; i < ref.Len(); i++ {
					elem := ref.Index(i).Interface()
					if mi, ok := elem.(map[string]any); ok {
						child := s.AppendEmpty().SetEmptyMap()
						ConvertNonPrimitive(mi, child)
					} else if mi, ok := elem.(mapstr.M); ok {
						child := s.AppendEmpty().SetEmptyMap()
						ConvertNonPrimitive(map[string]any(mi), child)
					} else {
						s.AppendEmpty().FromRaw(elem)
					}
				}
				break
			}
			dst.PutStr(key, fmt.Sprintf("unknown type: %T", x))
		}
	}
}

// marshalUnmarshal converts an interface to a mapstr.M by marshalling to JSON
// then unmarshalling the JSON object into a mapstr.M.
// Copied from libbeat/common/event.go
func marshalUnmarshal(in interface{}, out interface{}) error {
	// Decode and encode as JSON to normalize the types.
	marshaled, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("error marshalling to JSON: %w", err)
	}
	err = json.Unmarshal(marshaled, out)
	if err != nil {
		return fmt.Errorf("error unmarshalling from JSON: %w", err)
	}

	return nil
}
