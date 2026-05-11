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
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestFromMapstrTime(t *testing.T) {
	tests := []struct {
		mapstr_val  string
		pcommon_val string
	}{
		{mapstr_val: "2006-01-02T15:04:05+07:00", pcommon_val: "2006-01-02T08:04:05.000Z"},
		{mapstr_val: "1970-01-01T00:00:00+00:00", pcommon_val: "1970-01-01T00:00:00.000Z"},
	}
	for _, tc := range tests {
		origTime, err := time.Parse(time.RFC3339, tc.mapstr_val)
		require.NoError(t, err)
		dst := pcommon.NewMap()
		require.NoError(t, FromMapstr(dst, mapstr.M{"test": origTime}))
		assert.Equal(t, map[string]any{"test": tc.pcommon_val}, dst.AsRaw())
	}
}

func TestFromMapstrCommonTime(t *testing.T) {
	tests := []struct {
		mapstr_val  string
		pcommon_val string
	}{
		{mapstr_val: "2006-01-02T15:04:05+07:00", pcommon_val: "2006-01-02T08:04:05.000Z"},
		{mapstr_val: "1970-01-01T00:00:00+00:00", pcommon_val: "1970-01-01T00:00:00.000Z"},
	}
	for _, tc := range tests {
		origTime, err := time.Parse(time.RFC3339, tc.mapstr_val)
		require.NoError(t, err)
		dst := pcommon.NewMap()
		require.NoError(t, FromMapstr(dst, mapstr.M{"test": common.Time(origTime)}))
		assert.Equal(t, map[string]any{"test": tc.pcommon_val}, dst.AsRaw())
	}
}

func TestFromMapstrDuration(t *testing.T) {
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{"duration": 1500 * time.Millisecond}))
	assert.Equal(t, map[string]any{"duration": int64(1500 * time.Millisecond)}, dst.AsRaw())
}

func TestFromMapstrSliceDuration(t *testing.T) {
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{"durations": []time.Duration{1500 * time.Millisecond, 2 * time.Second}}))
	assert.Equal(t, map[string]any{"durations": []any{int64(1500 * time.Millisecond), int64(2 * time.Second)}}, dst.AsRaw())
}

func TestFromMapstrString(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  interface{}
		pcommon_val string
	}{
		"forty two": {mapstr_val: "forty two", pcommon_val: "forty two"},
		"empty":     {mapstr_val: "", pcommon_val: ""},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dst := pcommon.NewMap()
			require.NoError(t, FromMapstr(dst, mapstr.M{"test": tc.mapstr_val}))
			assert.Equal(t, map[string]any{"test": tc.pcommon_val}, dst.AsRaw())
		})
	}
}

func TestFromMapstrSliceString(t *testing.T) {
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{"slice": []string{"1", "2", "3"}}))
	assert.Equal(t, map[string]any{"slice": []any{"1", "2", "3"}}, dst.AsRaw())
}

func TestFromMapstrSliceInt(t *testing.T) {
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{"slice": []int{42, 43, 44}}))
	assert.Equal(t, map[string]any{"slice": []any{int64(42), int64(43), int64(44)}}, dst.AsRaw())
}

func TestFromMapstrSliceAny(t *testing.T) {
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{"slice": []any{42, "forty-three", true}}))
	assert.Equal(t, map[string]any{"slice": []any{int64(42), "forty-three", true}}, dst.AsRaw())
}

func TestFromMapstrSliceDouble(t *testing.T) {
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{"slice": []float32{4.2, 4.3, 4.4}}))
	want := []any{float64(float32(4.2)), float64(float32(4.3)), float64(float32(4.4))}
	assert.Equal(t, map[string]any{"slice": want}, dst.AsRaw())
}

func TestFromMapstrBool(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  interface{}
		pcommon_val bool
	}{
		"true":  {mapstr_val: true, pcommon_val: true},
		"false": {mapstr_val: false, pcommon_val: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dst := pcommon.NewMap()
			require.NoError(t, FromMapstr(dst, mapstr.M{"test": tc.mapstr_val}))
			assert.Equal(t, map[string]any{"test": tc.pcommon_val}, dst.AsRaw())
		})
	}
}

func TestFromMapstrSliceBool(t *testing.T) {
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{"slice": []bool{true, false, true}}))
	assert.Equal(t, map[string]any{"slice": []any{true, false, true}}, dst.AsRaw())
}

func TestFromMapstrMapstr(t *testing.T) {
	input := mapstr.M{
		"inner": mapstr.M{
			"inner_int":          42,
			"inner_string_slice": []string{"string"},
		},
	}
	want := map[string]any{
		"inner": map[string]any{
			"inner_int":          int64(42),
			"inner_string_slice": []any{"string"},
		},
	}
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, input))
	assert.Equal(t, want, dst.AsRaw())
}

func TestFromMapstrSliceMapstr(t *testing.T) {
	input := mapstr.M{
		"slice": []mapstr.M{{"item": 1}, {"item": 1}, {"item": 1}},
	}
	want := map[string]any{
		"slice": []any{
			map[string]any{"item": int64(1)},
			map[string]any{"item": int64(1)},
			map[string]any{"item": int64(1)},
		},
	}
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, input))
	assert.Equal(t, want, dst.AsRaw())
}

func TestFromMapstrSliceTime(t *testing.T) {
	times := []struct {
		mapstr_val  string
		pcommon_val string
	}{
		{mapstr_val: "2006-01-02T15:04:05+07:00", pcommon_val: "2006-01-02T08:04:05.000Z"},
		{mapstr_val: "1970-01-01T00:00:00+00:00", pcommon_val: "1970-01-01T00:00:00.000Z"},
	}
	var sliceTimes []time.Time
	var sliceTimesStr []any
	for _, tc := range times {
		targetTime, err := time.Parse(time.RFC3339, tc.mapstr_val)
		require.NoError(t, err)
		sliceTimes = append(sliceTimes, targetTime)
		sliceTimesStr = append(sliceTimesStr, tc.pcommon_val)
	}
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{"slice": sliceTimes}))
	assert.Equal(t, map[string]any{"slice": sliceTimesStr}, dst.AsRaw())
}

func TestFromMapstrSliceCommonTime(t *testing.T) {
	times := []struct {
		mapstr_val  string
		pcommon_val string
	}{
		{mapstr_val: "2006-01-02T15:04:05+07:00", pcommon_val: "2006-01-02T08:04:05.000Z"},
		{mapstr_val: "1970-01-01T00:00:00+00:00", pcommon_val: "1970-01-01T00:00:00.000Z"},
	}
	var sliceTimes []common.Time
	var sliceTimesStr []any
	for _, tc := range times {
		targetTime, err := time.Parse(time.RFC3339, tc.mapstr_val)
		require.NoError(t, err)
		sliceTimes = append(sliceTimes, common.Time(targetTime))
		sliceTimesStr = append(sliceTimesStr, tc.pcommon_val)
	}
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{"slice": sliceTimes}))
	assert.Equal(t, map[string]any{"slice": sliceTimesStr}, dst.AsRaw())
}

type structWithTextMarshaler struct {
	Value string `json:"value"`
}

func (s *structWithTextMarshaler) MarshalText() ([]byte, error) {
	return []byte("marshalled:" + s.Value), nil
}

func TestFromMapstrWithNestedData(t *testing.T) {
	input := mapstr.M{
		"any_array":  [3]any{1, "string", 3},
		"any_slice":  []any{5.1, 6.2},
		"bool_array": [2]bool{true, false},
		"bool_slice": []bool{false, true},
		"struct": struct {
			Value string `json:"value"`
		}{
			Value: "string",
		},
		"struct_with_text_marshaler": &structWithTextMarshaler{
			Value: "string",
		},
		"inner": []mapstr.M{
			{
				"inner_int":       42,
				"inner_map_slice": [1]any{nil},
				"inner_slice": []map[string]any{
					{"string": "string"},
					{"number": 12.3},
				},
				"inner_struct": struct {
					Value string `json:"value"`
				}{
					Value: "string",
				},
				"inner_struct_with_text_marshaler": &structWithTextMarshaler{
					Value: "string",
				},
			},
			{
				"inner_int": 43,
				"inner_map_slice": []any{
					map[string]any{"string": "string3"},
					mapstr.M{"number": 12.4},
				},
				"inner_slice": [2]map[string]any{
					{"string": "string2"},
					{"number": 12.4},
				},
			},
		},
	}
	want := map[string]any{
		"any_array":  []any{int64(1), "string", int64(3)},
		"any_slice":  []any{5.1, 6.2},
		"bool_array": []any{true, false},
		"bool_slice": []any{false, true},
		"struct": map[string]any{
			"value": "string",
		},
		"struct_with_text_marshaler": "marshalled:string",
		"inner": []any{
			map[string]any{
				"inner_int":       int64(42),
				"inner_map_slice": []any{nil},
				"inner_slice": []any{
					map[string]any{"string": "string"},
					map[string]any{"number": 12.3},
				},
				"inner_struct": map[string]any{
					"value": "string",
				},
				"inner_struct_with_text_marshaler": "marshalled:string",
			},
			map[string]any{
				"inner_int": int64(43),
				"inner_map_slice": []any{
					map[string]any{"string": "string3"},
					map[string]any{"number": 12.4},
				},
				"inner_slice": []any{
					map[string]any{"string": "string2"},
					map[string]any{"number": 12.4},
				},
			},
		},
	}
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, input))
	assert.Equal(t, want, dst.AsRaw())
}

// TestFromMapstrMasksLargeUnsignedInts pins down the masking behavior for
// uint values that exceed math.MaxInt64. pcommon.Value has no unsigned slot,
// so we clear bit 63 instead of letting the conversion wrap to a negative.
func TestFromMapstrMasksLargeUnsignedInts(t *testing.T) {
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{
		"max_uint64":    ^uint64(0),
		"max_uint":      ^uint(0),
		"uint64_values": []uint64{0, 1, ^uint64(0)},
	}))
	assert.Equal(t, map[string]any{
		"max_uint64":    int64(math.MaxInt64),
		"max_uint":      int64(math.MaxInt64),
		"uint64_values": []any{int64(0), int64(1), int64(math.MaxInt64)},
	}, dst.AsRaw())
}

func TestToMapstr(t *testing.T) {
	pm := pcommon.NewMap()
	pm.PutInt("int", 42)
	pm.PutDouble("float", 4.2)
	pm.PutStr("string", "forty two")

	s := pm.PutEmptySlice("slice")
	s.AppendEmpty().SetInt(1)
	s.AppendEmpty().SetInt(2)

	m := pm.PutEmptyMap("map")
	m.PutInt("int", 42)

	want := mapstr.M{
		"int":    int64(42),
		"float":  4.2,
		"string": "forty two",
		"slice": []interface{}{
			int64(1),
			int64(2),
		},
		"map": map[string]interface{}{
			"int": int64(42),
		},
	}

	got := ToMapstr(pm)
	assert.Equal(t, want, got)
}

// TestFromMapstrNamedPrimitives covers values whose static type is a defined
// type with a primitive underlying kind (e.g. type Severity string,
// json.Number). The typed switch in FromValue matches on exact static types,
// so these names fall through to fromReflective; that path must still encode
// them as their underlying primitive instead of "unknown type".
func TestFromMapstrNamedPrimitives(t *testing.T) {
	type Severity string
	type Method string
	type Port uint16
	type Score float64
	type Count int32
	type Enabled bool

	input := mapstr.M{
		"severity":    Severity("warn"),
		"method":      Method("GET"),
		"json_number": json.Number("9223372036854775808"),
		"port":        Port(8080),
		"score":       Score(0.75),
		"count":       Count(-7),
		"enabled":     Enabled(true),
	}
	want := map[string]any{
		"severity":    "warn",
		"method":      "GET",
		"json_number": "9223372036854775808",
		"port":        int64(8080),
		"score":       0.75,
		"count":       int64(-7),
		"enabled":     true,
	}

	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, input))
	assert.Equal(t, want, dst.AsRaw())
}

func TestFromMapstrNamedPrimitivesInSlice(t *testing.T) {
	type Severity string
	type Port uint16

	input := mapstr.M{
		"severities": []Severity{"info", "warn", "error"},
		"ports":      []Port{80, 443, 8080},
	}
	want := map[string]any{
		"severities": []any{"info", "warn", "error"},
		"ports":      []any{int64(80), int64(443), int64(8080)},
	}

	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, input))
	assert.Equal(t, want, dst.AsRaw())
}

func TestUnknownType(t *testing.T) {
	dst := pcommon.NewMap()
	require.NoError(t, FromMapstr(dst, mapstr.M{
		"unknown_map": map[string]int{"key": 42},
	}))
	assert.Equal(t, map[string]any{"unknown_map": "unknown type: map[string]int"}, dst.AsRaw())
}
