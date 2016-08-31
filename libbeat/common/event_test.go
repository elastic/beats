package common

import (
	"testing"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

func TestConvertNestedMapStr(t *testing.T) {
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})

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
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})

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
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})

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
