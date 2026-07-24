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
	"strings"
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
		if err := putIntoMap(key, value, dst); err != nil {
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
		setFloat32Value(dst, v)
		return nil
	case float64:
		setFloat64Value(dst, v)
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
	case common.NetString:
		// go-structform (Beats ES output path) encodes NetString as raw []byte,
		// producing a JSON integer array. Match that behavior for document shape
		// parity across both output paths.
		return fromUnsignedSlice(dst.SetEmptySlice(), []byte(v))
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
		return fromFloat32Slice(dst.SetEmptySlice(), v)
	case []float64:
		return fromFloat64Slice(dst.SetEmptySlice(), v)
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

// putIntoMap encodes val under key in dst using the typed Put* methods on
// pcommon.Map (PutStr, PutInt, PutBool, …) rather than the PutEmpty+Set*
// two-step. PutEmpty allocates an empty AnyValue slot and Set* allocates
// the typed wrapper, two allocations per field. The typed Put* methods fold
// both into one.
func putIntoMap(key string, val any, dst pcommon.Map) error {
	switch v := val.(type) {
	case nil:
		dst.PutEmpty(key)
		return nil
	case string:
		dst.PutStr(key, v)
	case bool:
		dst.PutBool(key, v)
	case int:
		dst.PutInt(key, int64(v))
	case int8:
		dst.PutInt(key, int64(v))
	case int16:
		dst.PutInt(key, int64(v))
	case int32:
		dst.PutInt(key, int64(v))
	case int64:
		dst.PutInt(key, v)
	case uint:
		dst.PutInt(key, maskUnsignedInt(uint64(v)))
	case uint8:
		dst.PutInt(key, int64(v))
	case uint16:
		dst.PutInt(key, int64(v))
	case uint32:
		dst.PutInt(key, int64(v))
	case uint64:
		dst.PutInt(key, maskUnsignedInt(v))
	case float32:
		setFloat32Value(dst.PutEmpty(key), v)
	case float64:
		setFloat64Value(dst.PutEmpty(key), v)
	case mapstr.M:
		return FromMapstr(dst.PutEmptyMap(key), v)
	case map[string]any:
		return FromMapstr(dst.PutEmptyMap(key), v)
	default:
		return FromValue(dst.PutEmpty(key), val)
	}
	return nil
}

// MergeMapstrIntoPdata deep-merges src into dst, equivalent to
// mapstr.M.DeepUpdate for pcommon.Map. For map-typed values, existing map
// entries in dst are recursed into rather than replaced. All other values are
// encoded via [putIntoMap]. Overwrite controls whether non-map values in
// dst are replaced when keys collide.
func MergeMapstrIntoPdata(src mapstr.M, dst pcommon.Map, overwrite bool) error {
	for key, val := range src {
		if err := mergeVal(key, val, dst, overwrite); err != nil {
			return err
		}
	}
	return nil
}

func mergeVal(key string, val any, dst pcommon.Map, overwrite bool) error {
	var m mapstr.M
	switch x := val.(type) {
	case mapstr.M:
		m = x
	case map[string]any:
		m = mapstr.M(x)
	}
	if m != nil {
		if existing, ok := dst.Get(key); ok && existing.Type() == pcommon.ValueTypeMap {
			// Both sides are maps: recurse, respecting overwrite.
			return MergeMapstrIntoPdata(m, existing.Map(), overwrite)
		}
		// src is a map and dst key is absent or a non-map scalar: always write the map.
		// This matches mapstr.deepUpdateValue which replaces non-map values with maps
		// regardless of the overwrite flag.
		return FromMapstr(dst.PutEmptyMap(key), m)
	}
	if _, ok := dst.Get(key); ok && !overwrite {
		return nil
	}
	return putIntoMap(key, val, dst)
}

// PdataValuesMap wraps a pcommon.Map to satisfy the conditions.ValuesMap interface,
// allowing condition evaluation directly on a log record body without a ToMapstr call.
type PdataValuesMap struct {
	M pcommon.Map
}

// GetValue retrieves the value at the given dotted key path and returns it as
// a Go primitive (string, int64, float64, bool, []interface{}, or map[string]interface{}).
func (p PdataValuesMap) GetValue(key string) (any, error) {
	v, ok := GetAtPath(key, p.M)
	if !ok {
		return nil, mapstr.ErrKeyNotFound
	}
	return v.AsRaw(), nil
}

// GetAtPath retrieves the value at a dotted key path (e.g. "cloud.instance.id")
// from m, traversing nested maps as needed.
// For keys that contain dots, it tries the full key as a literal name first
// (matching mapstr.M.GetValue behaviour for keys stored flat), then falls back
// to path navigation.
func GetAtPath(key string, m pcommon.Map) (pcommon.Value, bool) {
	before, after, ok := strings.Cut(key, ".")
	if !ok {
		return m.Get(key)
	}
	if v, found := m.Get(key); found {
		return v, true
	}
	parent, ok := m.Get(before)
	if !ok || parent.Type() != pcommon.ValueTypeMap {
		return pcommon.Value{}, false
	}
	return GetAtPath(after, parent.Map())
}

// PutAtPath encodes val at a dotted key path (e.g. "cloud.instance.id") in m,
// creating intermediate maps as needed. Existing intermediate maps are
// preserved (not replaced).
func PutAtPath(key string, val any, m pcommon.Map) error {
	before, after, ok := strings.Cut(key, ".")
	if !ok {
		return putIntoMap(key, val, m)
	}
	head, rest := before, after
	if existing, ok := m.Get(head); ok && existing.Type() == pcommon.ValueTypeMap {
		return PutAtPath(rest, val, existing.Map())
	}
	return PutAtPath(rest, val, m.PutEmptyMap(head))
}

// DeleteAtPath removes the value at a dotted key path (e.g. "cloud.instance.id")
// from m, traversing nested maps as needed. Returns false if the path did not exist.
// For keys that contain dots, it tries the full key as a literal name first
// (matching mapstr.M.Delete behaviour for keys stored flat), then falls back
// to path navigation.
func DeleteAtPath(key string, m pcommon.Map) bool {
	before, after, ok := strings.Cut(key, ".")
	if !ok {
		return m.Remove(key)
	}
	if m.Remove(key) {
		return true
	}
	parent, ok := m.Get(before)
	if !ok || parent.Type() != pcommon.ValueTypeMap {
		return false
	}
	return DeleteAtPath(after, parent.Map())
}

// FlattenKeys returns all key paths in m as dotted strings (e.g. "cloud.instance.id"),
// including intermediate map keys. This mirrors the behaviour of mapstr.M.FlattenKeys:
// children are listed before their parent map key.
func FlattenKeys(m pcommon.Map) []string {
	out := make([]string, 0, m.Len())
	flattenPdataKeys("", m, &out)
	return out
}

func flattenPdataKeys(prefix string, m pcommon.Map, out *[]string) {
	for k, v := range m.All() {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}
		if v.Type() == pcommon.ValueTypeMap {
			flattenPdataKeys(fullKey, v.Map(), out)
		}
		*out = append(*out, fullKey)
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
	case reflect.Float32:
		setFloat32Value(dst, float32(ref.Float()))
		return nil
	case reflect.Float64:
		setFloat64Value(dst, ref.Float())
		return nil
	case reflect.Complex64, reflect.Complex128:
		dst.SetStr(fmt.Sprintf("%v", ref.Complex()))
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
	for i := range n {
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

func fromFloat32Slice(dst pcommon.Slice, src []float32) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		setFloat32Value(dst.AppendEmpty(), item)
	}
	return nil
}

func fromFloat64Slice(dst pcommon.Slice, src []float64) error {
	dst.EnsureCapacity(len(src))
	for _, item := range src {
		setFloat64Value(dst.AppendEmpty(), item)
	}
	return nil
}

// isFloat32WholeNumber reports whether f is a whole number that can be
// precisely represented as an int64. float32 has 24 bits of mantissa so
// integers outside [-2²⁴+1, 2²⁴-1] cannot be represented exactly.
func isFloat32WholeNumber(f float32) bool {
	const preciseMax float32 = 0x1p24 - 1
	_, frac := math.Modf(float64(f))
	return frac == 0 && -preciseMax <= f && f <= preciseMax
}

// isFloat64WholeNumber reports whether f is a whole number that can be
// precisely represented as an int64. float64 has 53 bits of mantissa so
// integers outside [-2⁵³+1, 2⁵³-1] cannot be represented exactly.
func isFloat64WholeNumber(f float64) bool {
	const preciseMax = 0x1p53 - 1
	_, frac := math.Modf(f)
	return frac == 0 && -preciseMax <= f && f <= preciseMax
}

// float32ToFloat64 widens a float32 to float64 by direct bit-pattern
// conversion. This matches the current Beats go-structform encoder (see
// OnFloat32 in elastic/go-structform), which also converts via float64(v)
// rather than round-tripping through a decimal string.
func float32ToFloat64(v float32) float64 {
	return float64(v)
}

func setFloat32Value(dst pcommon.Value, v float32) {
	if isFloat32WholeNumber(v) {
		dst.SetInt(int64(v))
		return
	}
	dst.SetDouble(float32ToFloat64(v))
}

func setFloat64Value(dst pcommon.Value, v float64) {
	if isFloat64WholeNumber(v) {
		dst.SetInt(int64(v))
		return
	}
	dst.SetDouble(v)
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
	return int64(value & uint64(math.MaxInt64))
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
