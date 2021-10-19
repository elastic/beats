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

package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestConvertNestedMapStr(t *testing.T) {
	logp.TestingSetup()

	type io struct {
		Input  MapStr
		Output MapStr
	}

	type String string

	tests := []io{
		{
			Input: MapStr{
				"key": MapStr{
					"key1": "value1",
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": "value1",
				},
			},
		},
		{
			Input: MapStr{
				"key": MapStr{
					"key1": String("value1"),
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": "value1",
				},
			},
		},
		{
			Input: MapStr{
				"key": MapStr{
					"key1": []string{"value1", "value2"},
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": []string{"value1", "value2"},
				},
			},
		},
		{
			Input: MapStr{
				"key": MapStr{
					"key1": []String{"value1", "value2"},
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": []interface{}{"value1", "value2"},
				},
			},
		},
		{
			Input: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:56.123Z"),
			},
			Output: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:56.123Z"),
			},
		},
		{
			Input: MapStr{
				"env":  nil,
				"key2": uintptr(88),
				"key3": func() { t.Log("hello") },
			},
			Output: MapStr{},
		},
		{
			Input: MapStr{
				"key": []MapStr{
					{"keyX": []String{"value1", "value2"}},
				},
			},
			Output: MapStr{
				"key": []MapStr{
					{"keyX": []interface{}{"value1", "value2"}},
				},
			},
		},
		{
			Input: MapStr{
				"key": []interface{}{
					MapStr{"key1": []string{"value1", "value2"}},
				},
			},
			Output: MapStr{
				"key": []interface{}{
					MapStr{"key1": []string{"value1", "value2"}},
				},
			},
		},
		{
			MapStr{"k": map[string]int{"hits": 1}},
			MapStr{"k": MapStr{"hits": float64(1)}},
		},
	}

	g := NewGenericEventConverter(false)
	for i, test := range tests {
		assert.Equal(t, test.Output, g.Convert(test.Input), "Test case %d", i)
	}
}

func TestConvertNestedStruct(t *testing.T) {
	logp.TestingSetup()

	type io struct {
		Input  MapStr
		Output MapStr
	}

	type TestStruct struct {
		A string
		B int
	}

	tests := []io{
		{
			Input: MapStr{
				"key": MapStr{
					"key1": TestStruct{
						A: "hello",
						B: 5,
					},
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": MapStr{
						"A": "hello",
						"B": float64(5),
					},
				},
			},
		},
		{
			Input: MapStr{
				"key": []interface{}{
					TestStruct{
						A: "hello",
						B: 5,
					},
				},
			},
			Output: MapStr{
				"key": []interface{}{
					MapStr{
						"A": "hello",
						"B": float64(5),
					},
				},
			},
		},
	}

	g := NewGenericEventConverter(false)
	for i, test := range tests {
		assert.EqualValues(t, test.Output, g.Convert(test.Input), "Test case %v", i)
	}
}

func TestConvertWithNullEmission(t *testing.T) {
	logp.TestingSetup()

	type io struct {
		Input  MapStr
		Output MapStr
	}

	type String string
	type TestStruct struct {
		A interface{}
	}

	tests := []io{
		{
			Input: MapStr{
				"key": MapStr{
					"key1": nil,
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": nil,
				},
			},
		},
		{
			Input: MapStr{
				"key": TestStruct{
					A: nil,
				},
			},
			Output: MapStr{
				"key": MapStr{
					"A": nil,
				},
			},
		}}

	g := NewGenericEventConverter(true)
	for i, test := range tests {
		assert.EqualValues(t, test.Output, g.Convert(test.Input), "Test case %v", i)
	}
}

func TestNormalizeValue(t *testing.T) {
	logp.TestingSetup()

	type testCase struct{ in, out interface{} }

	runTests := func(check func(t *testing.T, a, b interface{}), tests map[string]testCase) {
		g := NewGenericEventConverter(false)
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				out, err := g.normalizeValue(test.in)
				if err != nil {
					t.Error(err)
					return
				}
				check(t, test.out, out)
			})
		}
	}

	checkEq := func(t *testing.T, a, b interface{}) {
		assert.Equal(t, a, b)
	}

	checkDelta := func(t *testing.T, a, b interface{}) {
		assert.InDelta(t, a, b, 0.000001)
	}

	var nilStringPtr *string
	var nilTimePtr *time.Time
	someString := "foo"
	uuidValue, err := uuid.NewV1()
	if err != nil {
		t.Fatalf("error while generating uuid: %v", err)
	}

	type mybool bool
	type myint int32
	type myuint uint8
	type myuint64 uint64

	runTests(checkEq, map[string]testCase{
		"nil":                               {nil, nil},
		"pointers are dereferenced":         {&someString, someString},
		"drop nil string pointer":           {nilStringPtr, nil},
		"drop nil time pointer":             {nilTimePtr, nil},
		"UUID supports TextMarshaller":      {uuidValue, uuidValue.String()},
		"NetString supports TextMarshaller": {NetString("test"), "test"},
		"bool value":                        {true, true},
		"int8 value":                        {int8(8), int8(8)},
		"uint8 value":                       {uint8(8), uint8(8)},
		"uint64 masked":                     {uint64(1<<63 + 10), uint64(10)},
		"string value":                      {"hello", "hello"},
		"map to MapStr":                     {map[string]interface{}{"foo": "bar"}, MapStr{"foo": "bar"}},

		// Other map types are converted using marshalUnmarshal which will lose
		// type information for arrays which become []interface{} and numbers
		// which all become float64.
		"map[string]string to MapStr":   {map[string]string{"foo": "bar"}, MapStr{"foo": "bar"}},
		"map[string][]string to MapStr": {map[string][]string{"list": {"foo", "bar"}}, MapStr{"list": []interface{}{"foo", "bar"}}},

		"array of strings":       {[]string{"foo", "bar"}, []string{"foo", "bar"}},
		"array of bools":         {[]bool{true, false}, []bool{true, false}},
		"array of ints":          {[]int{10, 11}, []int{10, 11}},
		"array of uint64 ok":     {[]uint64{1, 2, 3}, []uint64{1, 2, 3}},
		"array of uint64 masked": {[]uint64{1<<63 + 1, 1<<63 + 2, 1<<63 + 3}, []uint64{1, 2, 3}},
		"array of MapStr":        {[]MapStr{{"foo": "bar"}}, []MapStr{{"foo": "bar"}}},
		"array of map to MapStr": {[]map[string]interface{}{{"foo": "bar"}}, []MapStr{{"foo": "bar"}}},

		// Wrapper types are converted to primitives using reflection.
		"custom bool type":          {mybool(true), true},
		"custom int type":           {myint(32), int64(32)},
		"custom uint type":          {myuint(8), uint64(8)},
		"custom uint64 type ok":     {myuint64(23), uint64(23)},
		"custom uint64 type masked": {myuint64(1<<63 + 42), uint64(42)},

		// Slices of wrapper types are converted to an []interface{} of primitives.
		"array of custom bool type":     {[]mybool{true, false}, []interface{}{true, false}},
		"array of custom int type":      {[]myint{32}, []interface{}{int64(32)}},
		"array of custom uint type":     {[]myuint{8}, []interface{}{uint64(8)}},
		"array of custom uint64 ok":     {[]myuint64{64}, []interface{}{uint64(64)}},
		"array of custom uint64 masked": {[]myuint64{1<<63 + 64}, []interface{}{uint64(64)}},
	})

	runTests(checkDelta, map[string]testCase{
		"float32 value": {float32(1), float64(1)},
		"float64 value": {float64(1), float64(1)},
	})
}

func TestNormalizeMapError(t *testing.T) {
	badInputs := []MapStr{
		{"func": func() {}},
		{"chan": make(chan struct{})},
		{"uintptr": uintptr(123)},
	}

	g := NewGenericEventConverter(false)
	for i, in := range badInputs {
		_, errs := g.normalizeMap(in, "bad.type")
		if assert.Len(t, errs, 1) {
			t.Log(errs[0])
			assert.Contains(t, errs[0].Error(), "key=bad.type", "Test case %v", i)
		}
	}
}

func TestJoinKeys(t *testing.T) {
	assert.Equal(t, "", joinKeys(""))
	assert.Equal(t, "co", joinKeys("co"))
	assert.Equal(t, "co.elastic", joinKeys("", "co", "elastic"))
	assert.Equal(t, "co.elastic", joinKeys("co", "elastic"))
}

func TestMarshalUnmarshalMap(t *testing.T) {
	tests := []struct {
		in  MapStr
		out MapStr
	}{
		{MapStr{"names": []string{"a", "b"}}, MapStr{"names": []interface{}{"a", "b"}}},
	}

	for i, test := range tests {
		var out MapStr
		err := marshalUnmarshal(test.in, &out)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.out, out, "Test case %v", i)
	}
}

func TestMarshalUnmarshalArray(t *testing.T) {
	tests := []struct {
		in  interface{}
		out interface{}
	}{
		{[]string{"a", "b"}, []interface{}{"a", "b"}},
	}

	for i, test := range tests {
		var out interface{}
		err := marshalUnmarshal(test.in, &out)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.out, out, "Test case %v", i)
	}
}

func TestNormalizeTime(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().In(ny)
	g := NewGenericEventConverter(false)
	v, errs := g.normalizeValue(now, "@timestamp")
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	utcCommonTime, ok := v.(Time)
	if !ok {
		t.Fatalf("expected common.Time, but got %T (%v)", v, v)
	}

	assert.Equal(t, time.UTC, time.Time(utcCommonTime).Location())
	assert.True(t, now.Equal(time.Time(utcCommonTime)))
}

// Uses TextMarshaler interface.
func BenchmarkConvertToGenericEventNetString(b *testing.B) {
	g := NewGenericEventConverter(false)
	for i := 0; i < b.N; i++ {
		g.Convert(MapStr{"key": NetString("hola")})
	}
}

// Uses reflection.
func BenchmarkConvertToGenericEventMapStringString(b *testing.B) {
	g := NewGenericEventConverter(false)
	for i := 0; i < b.N; i++ {
		g.Convert(MapStr{"key": map[string]string{"greeting": "hola"}})
	}
}

// Uses recursion to step into the nested MapStr.
func BenchmarkConvertToGenericEventMapStr(b *testing.B) {
	g := NewGenericEventConverter(false)
	for i := 0; i < b.N; i++ {
		g.Convert(MapStr{"key": map[string]interface{}{"greeting": "hola"}})
	}
}

// No reflection required.
func BenchmarkConvertToGenericEventStringSlice(b *testing.B) {
	g := NewGenericEventConverter(false)
	for i := 0; i < b.N; i++ {
		g.Convert(MapStr{"key": []string{"foo", "bar"}})
	}
}

// Uses reflection to convert the string array.
func BenchmarkConvertToGenericEventCustomStringSlice(b *testing.B) {
	g := NewGenericEventConverter(false)
	type myString string
	for i := 0; i < b.N; i++ {
		g.Convert(MapStr{"key": []myString{"foo", "bar"}})
	}
}

// Pointers require reflection to generically dereference.
func BenchmarkConvertToGenericEventStringPointer(b *testing.B) {
	g := NewGenericEventConverter(false)
	val := "foo"
	for i := 0; i < b.N; i++ {
		g.Convert(MapStr{"key": &val})
	}
}
func TestDeDotJSON(t *testing.T) {
	var tests = []struct {
		input  []byte
		output []byte
		valuer func() interface{}
	}{
		{
			input: []byte(`[
				{"key_with_dot.1":"value1_1"},
				{"key_without_dot_2":"value1_2"},
				{"key_with_multiple.dots.3": {"key_with_dot.2":"value2_1"}}
			]
			`),
			output: []byte(`[
				{"key_with_dot_1":"value1_1"},
				{"key_without_dot_2":"value1_2"},
				{"key_with_multiple_dots_3": {"key_with_dot_2":"value2_1"}}
			]
			`),
			valuer: func() interface{} { return []interface{}{} },
		},
		{
			input: []byte(`{
				"key_without_dot_l1": {
					"key_with_dot.l2": 1,
					"key.with.multiple.dots_l2": 2,
					"key_without_dot_l2": {
						"key_with_dot.l3": 3,
						"key.with.multiple.dots_l3": 4
					}
				}
			}
			`),
			output: []byte(`{
				"key_without_dot_l1": {
					"key_with_dot_l2": 1,
					"key_with_multiple_dots_l2": 2,
					"key_without_dot_l2": {
						"key_with_dot_l3": 3,
						"key_with_multiple_dots_l3": 4
					}
				}
			}
			`),
			valuer: func() interface{} { return map[string]interface{}{} },
		},
	}
	for _, test := range tests {
		input, output := test.valuer(), test.valuer()
		assert.Nil(t, json.Unmarshal(test.input, &input))
		assert.Nil(t, json.Unmarshal(test.output, &output))
		assert.Equal(t, output, DeDotJSON(input))
		if _, ok := test.valuer().(map[string]interface{}); ok {
			assert.Equal(t, MapStr(output.(map[string]interface{})), DeDotJSON(MapStr(input.(map[string]interface{}))))
		}
	}
}
