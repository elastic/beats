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
	"math"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
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
		assert.NoError(t, err, "Error parsing time")
		a := mapstr.M{"test": origTime}
		want := mapstr.M{}
		want["test"] = tc.pcommon_val
		ConvertNonPrimitive(a)
		assert.Equal(t, want, a)
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
		assert.NoError(t, err, "Error parsing time")
		a := mapstr.M{"test": common.Time(origTime)}
		want := mapstr.M{}
		want["test"] = tc.pcommon_val
		ConvertNonPrimitive(a)
		assert.Equal(t, want, a)
	}
}

func TestFromMapstrDuration(t *testing.T) {
	input := mapstr.M{"duration": 1500 * time.Millisecond}
	want := mapstr.M{"duration": int64(1500 * time.Millisecond)}

	ConvertNonPrimitive(input)
	assert.Equal(t, want, input)
}

func TestFromMapstrSliceDuration(t *testing.T) {
	input := mapstr.M{"durations": []time.Duration{1500 * time.Millisecond, 2 * time.Second}}
	want := mapstr.M{"durations": []any{int64(1500 * time.Millisecond), int64(2 * time.Second)}}

	ConvertNonPrimitive(input)
	assert.Equal(t, want, input)

	pm := pcommon.NewMap()
	err := pm.FromRaw(map[string]any(input))
	assert.NoError(t, err, "unexpected error converting duration slice to pcommon map")
}

func TestFromMapstrString(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  any
		pcommon_val string
	}{
		"forty two": {mapstr_val: "forty two", pcommon_val: "forty two"},
		"empty":     {mapstr_val: "", pcommon_val: ""},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a := mapstr.M{"test": tc.mapstr_val}
			want := mapstr.M{}
			want["test"] = tc.pcommon_val
			ConvertNonPrimitive(a)
			assert.Equal(t, want, a)
		})
	}
}

func TestFromMapstrSliceString(t *testing.T) {
	inputSlice := []string{"1", "2", "3"}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := mapstr.M{}
	slice := make([]any, 0)
	for _, i := range inputSlice {
		slice = append(slice, i)
	}
	want["slice"] = slice
	ConvertNonPrimitive(inputMap)
	assert.Equal(t, want, inputMap)
}

func TestFromMapstrSliceInt(t *testing.T) {
	inputSlice := []int{42, 43, 44}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := mapstr.M{}
	slice := make([]any, 0)
	for _, i := range inputSlice {
		slice = append(slice, i)
	}
	want["slice"] = slice

	ConvertNonPrimitive(inputMap)
	assert.Equal(t, want, inputMap)
}

func TestFromMapstrSliceAny(t *testing.T) {
	inputSlice := []any{42, "forty-three", true}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := mapstr.M{
		"slice": inputSlice,
	}

	ConvertNonPrimitive(inputMap)
	assert.Equal(t, want, inputMap)
}

func TestFromMapstrSliceDouble(t *testing.T) {
	inputSlice := []float32{4.2, 4.3, 4.4}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := mapstr.M{}
	slice := make([]any, 0)
	for _, i := range inputSlice {
		slice = append(slice, i)
	}
	want["slice"] = slice

	ConvertNonPrimitive(inputMap)
	assert.Equal(t, want, inputMap)
}

// TestConvertNonPrimitiveWholeFloat verifies that whole-number floats are
// converted to int64 so that the OTel ES exporter (ExplicitRadixPoint=true)
// serialises them without a trailing ".0", matching Beats output.
func TestConvertNonPrimitiveWholeFloat(t *testing.T) {
	input := mapstr.M{
		"zero_f64":  float64(0.0),
		"one_f64":   float64(1.0),
		"neg_f64":   float64(-2.0),
		"zero_f32":  float32(0.0),
		"two_f32":   float32(2.0),
		"frac_f64":  float64(1.5),
		"frac_f32":  float32(1.5),
		"neg_frac":  float64(-1.5),
		"f64_slice": []float64{1.5, 2.0, 0.0},
		"f32_slice": []float32{1.5, 2.0, 0.0},
	}
	want := mapstr.M{
		"zero_f64":  int64(0),
		"one_f64":   int64(1),
		"neg_f64":   int64(-2),
		"zero_f32":  int64(0),
		"two_f32":   int64(2),
		"frac_f64":  float64(1.5),
		"frac_f32":  float32(1.5),
		"neg_frac":  float64(-1.5),
		"f64_slice": []any{float64(1.5), int64(2), int64(0)},
		"f32_slice": []any{float32(1.5), int64(2), int64(0)},
	}

	ConvertNonPrimitive(input)
	assert.Equal(t, want, input)
}

func TestFromMapstrBool(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  any
		pcommon_val bool
	}{
		"true":  {mapstr_val: true, pcommon_val: true},
		"false": {mapstr_val: false, pcommon_val: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a := mapstr.M{"test": tc.mapstr_val}
			want := mapstr.M{}
			want["test"] = tc.pcommon_val
			ConvertNonPrimitive(a)
			assert.Equal(t, want, a)
		})
	}
}

func TestFromMapstrSliceBool(t *testing.T) {
	inputSlice := []bool{true, false, true}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := mapstr.M{}
	slice := make([]any, 0)
	for _, i := range inputSlice {
		slice = append(slice, i)
	}
	want["slice"] = slice

	ConvertNonPrimitive(inputMap)
	assert.Equal(t, want, inputMap)
}

func TestFromMapstrMapstr(t *testing.T) {
	input := mapstr.M{
		"inner": mapstr.M{
			"inner_int":          42,
			"inner_string_slice": []string{"string"},
		},
	}
	want := mapstr.M{}
	want["inner"] = map[string]any{
		"inner_int":          42,
		"inner_string_slice": []any{"string"},
	}

	ConvertNonPrimitive(input)
	assert.Equal(t, want, input)
}

func TestFromMapstrSliceMapstr(t *testing.T) {
	inputSlice := []mapstr.M{{"item": 1}, {"item": 1}, {"item": 1}}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := mapstr.M{}
	want["slice"] = []any{
		map[string]any{
			"item": 1,
		},
		map[string]any{
			"item": 1,
		},
		map[string]any{
			"item": 1,
		},
	}

	ConvertNonPrimitive(inputMap)
	assert.Equal(t, want, inputMap)
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
		assert.NoError(t, err, "Error parsing time")
		sliceTimes = append(sliceTimes, targetTime)
		sliceTimesStr = append(sliceTimesStr, tc.pcommon_val)
	}
	inputMap := mapstr.M{
		"slice": sliceTimes,
	}
	want := mapstr.M{
		"slice": sliceTimesStr,
	}

	ConvertNonPrimitive(inputMap)
	assert.Equal(t, want, inputMap)
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
		assert.NoError(t, err, "Error parsing time")
		sliceTimes = append(sliceTimes, common.Time(targetTime))
		sliceTimesStr = append(sliceTimesStr, tc.pcommon_val)
	}
	inputMap := mapstr.M{
		"slice": sliceTimes,
	}
	want := mapstr.M{
		"slice": sliceTimesStr,
	}

	ConvertNonPrimitive(inputMap)
	assert.Equal(t, want, inputMap)
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
				"inner_slice": []map[string]any{ // slice -> slice
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
				"inner_slice": [2]map[string]any{ // array -> slice
					{"string": "string2"},
					{"number": 12.4},
				},
			},
		},
	}
	want := mapstr.M{
		"any_array":  []any{1, "string", 3},
		"any_slice":  []any{5.1, 6.2},
		"bool_array": []any{true, false},
		"bool_slice": []any{false, true},
		"struct": map[string]any{
			"value": "string",
		},
		"struct_with_text_marshaler": "marshalled:string",
		"inner": []any{
			map[string]any{
				"inner_int":       42,
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
				"inner_int": 43,
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

	ConvertNonPrimitive(input)
	assert.Equal(t, want, input)
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
		"slice": []any{
			int64(1),
			int64(2),
		},
		"map": map[string]any{
			"int": int64(42),
		},
	}

	got := ToMapstr(pm)
	assert.Equal(t, want, got)
}

func TestUnknownType(t *testing.T) {
	inputMap := mapstr.M{
		"unknown_map": map[string]int{"key": 42},
	}

	expected := mapstr.M{
		"unknown_map": "unknown type: map[string]int",
	}

	ConvertNonPrimitive(inputMap)
	assert.Equal(t, expected, inputMap)
}

func TestIsFloatWholeNumber(t *testing.T) {
	tests := []struct {
		name string
		f    float64
		want bool
	}{
		{name: "zero", f: 0.0, want: true},
		{name: "positive whole", f: 1.0, want: true},
		{name: "negative whole", f: -2.0, want: true},
		{name: "large whole", f: 1e15, want: true},
		{name: "min int64", f: float64(math.MinInt64), want: true},
		{name: "fractional", f: 1.5, want: false},
		{name: "negative fractional", f: -1.5, want: false},
		{name: "small nonzero", f: math.SmallestNonzeroFloat64, want: false},
		{name: "max float64", f: math.MaxFloat64, want: false},
		// float64(math.MaxInt64) rounds up to 2^63, which overflows int64.
		{name: "max int64 as float64 overflows", f: float64(math.MaxInt64), want: false},
		{name: "NaN", f: math.NaN(), want: false},
		{name: "positive infinity", f: math.Inf(1), want: false},
		{name: "negative infinity", f: math.Inf(-1), want: false},
		{name: "negative fraction no rounding", f: -0.99999999999999994, want: false},
		{name: "negative fraction that causes rounding", f: -0.99999999999999995, want: true}, // should round to -1
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isFloatWholeNumber(tc.f))
		})
	}
}
