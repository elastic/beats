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

package otelconsumer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestMapstrToPcommonMapString(t *testing.T) {
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
			got := mapstrToPcommonMap(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestMapstrToPcommonMapSliceString(t *testing.T) {
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

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapInt(t *testing.T) {
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
			got := mapstrToPcommonMap(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestMapstrToPcommonMapSliceInt(t *testing.T) {
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

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapDouble(t *testing.T) {
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
			got := mapstrToPcommonMap(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestMapstrToPcommonMapSliceDouble(t *testing.T) {
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

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapBool(t *testing.T) {
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
			got := mapstrToPcommonMap(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestMapstrToPcommonMapSliceBool(t *testing.T) {
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

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapMapstr(t *testing.T) {
	input := mapstr.M{
		"inner": mapstr.M{
			"inner_int": 42,
		},
	}
	want := pcommon.NewMap()
	inner := want.PutEmptyMap("inner")
	inner.PutInt("inner_int", 42)

	got := mapstrToPcommonMap(input)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapSliceMapstr(t *testing.T) {
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

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapSliceTime(t *testing.T) {
	times := []struct {
		mapstr_val  string
		pcommon_val int64
	}{
		{mapstr_val: "2006-01-02T15:04:05+07:00", pcommon_val: 1136189045000},
		{mapstr_val: "1970-01-01T00:00:00+00:00", pcommon_val: 0},
	}
	var sliceTimes []time.Time
	pcommonSlice := pcommon.NewSlice()
	for _, tc := range times {
		targetTime, err := time.Parse(time.RFC3339, tc.mapstr_val)
		assert.NoError(t, err, "Error parsing time")
		sliceTimes = append(sliceTimes, targetTime)
		pVal := pcommonSlice.AppendEmpty()
		pVal.SetInt(tc.pcommon_val)
	}
	inputMap := mapstr.M{
		"slice": sliceTimes,
	}
	want := pcommon.NewMap()
	pcommonSlice.CopyTo(want.PutEmptySlice("slice"))
	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}
