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
	"math"
	"reflect"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// otelTimestampLayout is the timestamp format the elasticsearchexporter
// expects for the @timestamp field when using the bodymap encoding.
const otelTimestampLayout = "2006-01-02T15:04:05.000Z"

type signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type floating interface {
	~float32 | ~float64
}

type mapstrOrMap interface {
	mapstr.M | map[string]any
}

// ToMapstr converts a [pcommon.Map] to a [mapstr.M].
func ToMapstr(m pcommon.Map) mapstr.M {
	return m.AsRaw()
}

// FromMapstr encodes src directly into dst as the inverse of [ToMapstr].
func FromMapstr[T mapstrOrMap](dst pcommon.Map, src T) error {
	dst.EnsureCapacity(len(src))
	for key, value := range src {
		if err := FromValue(dst.PutEmpty(key), value); err != nil {
			return err
		}
	}
	return nil
}

// FromValue encodes a single Go value into dst (the pdata-side sibling of
// [ToMapstr]).
func FromValue(dst pcommon.Value, value any) error {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		dst.SetStr(v)
		return nil
	case int:
		dst.SetInt(int64(v))
		return nil
	case int8:
		dst.SetInt(int64(v))
		return nil
	case int16:
		dst.SetInt(int64(v))
		return nil
	case int32:
		dst.SetInt(int64(v))
		return nil
	case int64:
		dst.SetInt(v)
		return nil
	case uint:
		dst.SetInt(maskUnsignedInt(uint64(v)))
		return nil
	case uint8:
		dst.SetInt(int64(v))
		return nil
	case uint16:
		dst.SetInt(int64(v))
		return nil
	case uint32:
		dst.SetInt(int64(v))
		return nil
	case uint64:
		dst.SetInt(maskUnsignedInt(v))
		return nil
	case float32:
		dst.SetDouble(float64(v))
		return nil
	case float64:
		dst.SetDouble(v)
		return nil
	case bool:
		dst.SetBool(v)
		return nil
	case mapstr.M:
		return FromMapstr(dst.SetEmptyMap(), v)
	case map[string]any:
		return FromMapstr(dst.SetEmptyMap(), v)
	case []mapstr.M:
		return fromMapSlice(dst.SetEmptySlice(), v)
	case []map[string]any:
		return fromMapSlice(dst.SetEmptySlice(), v)
	case []any:
		return fromAnySlice(dst.SetEmptySlice(), v)
	case time.Time:
		dst.SetStr(FormatTimestamp(v))
		return nil
	case common.Time:
		dst.SetStr(FormatTimestamp(time.Time(v)))
		return nil
	case time.Duration:
		dst.SetInt(int64(v))
		return nil
	case encoding.TextMarshaler:
		text, err := v.MarshalText()
		if err != nil {
			dst.SetStr(fmt.Sprintf("error converting %T to string: %s", v, err))
			return nil
		}
		dst.SetStr(string(text))
		return nil
	case []time.Time:
		return fromTimeSlice(dst.SetEmptySlice(), v)
	case []common.Time:
		return fromCommonTimeSlice(dst.SetEmptySlice(), v)
	case []string:
		return fromStringSlice(dst.SetEmptySlice(), v)
	case []bool:
		return fromBoolSlice(dst.SetEmptySlice(), v)
	case []float32:
		return fromFloatSlice(dst.SetEmptySlice(), v)
	case []float64:
		return fromFloatSlice(dst.SetEmptySlice(), v)
	case []int:
		return fromSignedSlice(dst.SetEmptySlice(), v)
	case []int8:
		return fromSignedSlice(dst.SetEmptySlice(), v)
	case []int16:
		return fromSignedSlice(dst.SetEmptySlice(), v)
	case []int32:
		return fromSignedSlice(dst.SetEmptySlice(), v)
	case []int64:
		return fromSignedSlice(dst.SetEmptySlice(), v)
	case []uint:
		return fromUnsignedSlice(dst.SetEmptySlice(), v)
	case []uint8:
		return fromUnsignedSlice(dst.SetEmptySlice(), v)
	case []uint16:
		return fromUnsignedSlice(dst.SetEmptySlice(), v)
	case []uint32:
		return fromUnsignedSlice(dst.SetEmptySlice(), v)
	case []uint64:
		return fromUnsignedSlice(dst.SetEmptySlice(), v)
	default:
		return fromReflective(dst, value)
	}
}

// FormatTimestamp renders t in the layout the elasticsearchexporter's
// bodymap encoding expects for @timestamp.
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format(otelTimestampLayout)
}

// fromReflective handles reflect values that don't match the typed switch in
// [FromValue].
func fromReflective(dst pcommon.Value, value any) error {
	ref := reflect.ValueOf(value)
	switch ref.Kind() {
	case reflect.String:
		dst.SetStr(ref.String())
		return nil
	case reflect.Bool:
		dst.SetBool(ref.Bool())
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst.SetInt(ref.Int())
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		dst.SetInt(maskUnsignedInt(ref.Uint()))
		return nil
	case reflect.Float32, reflect.Float64:
		dst.SetDouble(ref.Float())
		return nil
	case reflect.Struct:
		m, err := structToMap(value)
		if err != nil {
			dst.SetStr(fmt.Sprintf("error encoding struct to map: %s", err))
			return nil
		}
		return FromMapstr(dst.SetEmptyMap(), m)
	case reflect.Slice, reflect.Array:
		return fromReflectiveSlice(dst.SetEmptySlice(), ref)
	case reflect.Pointer, reflect.Interface:
		if ref.IsNil() {
			return nil
		}
		return FromValue(dst, ref.Elem().Interface())
	default:
		dst.SetStr(fmt.Sprintf("unknown type: %T", value))
		return nil
	}
}

func fromReflectiveSlice(dst pcommon.Slice, ref reflect.Value) error {
	n := ref.Len()
	dst.EnsureCapacity(n)
	for i := 0; i < n; i++ {
		if err := FromValue(dst.AppendEmpty(), ref.Index(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

func fromMapSlice[T mapstrOrMap](dst pcommon.Slice, src []T) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		if err := FromMapstr(dst.AppendEmpty().SetEmptyMap(), item); err != nil {
			return err
		}
	}
	return nil
}

func fromAnySlice(dst pcommon.Slice, src []any) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		if err := FromValue(dst.AppendEmpty(), item); err != nil {
			return err
		}
	}
	return nil
}

func fromTimeSlice(dst pcommon.Slice, src []time.Time) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		dst.AppendEmpty().SetStr(FormatTimestamp(item))
	}
	return nil
}

func fromCommonTimeSlice(dst pcommon.Slice, src []common.Time) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		dst.AppendEmpty().SetStr(FormatTimestamp(time.Time(item)))
	}
	return nil
}

func fromStringSlice(dst pcommon.Slice, src []string) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		dst.AppendEmpty().SetStr(item)
	}
	return nil
}

func fromBoolSlice(dst pcommon.Slice, src []bool) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		dst.AppendEmpty().SetBool(item)
	}
	return nil
}

func fromFloatSlice[T floating](dst pcommon.Slice, src []T) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		dst.AppendEmpty().SetDouble(float64(item))
	}
	return nil
}

func fromSignedSlice[T signed](dst pcommon.Slice, src []T) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		dst.AppendEmpty().SetInt(int64(item))
	}
	return nil
}

func fromUnsignedSlice[T unsigned](dst pcommon.Slice, src []T) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		dst.AppendEmpty().SetInt(maskUnsignedInt(uint64(item)))
	}
	return nil
}

// maskUnsignedInt converts a uint64 to int64 by clearing bit 63. pcommon.Value
// only carries a signed 64-bit integer, so values above math.MaxInt64 cannot
// be represented losslessly.
func maskUnsignedInt(value uint64) int64 {
	return int64(value & uint64(math.MaxInt64)) //nolint:gosec // mask clears bit 63, conversion is safe
}

// structToMap round-trips a struct value through JSON to obtain a
// map[string]any with the same field name semantics (json tags, casing) the
// caller would see if the struct were marshaled to JSON.
func structToMap(value any) (map[string]any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshalling to JSON: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil, fmt.Errorf("unmarshalling from JSON: %w", err)
	}
	return out, nil
}
