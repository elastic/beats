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

package otelmap

import (
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestFromMapstrMatchesLegacy(t *testing.T) {
	for _, tc := range benchmarkCases() {
		t.Run(tc.name, func(t *testing.T) {
			legacyClone := tc.src.Clone()
			legacyConvertNonPrimitive(legacyClone)
			legacyMap := pcommon.NewMap()
			require.NoError(t, legacyMap.FromRaw(map[string]any(legacyClone)))

			directMap := pcommon.NewMap()
			require.NoError(t, FromMapstr(directMap, tc.src))

			assert.Equal(t, legacyMap.AsRaw(), directMap.AsRaw())
		})
	}
}

func legacyConvertNonPrimitive[T mapstrOrMap](m T) {
	for key, val := range m {
		switch x := val.(type) {
		case mapstr.M:
			legacyConvertNonPrimitive(x)
			m[key] = map[string]any(x)
		case []mapstr.M:
			s := make([]any, len(x))
			for i, val := range x {
				legacyConvertNonPrimitive(val)
				s[i] = map[string]any(val)
			}
			m[key] = s
		case map[string]any:
			legacyConvertNonPrimitive(x)
			m[key] = x
		case []map[string]any:
			s := make([]any, len(x))
			for i := range x {
				legacyConvertNonPrimitive(x[i])
				s[i] = x[i]
			}
			m[key] = s
		case time.Time:
			m[key] = x.UTC().Format("2006-01-02T15:04:05.000Z")
		case common.Time:
			m[key] = time.Time(x).UTC().Format("2006-01-02T15:04:05.000Z")
		case time.Duration:
			m[key] = int64(x)
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
		case []time.Duration:
			s := make([]any, len(x))
			for i, d := range x {
				s[i] = int64(d)
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
		case nil, string, int, int8, int16, int32, int64, uint8, uint16, uint32, float32, float64, bool:
			// FromRaw handles these primitives directly.
		case uint:
			m[key] = int64(uint64(x) & uint64(math.MaxInt64))
		case uint64:
			m[key] = int64(x & uint64(math.MaxInt64))
		default:
			ref := reflect.ValueOf(x)
			switch ref.Kind() {
			case reflect.String:
				m[key] = ref.String()
			case reflect.Bool:
				m[key] = ref.Bool()
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				m[key] = ref.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				m[key] = int64(ref.Uint() & uint64(math.MaxInt64))
			case reflect.Float32, reflect.Float64:
				m[key] = ref.Float()
			case reflect.Struct:
				var im map[string]any
				marshaled, err := json.Marshal(x)
				if err != nil {
					m[key] = fmt.Sprintf("error encoding struct to map: %s", err)
					continue
				}
				if err := json.Unmarshal(marshaled, &im); err != nil {
					m[key] = fmt.Sprintf("error encoding struct to map: %s", err)
					continue
				}
				legacyConvertNonPrimitive(im)
				m[key] = im
			case reflect.Slice, reflect.Array:
				s := make([]any, ref.Len())
				for i := 0; i < ref.Len(); i++ {
					elem := ref.Index(i).Interface()
					if mi, ok := elem.(map[string]any); ok {
						legacyConvertNonPrimitive(mi)
						s[i] = mi
					} else if mi, ok := elem.(mapstr.M); ok {
						legacyConvertNonPrimitive(mi)
						s[i] = map[string]any(mi)
					} else {
						s[i] = elem
					}
				}
				m[key] = s
			case reflect.Pointer, reflect.Interface:
				if ref.IsNil() {
					m[key] = nil
					continue
				}
				wrapper := map[string]any{key: ref.Elem().Interface()}
				legacyConvertNonPrimitive(wrapper)
				m[key] = wrapper[key]
			default:
				m[key] = fmt.Sprintf("unknown type: %T", x)
			}
		}
	}
}
