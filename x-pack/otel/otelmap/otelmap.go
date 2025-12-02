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
func ConvertNonPrimitive[T mapstrOrMap](m T) {
	for key, val := range m {
		switch x := val.(type) {
		case mapstr.M:
			ConvertNonPrimitive(x)
			m[key] = map[string]any(x)
		case []mapstr.M:
			s := make([]any, len(x))
			for i, val := range x {
				ConvertNonPrimitive(val)
				s[i] = map[string]any(val)
			}
			m[key] = s
		case map[string]any:
			ConvertNonPrimitive(x)
			m[key] = x
		case []map[string]any:
			s := make([]any, len(x))
			for i := range x {
				ConvertNonPrimitive(x[i])
				s[i] = x[i]
			}
			m[key] = s
		case time.Time:
			m[key] = x.UTC().Format("2006-01-02T15:04:05.000Z")
		case common.Time:
			m[key] = time.Time(x).UTC().Format("2006-01-02T15:04:05.000Z")
		case []time.Time:
			s := make([]any, 0, len(x))
			for _, i := range x {
				s = append(s, i.UTC().Format("2006-01-02T15:04:05.000Z"))
			}
			m[key] = s
		case []common.Time:
			s := make([]any, 0, len(x))
			for _, i := range x {
				s = append(s, time.Time(i).UTC().Format("2006-01-02T15:04:05.000Z"))
			}
			m[key] = s
		case encoding.TextMarshaler:
			text, err := x.MarshalText()
			if err != nil {
				m[key] = fmt.Sprintf("error converting %T to string: %s", x, err)
				continue
			}
			m[key] = string(text)
		case []bool, []string, []float32, []float64, []int, []int8, []int16, []int32, []int64,
			[]uint, []uint8, []uint16, []uint32, []uint64:
			ref := reflect.ValueOf(x)
			s := make([]any, ref.Len())
			for i := 0; i < ref.Len(); i++ {
				s[i] = ref.Index(i).Interface()
			}
			m[key] = s
		case nil, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		default:
			ref := reflect.ValueOf(x)
			if ref.Kind() == reflect.Struct {
				var im map[string]any
				err := marshalUnmarshal(x, &im)
				if err != nil {
					m[key] = fmt.Sprintf("error encoding struct to map: %s", err)
					continue
				}
				ConvertNonPrimitive(im)
				m[key] = im
				break
			}
			if ref.Kind() == reflect.Slice || ref.Kind() == reflect.Array {
				s := make([]any, ref.Len())
				for i := 0; i < ref.Len(); i++ {
					elem := ref.Index(i).Interface()
					if mi, ok := elem.(map[string]any); ok {
						ConvertNonPrimitive(mi)
						s[i] = mi
					} else if mi, ok := elem.(mapstr.M); ok {
						ConvertNonPrimitive(mi)
						s[i] = map[string]any(mi)
					} else {
						s[i] = elem
					}
				}
				m[key] = s
				break // we figured out the type, so we don't need the unknown type case
			}
			m[key] = fmt.Sprintf("unknown type: %T", x)
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
