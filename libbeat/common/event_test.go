package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"
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

	for i, test := range tests {
		assert.Equal(t, test.Output, ConvertToGenericEvent(test.Input), "Test case %d", i)
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

	for i, test := range tests {
		assert.EqualValues(t, test.Output, ConvertToGenericEvent(test.Input), "Test case %v", i)
	}
}

func TestNormalizeValue(t *testing.T) {
	logp.TestingSetup()

	var nilStringPtr *string
	someString := "foo"

	type mybool bool
	type myint int32
	type myuint uint8

	var tests = []struct {
		in  interface{}
		out interface{}
	}{
		{nil, nil},
		{&someString, someString},   // Pointers are dereferenced.
		{nilStringPtr, nil},         // Nil pointers are dropped.
		{NetString("test"), "test"}, // It honors the TextMarshaler contract.
		{true, true},
		{int8(8), int8(8)},
		{uint8(8), uint8(8)},
		{"hello", "hello"},
		{map[string]interface{}{"foo": "bar"}, MapStr{"foo": "bar"}},

		// Other map types are converted using marshalUnmarshal which will lose
		// type information for arrays which become []interface{} and numbers
		// which all become float64.
		{map[string]string{"foo": "bar"}, MapStr{"foo": "bar"}},
		{map[string][]string{"list": {"foo", "bar"}}, MapStr{"list": []interface{}{"foo", "bar"}}},

		{[]string{"foo", "bar"}, []string{"foo", "bar"}},
		{[]bool{true, false}, []bool{true, false}},
		{[]string{"foo", "bar"}, []string{"foo", "bar"}},
		{[]int{10, 11}, []int{10, 11}},
		{[]MapStr{{"foo": "bar"}}, []MapStr{{"foo": "bar"}}},
		{[]map[string]interface{}{{"foo": "bar"}}, []MapStr{{"foo": "bar"}}},

		// Wrapper types are converted to primitives using reflection.
		{mybool(true), true},
		{myint(32), int64(32)},
		{myuint(8), uint64(8)},

		// Slices of wrapper types are converted to an []interface{} of primitives.
		{[]mybool{true, false}, []interface{}{true, false}},
		{[]myint{32}, []interface{}{int64(32)}},
		{[]myuint{8}, []interface{}{uint64(8)}},
	}

	for i, test := range tests {
		out, err := normalizeValue(test.in)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.out, out, "Test case %v", i)
	}

	var floatTests = []struct {
		in  interface{}
		out interface{}
	}{
		{float32(1), float64(1)},
		{float64(1), float64(1)},
	}

	for i, test := range floatTests {
		out, err := normalizeValue(test.in)
		if err != nil {
			t.Error(err)
			continue
		}
		assert.InDelta(t, test.out, float64(out.(Float)), 0.000001, "(approximate) Test case %v", i)
	}
}

func TestNormalizeMapError(t *testing.T) {
	badInputs := []MapStr{
		{"func": func() {}},
		{"chan": make(chan struct{})},
		{"uintptr": uintptr(123)},
	}

	for i, in := range badInputs {
		_, errs := normalizeMap(in, "bad.type")
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

func TestMarshalFloatValues(t *testing.T) {
	assert := assert.New(t)

	var f float64

	f = 5

	a := MapStr{
		"f": Float(f),
	}

	b, err := json.Marshal(a)
	assert.Nil(err)
	assert.Equal(string(b), "{\"f\":5.000000}")
}

func TestNormalizeTime(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().In(ny)
	v, errs := normalizeValue(now, "@timestamp")
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
	for i := 0; i < b.N; i++ {
		ConvertToGenericEvent(MapStr{"key": NetString("hola")})
	}
}

// Uses reflection.
func BenchmarkConvertToGenericEventMapStringString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConvertToGenericEvent(MapStr{"key": map[string]string{"greeting": "hola"}})
	}
}

// Uses recursion to step into the nested MapStr.
func BenchmarkConvertToGenericEventMapStr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConvertToGenericEvent(MapStr{"key": map[string]interface{}{"greeting": "hola"}})
	}
}

// No reflection required.
func BenchmarkConvertToGenericEventStringSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConvertToGenericEvent(MapStr{"key": []string{"foo", "bar"}})
	}
}

// Uses reflection to convert the string array.
func BenchmarkConvertToGenericEventCustomStringSlice(b *testing.B) {
	type myString string
	for i := 0; i < b.N; i++ {
		ConvertToGenericEvent(MapStr{"key": []myString{"foo", "bar"}})
	}
}

// Pointers require reflection to generically dereference.
func BenchmarkConvertToGenericEventStringPointer(b *testing.B) {
	val := "foo"
	for i := 0; i < b.N; i++ {
		ConvertToGenericEvent(MapStr{"key": &val})
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
