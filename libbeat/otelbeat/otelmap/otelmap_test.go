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
		want := pcommon.NewMap()
		want.PutStr("test", tc.pcommon_val)
		got := FromMapstr(a)
		assert.Equal(t, want, got)
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
			want := pcommon.NewMap()
			want.PutStr("test", tc.pcommon_val)
			got := FromMapstr(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestFromMapstrSliceString(t *testing.T) {
	inputSlice := []string{"1", "2", "3"}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	sliceOfInt := want.PutEmptySlice("slice")
	for _, i := range inputSlice {
		val := sliceOfInt.AppendEmpty()
		val.SetStr(i)
	}

	got := FromMapstr(inputMap)
	assert.Equal(t, want, got)
}

func TestFromMapstrInt(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  interface{}
		pcommon_val int
	}{
		"int":    {mapstr_val: int(42), pcommon_val: 42},
		"int8":   {mapstr_val: int8(42), pcommon_val: 42},
		"int16":  {mapstr_val: int16(42), pcommon_val: 42},
		"int32":  {mapstr_val: int32(42), pcommon_val: 42},
		"int64":  {mapstr_val: int32(42), pcommon_val: 42},
		"uint":   {mapstr_val: uint(42), pcommon_val: 42},
		"uint8":  {mapstr_val: uint8(42), pcommon_val: 42},
		"uint16": {mapstr_val: uint16(42), pcommon_val: 42},
		"uint32": {mapstr_val: uint32(42), pcommon_val: 42},
		"uint64": {mapstr_val: uint64(42), pcommon_val: 42},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a := mapstr.M{"test": tc.mapstr_val}
			want := pcommon.NewMap()
			want.PutInt("test", int64(tc.pcommon_val))
			got := FromMapstr(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestFromMapstrSliceInt(t *testing.T) {
	inputSlice := []int{42, 43, 44}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	sliceOfInt := want.PutEmptySlice("slice")
	for _, i := range inputSlice {
		val := sliceOfInt.AppendEmpty()
		val.SetInt(int64(i))
	}

	got := FromMapstr(inputMap)
	assert.Equal(t, want, got)
}

func TestFromMapstrSliceAny(t *testing.T) {
	inputSlice := []any{42, "forty-three", true}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	sliceOfInt := want.PutEmptySlice("slice")

	val := sliceOfInt.AppendEmpty()
	val.SetInt(int64(inputSlice[0].(int)))
	val = sliceOfInt.AppendEmpty()
	val.SetStr(inputSlice[1].(string))
	val = sliceOfInt.AppendEmpty()
	val.SetBool(inputSlice[2].(bool))

	got := FromMapstr(inputMap)
	assert.Equal(t, want, got)
}

func TestFromMapstrDouble(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  interface{}
		pcommon_val float64
	}{
		"float32": {mapstr_val: float32(4.2), pcommon_val: float64(float32(4.2))},
		"float64": {mapstr_val: float64(4.2), pcommon_val: 4.2},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a := mapstr.M{"test": tc.mapstr_val}
			want := pcommon.NewMap()
			want.PutDouble("test", tc.pcommon_val)
			got := FromMapstr(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestFromMapstrSliceDouble(t *testing.T) {
	inputSlice := []float32{4.2, 4.3, 4.4}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	sliceOfInt := want.PutEmptySlice("slice")
	for _, i := range inputSlice {
		val := sliceOfInt.AppendEmpty()
		val.SetDouble(float64(i))
	}

	got := FromMapstr(inputMap)
	assert.Equal(t, want, got)
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
			want := pcommon.NewMap()
			want.PutBool("test", tc.pcommon_val)
			got := FromMapstr(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestFromMapstrSliceBool(t *testing.T) {
	inputSlice := []bool{true, false, true}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	pcommonSlice := want.PutEmptySlice("slice")
	for _, i := range inputSlice {
		val := pcommonSlice.AppendEmpty()
		val.SetBool(i)
	}

	got := FromMapstr(inputMap)
	assert.Equal(t, want, got)
}

func TestFromMapstrMapstr(t *testing.T) {
	input := mapstr.M{
		"inner": mapstr.M{
			"inner_int": 42,
		},
	}
	want := pcommon.NewMap()
	inner := want.PutEmptyMap("inner")
	inner.PutInt("inner_int", 42)

	got := FromMapstr(input)
	assert.Equal(t, want, got)
}

func TestFromMapstrSliceMapstr(t *testing.T) {
	inputSlice := []mapstr.M{mapstr.M{"item": 1}, mapstr.M{"item": 1}, mapstr.M{"item": 1}}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	sliceOfInt := want.PutEmptySlice("slice")
	for range inputSlice {
		val := sliceOfInt.AppendEmpty()
		newMap := pcommon.NewMap()
		newMap.PutInt("item", 1)
		newMap.CopyTo(val.SetEmptyMap())
	}

	got := FromMapstr(inputMap)
	assert.Equal(t, want, got)
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
	pcommonSlice := pcommon.NewSlice()
	for _, tc := range times {
		targetTime, err := time.Parse(time.RFC3339, tc.mapstr_val)
		assert.NoError(t, err, "Error parsing time")
		sliceTimes = append(sliceTimes, targetTime)
		pVal := pcommonSlice.AppendEmpty()
		pVal.SetStr(tc.pcommon_val)
	}
	inputMap := mapstr.M{
		"slice": sliceTimes,
	}
	want := pcommon.NewMap()
	pcommonSlice.CopyTo(want.PutEmptySlice("slice"))
	got := FromMapstr(inputMap)
	assert.Equal(t, want, got)
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
