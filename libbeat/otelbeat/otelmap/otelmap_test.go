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
	inputSlice := []mapstr.M{mapstr.M{"item": 1}, mapstr.M{"item": 1}, mapstr.M{"item": 1}}
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

type unknown int

func TestUnknownType(t *testing.T) {
	inputMap := mapstr.M{
		"slice": []unknown{42, 43, 44},
	}

	expected := mapstr.M{
		"slice": "unknown type: []otelmap.unknown",
	}

	ConvertNonPrimitive(inputMap)
	assert.Equal(t, expected, inputMap)
}
