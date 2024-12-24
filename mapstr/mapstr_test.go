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

package mapstr

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestMapStrUpdate(t *testing.T) {
	assert := assert.New(t)

	a := M{
		"a": 1,
		"b": 2,
	}
	b := M{
		"b": 3,
		"c": 4,
	}

	a.Update(b)

	assert.Equal(a, M{"a": 1, "b": 3, "c": 4})
}

func TestMapStrDeepUpdate(t *testing.T) {
	tests := []struct {
		a, b, expected M
	}{
		{
			M{"a": 1},
			M{"b": 2},
			M{"a": 1, "b": 2},
		},
		{
			M{"a": 1},
			M{"a": 2},
			M{"a": 2},
		},
		{
			M{"a": 1},
			M{"a": M{"b": 1}},
			M{"a": M{"b": 1}},
		},
		{
			M{"a": M{"b": 1}},
			M{"a": M{"c": 2}},
			M{"a": M{"b": 1, "c": 2}},
		},
		{
			M{"a": M{"b": 1}},
			M{"a": 1},
			M{"a": 1},
		},
		{
			M{"a.b": 1},
			M{"a": 1},
			M{"a": 1, "a.b": 1},
		},
		{
			M{"a": 1},
			M{"a.b": 1},
			M{"a": 1, "a.b": 1},
		},
		{
			M{"a": (M)(nil)},
			M{"a": M{"b": 1}},
			M{"a": M{"b": 1}},
		},
	}

	for i, test := range tests {
		a, b, expected := test.a, test.b, test.expected
		name := fmt.Sprintf("%v: %v + %v = %v", i, a, b, expected)

		t.Run(name, func(t *testing.T) {
			a.DeepUpdate(b)
			assert.Equal(t, expected, a)
		})
	}
}

func TestMapStrUnion(t *testing.T) {
	assert := assert.New(t)

	a := M{
		"a": 1,
		"b": 2,
	}
	b := M{
		"b": 3,
		"c": 4,
	}

	c := Union(a, b)

	assert.Equal(c, M{"a": 1, "b": 3, "c": 4})
}

func TestMapStrCopyFieldsTo(t *testing.T) {
	assert := assert.New(t)

	m := M{
		"a": M{
			"a1": 2,
			"a2": 3,
		},
		"b": 2,
		"c": M{
			"c1": 1,
			"c2": 2,
			"c3": M{
				"c31": 1,
				"c32": 2,
			},
		},
	}
	c := M{}

	err := m.CopyFieldsTo(c, "dd")
	assert.Error(err)
	assert.Equal(M{}, c)

	err = m.CopyFieldsTo(c, "a")
	assert.Equal(nil, err)
	assert.Equal(M{"a": M{"a1": 2, "a2": 3}}, c)

	err = m.CopyFieldsTo(c, "c.c1")
	assert.Equal(nil, err)
	assert.Equal(M{"a": M{"a1": 2, "a2": 3}, "c": M{"c1": 1}}, c)

	err = m.CopyFieldsTo(c, "b")
	assert.Equal(nil, err)
	assert.Equal(M{"a": M{"a1": 2, "a2": 3}, "c": M{"c1": 1}, "b": 2}, c)

	err = m.CopyFieldsTo(c, "c.c3.c32")
	assert.Equal(nil, err)
	assert.Equal(M{"a": M{"a1": 2, "a2": 3}, "c": M{"c1": 1, "c3": M{"c32": 2}}, "b": 2}, c)
}

func TestMapStrDelete(t *testing.T) {
	assert := assert.New(t)

	m := M{
		"c": M{
			"c1": 1,
			"c2": 2,
			"c3": M{
				"c31": 1,
				"c32": 2,
			},
		},
	}

	err := m.Delete("c.c2")
	assert.Equal(nil, err)
	assert.Equal(M{"c": M{"c1": 1, "c3": M{"c31": 1, "c32": 2}}}, m)

	err = m.Delete("c.c2.c21")
	assert.NotEqual(nil, err)
	assert.Equal(M{"c": M{"c1": 1, "c3": M{"c31": 1, "c32": 2}}}, m)

	err = m.Delete("c.c3.c31")
	assert.Equal(nil, err)
	assert.Equal(M{"c": M{"c1": 1, "c3": M{"c32": 2}}}, m)

	err = m.Delete("c")
	assert.Equal(nil, err)
	assert.Equal(M{}, m)
}

func TestHasKey(t *testing.T) {
	assert := assert.New(t)

	m := M{
		"c": M{
			"c1": 1,
			"c2": 2,
			"c3": M{
				"c31": 1,
				"c32": 2,
			},
			"c4.f": 19,
		},
		"d.f": 1,
	}

	hasKey, err := m.HasKey("c.c2")
	assert.Equal(nil, err)
	assert.Equal(true, hasKey)

	hasKey, err = m.HasKey("c.c4")
	assert.Equal(nil, err)
	assert.Equal(false, hasKey)

	hasKey, err = m.HasKey("c.c3.c32")
	assert.Equal(nil, err)
	assert.Equal(true, hasKey)

	hasKey, err = m.HasKey("dd")
	assert.Equal(nil, err)
	assert.Equal(false, hasKey)

	hasKey, err = m.HasKey("d.f")
	assert.Equal(nil, err)
	assert.Equal(true, hasKey)

	hasKey, err = m.HasKey("c.c4.f")
	assert.Equal(nil, err)
	assert.Equal(true, hasKey)
}

func TestMPut(t *testing.T) {
	m := M{
		"subMap": M{
			"a": 1,
		},
	}

	// Add new value to the top-level.
	v, err := m.Put("a", "ok")
	assert.NoError(t, err)
	assert.Nil(t, v)
	assert.Equal(t, M{"a": "ok", "subMap": M{"a": 1}}, m)

	// Add new value to subMap.
	v, err = m.Put("subMap.b", 2)
	assert.NoError(t, err)
	assert.Nil(t, v)
	assert.Equal(t, M{"a": "ok", "subMap": M{"a": 1, "b": 2}}, m)

	// Overwrite a value in subMap.
	v, err = m.Put("subMap.a", 2)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)
	assert.Equal(t, M{"a": "ok", "subMap": M{"a": 2, "b": 2}}, m)

	// Add value to map that does not exist.
	m = M{}
	v, err = m.Put("subMap.newMap.a", 1)
	assert.NoError(t, err)
	assert.Nil(t, v)
	assert.Equal(t, M{"subMap": M{"newMap": M{"a": 1}}}, m)
}

func TestMapStrGetValue(t *testing.T) {

	tests := []struct {
		input  M
		key    string
		output interface{}
		error  bool
	}{
		{
			M{"a": 1},
			"a",
			1,
			false,
		},
		{
			M{"a": M{"b": 1}},
			"a",
			M{"b": 1},
			false,
		},
		{
			M{"a": M{"b": 1}},
			"a.b",
			1,
			false,
		},
		{
			M{"a": M{"b.c": 1}},
			"a",
			M{"b.c": 1},
			false,
		},
		{
			M{"a": M{"b.c": 1}},
			"a.b",
			nil,
			true,
		},
		{
			M{"a.b": M{"c": 1}},
			"a.b",
			M{"c": 1},
			false,
		},
		{
			M{"a.b": M{"c": 1}},
			"a.b.c",
			nil,
			true,
		},
		{
			M{"a": M{"b.c": 1}},
			"a.b.c",
			1,
			false,
		},
	}

	for _, test := range tests {
		v, err := test.input.GetValue(test.key)
		if test.error {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, test.output, v)

	}
}

func TestClone(t *testing.T) {
	assert := assert.New(t)

	original := M{
		"c1": 1,
		"c2": 2,
		"c3": M{
			"c31": 1,
			"c32": 2,
		},
	}

	// Clone the original mapstr and then increment every value in it. Ensures the test will fail if
	// the cloned mapstr kept a reference to any part of the original.
	cloned := original.Clone()
	incrementMapstrValues(original)

	// Ensure that the cloned copy is as expected and no longer matches the original mapstr.
	assert.Equal(
		M{
			"c1": 1,
			"c2": 2,
			"c3": M{
				"c31": 1,
				"c32": 2,
			},
		},
		cloned,
	)
	assert.NotEqual(cloned, original)
}

func incrementMapstrValues(m M) {
	for k := range m {
		switch v := m[k].(type) {
		case int:
			m[k] = v + 1
		case M:
			incrementMapstrValues(m[k].(M))
		}

	}
}

func BenchmarkClone(b *testing.B) {
	assert := assert.New(b)

	m := M{
		"c1": 1,
		"c2": 2,
		"c3": M{
			"c31": 1,
			"c32": 2,
			"c33": 3,
			"c34": 4,
			"c35": 5,
			"c36": 6,
			"c37": 7,
			"c38": 8,
			"c39": 9,
		},
		"c4": 4,
		"c5": 5,
		"c6": 6,
		"c7": 7,
		"c8": 8,
		"c9": 9,
	}

	for i := 0; i < b.N; i++ {
		c := m.Clone()
		assert.Equal(m, c)
	}
}

func TestString(t *testing.T) {
	type io struct {
		Input  M
		Output string
	}
	tests := []io{
		{
			Input: M{
				"a": "b",
			},
			Output: `{"a":"b"}`,
		},
		{
			Input: M{
				"a": []int{1, 2, 3},
			},
			Output: `{"a":[1,2,3]}`,
		},
	}
	for _, test := range tests {
		assert.Equal(t, test.Output, test.Input.String())
	}
}

// Smoke test. The method has no observable outputs so this
// is only verifying there are no panics.
func TestStringToPrint(t *testing.T) {
	m := M{}

	assert.Equal(t, "{}", m.StringToPrint())
	assert.Equal(t, true, len(m.StringToPrint()) > 0)
}

func TestMergeFields(t *testing.T) {
	type io struct {
		UnderRoot bool
		Event     M
		Fields    M
		Output    M
		Err       string
	}
	tests := []io{
		// underRoot = true, merges
		{
			UnderRoot: true,
			Event: M{
				"a": "1",
			},
			Fields: M{
				"b": 2,
			},
			Output: M{
				"a": "1",
				"b": 2,
			},
		},

		// underRoot = true, overwrites existing
		{
			UnderRoot: true,
			Event: M{
				"a": "1",
			},
			Fields: M{
				"a": 2,
			},
			Output: M{
				"a": 2,
			},
		},

		// underRoot = false, adds new 'fields' when it doesn't exist
		{
			UnderRoot: false,
			Event: M{
				"a": "1",
			},
			Fields: M{
				"a": 2,
			},
			Output: M{
				"a": "1",
				"fields": M{
					"a": 2,
				},
			},
		},

		// underRoot = false, merge with existing 'fields' and overwrites existing keys
		{
			UnderRoot: false,
			Event: M{
				"fields": M{
					"a": "1",
					"b": 2,
				},
			},
			Fields: M{
				"a": 3,
				"c": 4,
			},
			Output: M{
				"fields": M{
					"a": 3,
					"b": 2,
					"c": 4,
				},
			},
		},

		// underRoot = false, error when 'fields' is wrong type
		{
			UnderRoot: false,
			Event: M{
				"fields": "not a M",
			},
			Fields: M{
				"a": 3,
			},
			Output: M{
				"fields": "not a M",
			},
			Err: "expected map",
		},
	}

	for _, test := range tests {
		err := MergeFields(test.Event, test.Fields, test.UnderRoot)
		assert.Equal(t, test.Output, test.Event)
		if test.Err != "" {
			assert.Contains(t, err.Error(), test.Err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestMergeFieldsDeep(t *testing.T) {
	type io struct {
		UnderRoot bool
		Event     M
		Fields    M
		Output    M
		Err       string
	}
	tests := []io{
		// underRoot = true, merges
		{
			UnderRoot: true,
			Event: M{
				"a": "1",
			},
			Fields: M{
				"b": 2,
			},
			Output: M{
				"a": "1",
				"b": 2,
			},
		},

		// underRoot = true, overwrites existing
		{
			UnderRoot: true,
			Event: M{
				"a": "1",
			},
			Fields: M{
				"a": 2,
			},
			Output: M{
				"a": 2,
			},
		},

		// underRoot = false, adds new 'fields' when it doesn't exist
		{
			UnderRoot: false,
			Event: M{
				"a": "1",
			},
			Fields: M{
				"a": 2,
			},
			Output: M{
				"a": "1",
				"fields": M{
					"a": 2,
				},
			},
		},

		// underRoot = false, merge with existing 'fields' and overwrites existing keys
		{
			UnderRoot: false,
			Event: M{
				"fields": M{
					"a": "1",
					"b": 2,
				},
			},
			Fields: M{
				"a": 3,
				"c": 4,
			},
			Output: M{
				"fields": M{
					"a": 3,
					"b": 2,
					"c": 4,
				},
			},
		},

		// underRoot = false, error when 'fields' is wrong type
		{
			UnderRoot: false,
			Event: M{
				"fields": "not a M",
			},
			Fields: M{
				"a": 3,
			},
			Output: M{
				"fields": "not a M",
			},
			Err: "expected map",
		},

		// underRoot = true, merges recursively
		{
			UnderRoot: true,
			Event: M{
				"my": M{
					"field1": "field1",
				},
			},
			Fields: M{
				"my": M{
					"field2": "field2",
					"field3": "field3",
				},
			},
			Output: M{
				"my": M{
					"field1": "field1",
					"field2": "field2",
					"field3": "field3",
				},
			},
		},

		// underRoot = true, merges recursively and overrides
		{
			UnderRoot: true,
			Event: M{
				"my": M{
					"field1": "field1",
					"field2": "field2",
				},
			},
			Fields: M{
				"my": M{
					"field2": "fieldTWO",
					"field3": "field3",
				},
			},
			Output: M{
				"my": M{
					"field1": "field1",
					"field2": "fieldTWO",
					"field3": "field3",
				},
			},
		},

		// underRoot = false, merges recursively under existing 'fields'
		{
			UnderRoot: false,
			Event: M{
				"fields": M{
					"my": M{
						"field1": "field1",
					},
				},
			},
			Fields: M{
				"my": M{
					"field2": "field2",
					"field3": "field3",
				},
			},
			Output: M{
				"fields": M{
					"my": M{
						"field1": "field1",
						"field2": "field2",
						"field3": "field3",
					},
				},
			},
		},
	}

	for _, test := range tests {
		err := MergeFieldsDeep(test.Event, test.Fields, test.UnderRoot)
		assert.Equal(t, test.Output, test.Event)
		if test.Err != "" {
			assert.Contains(t, err.Error(), test.Err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestAddTag(t *testing.T) {
	type io struct {
		Event  M
		Tags   []string
		Output M
		Err    string
	}
	tests := []io{
		// No existing tags, creates new tag array
		{
			Event: M{},
			Tags:  []string{"json"},
			Output: M{
				"tags": []string{"json"},
			},
		},
		// Existing tags is a []string, appends
		{
			Event: M{
				"tags": []string{"json"},
			},
			Tags: []string{"docker"},
			Output: M{
				"tags": []string{"json", "docker"},
			},
		},
		// Existing tags is a []interface{}, appends
		{
			Event: M{
				"tags": []interface{}{"json"},
			},
			Tags: []string{"docker"},
			Output: M{
				"tags": []interface{}{"json", "docker"},
			},
		},
		// Existing tags is not a []string or []interface{}
		{
			Event: M{
				"tags": "not a slice",
			},
			Tags: []string{"docker"},
			Output: M{
				"tags": "not a slice",
			},
			Err: "expected string array",
		},
	}

	for _, test := range tests {
		err := AddTags(test.Event, test.Tags)
		assert.Equal(t, test.Output, test.Event)
		if test.Err != "" {
			assert.Contains(t, err.Error(), test.Err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestAddTagsWithKey(t *testing.T) {
	type io struct {
		Event  M
		Key    string
		Tags   []string
		Output M
		Err    string
	}
	tests := []io{
		// No existing tags, creates new tag array
		{
			Event: M{},
			Key:   "tags",
			Tags:  []string{"json"},
			Output: M{
				"tags": []string{"json"},
			},
		},
		// Existing tags is a []string, appends
		{
			Event: M{
				"tags": []string{"json"},
			},
			Key:  "tags",
			Tags: []string{"docker"},
			Output: M{
				"tags": []string{"json", "docker"},
			},
		},
		// Existing tags are in submap and is a []interface{}, appends
		{
			Event: M{
				"log": M{
					"flags": []interface{}{"json"},
				},
			},
			Key:  "log.flags",
			Tags: []string{"docker"},
			Output: M{
				"log": M{
					"flags": []interface{}{"json", "docker"},
				},
			},
		},
		// Existing tags are in a submap and is not a []string or []interface{}
		{
			Event: M{
				"log": M{
					"flags": "not a slice",
				},
			},
			Key:  "log.flags",
			Tags: []string{"docker"},
			Output: M{
				"log": M{
					"flags": "not a slice",
				},
			},
			Err: "expected string array",
		},
	}

	for _, test := range tests {
		err := AddTagsWithKey(test.Event, test.Key, test.Tags)
		assert.Equal(t, test.Output, test.Event)
		if test.Err != "" {
			assert.Contains(t, err.Error(), test.Err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestFlatten(t *testing.T) {
	type data struct {
		Event    M
		Expected M
	}
	tests := []data{
		{
			Event: M{
				"hello": M{
					"world": 15,
				},
			},
			Expected: M{
				"hello.world": 15,
			},
		},
		{
			Event: M{
				"test": 15,
			},
			Expected: M{
				"test": 15,
			},
		},
		{
			Event: M{
				"test": 15,
				"hello": M{
					"world": M{
						"ok": "test",
					},
				},
				"elastic": M{
					"for": "search",
				},
			},
			Expected: M{
				"test":           15,
				"hello.world.ok": "test",
				"elastic.for":    "search",
			},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Expected, test.Event.Flatten())
	}
}

func TestFlattenKeys(t *testing.T) {
	expected := []string{"elastic.search.keys", "elastic.search", "elastic"}
	input := M{
		"elastic": M{
			"search": M{
				"keys": "value",
			},
		},
	}

	result := input.FlattenKeys()

	assert.Equal(t, &expected, result)
}

func BenchmarkMapStrFlatten(b *testing.B) {
	m := M{
		"test": 15,
		"hello": M{
			"world": M{
				"ok": "test",
			},
		},
		"elastic": M{
			"for": "search",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Flatten()
	}
}

// Ensure the MapStr is marshaled in logs the same way it is by json.Marshal.
func TestMapStrJSONLog(t *testing.T) {
	err := logp.DevelopmentSetup(logp.ToObserverOutput())
	require.Nil(t, err)

	m := M{
		"test": 15,
		"hello": M{
			"world": M{
				"ok": "test",
			},
		},
		"elastic": M{
			"for": "search",
		},
	}

	data, err := json.Marshal(M{"m": m})
	if err != nil {
		t.Fatal(err)
	}
	expectedJSON := string(data)

	logp.NewLogger("test").Infow("msg", "m", m)
	logs := logp.ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		log := logs[0]

		// Encode like zap does.
		e := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})
		buf, err := e.EncodeEntry(log.Entry, log.Context)
		if err != nil {
			t.Fatal(err)
		}

		// Zap adds a newline to end the JSON object.
		actualJSON := strings.TrimSpace(buf.String())

		assert.Equal(t, expectedJSON, actualJSON)
	}
}

func BenchmarkMapStrLogging(b *testing.B) {
	err := logp.DevelopmentSetup(logp.ToDiscardOutput())
	require.Nil(b, err)
	logger := logp.NewLogger("benchtest")

	m := M{
		"test": 15,
		"hello": M{
			"world": M{
				"ok": "test",
			},
		},
		"elastic": M{
			"for": "search",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infow("test", "mapstr", m)
	}
}

func BenchmarkWalkMap(b *testing.B) {

	globalM := M{
		"hello": M{
			"world": M{
				"ok": "test",
			},
		},
	}

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = globalM.GetValue("test.world.ok")
		}
	})

	b.Run("Put", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			m := M{
				"hello": M{
					"world": M{
						"ok": "test",
					},
				},
			}

			_, _ = m.Put("hello.world.new", 17)
		}
	})

	b.Run("PutMissing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			m := M{}

			_, _ = m.Put("a.b.c", 17)
		}
	})

	b.Run("HasKey", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = globalM.HasKey("hello.world.ok")
			_, _ = globalM.HasKey("hello.world.no_ok")
		}
	})

	b.Run("HasKeyFirst", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = globalM.HasKey("hello")
		}
	})

	b.Run("Delete", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			m := M{
				"hello": M{
					"world": M{
						"ok": "test",
					},
				},
			}
			_, _ = m.Put("hello.world.test", 17)
		}
	})
}

func TestFormat(t *testing.T) {
	input := M{
		"foo":      "bar",
		"password": "SUPER_SECURE",
	}

	tests := map[string]string{
		"%v":  `{"foo":"bar","password":"xxxxx"}`,
		"%+v": `{"foo":"bar","password":"SUPER_SECURE"}`,
		"%#v": `{"foo":"bar","password":"SUPER_SECURE"}`,
		"%s":  `{"foo":"bar","password":"xxxxx"}`,
		"%+s": `{"foo":"bar","password":"SUPER_SECURE"}`,
		"%#s": `{"foo":"bar","password":"SUPER_SECURE"}`,
	}

	for verb, expected := range tests {
		t.Run(verb, func(t *testing.T) {
			actual := fmt.Sprintf(verb, input)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestFindFold(t *testing.T) {
	field1level2 := M{
		"level3_Field1": "value2",
	}
	field1level1 := M{
		"non_map":       "value1",
		"level2_Field1": field1level2,
	}

	input := M{
		// baseline
		"level1_Field1": field1level1,
		// fold equal testing
		"Level1_fielD2": M{
			"lEvel2_fiEld2": M{
				"levEl3_fIeld2": "value3",
			},
		},
		// collision testing
		"level1_field2": M{
			"level2_field2": M{
				"level3_field2": "value4",
			},
		},
	}

	cases := []struct {
		name   string
		key    string
		expKey string
		expVal interface{}
		expErr string
	}{
		{
			name:   "returns normal key, full match",
			key:    "level1_Field1.level2_Field1.level3_Field1",
			expKey: "level1_Field1.level2_Field1.level3_Field1",
			expVal: "value2",
		},
		{
			name:   "returns normal key, partial match",
			key:    "level1_Field1.level2_Field1",
			expKey: "level1_Field1.level2_Field1",
			expVal: field1level2,
		},
		{
			name:   "returns normal key, one level",
			key:    "level1_Field1",
			expKey: "level1_Field1",
			expVal: field1level1,
		},
		{
			name:   "returns case-insensitive full match",
			key:    "level1_field1.level2_field1.level3_field1",
			expKey: "level1_Field1.level2_Field1.level3_Field1",
			expVal: "value2",
		},
		{
			name:   "returns case-insensitive partial match",
			key:    "level1_field1.level2_field1",
			expKey: "level1_Field1.level2_Field1",
			expVal: field1level2,
		},
		{
			name:   "returns case-insensitive one-level match",
			key:    "level1_field1",
			expKey: "level1_Field1",
			expVal: field1level1,
		},
		{
			name:   "returns collision error",
			key:    "level1_field2.level2_field2.level3_field2",
			expErr: "collision",
		},
		{
			name:   "returns non-map error",
			key:    "level1_field1.non_map.some_key",
			expErr: "is not a map",
		},
		{
			name:   "returns non-found error",
			key:    "level1_field1.not_exists.some_key",
			expErr: "key not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, val, err := input.FindFold(tc.key)
			if tc.expErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expErr)
				assert.Nil(t, val)
				assert.Empty(t, key)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expKey, key)
			assert.Equal(t, tc.expVal, val)
		})
	}
}

func TestAlterPath(t *testing.T) {
	var (
		lower AlterFunc = func(s string) (string, error) {
			return strings.ToLower(s), nil
		}

		exclamation AlterFunc = func(s string) (string, error) {
			return s + "!", nil
		}

		empty AlterFunc = func(string) (string, error) {
			return "", nil
		}

		errorFunc AlterFunc = func(string) (string, error) {
			return "", errors.New("oops")
		}
	)

	cases := []struct {
		name      string
		from      string
		mode      TraversalMode
		alterFunc AlterFunc
		m         M
		exp       M
		expErr    string
	}{
		{
			name:      "alters keys on root level with case-insensitive matching",
			from:      "level1_field1",
			mode:      CaseInsensitiveMode,
			alterFunc: lower,
			m: M{
				"level1_Field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "Value2",
						"level3_Field1": "Value3",
					},
				},
			},
			exp: M{
				"level1_field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "Value2",
						"level3_Field1": "Value3",
					},
				},
			},
		},
		{
			name:      "does not return an error if alterFunc returns the key unchanged",
			from:      "level1_field1.key",
			mode:      CaseInsensitiveMode,
			alterFunc: lower,
			m: M{
				"level1_field1": M{
					"Key": "value1",
				},
			},
			exp: M{
				"level1_field1": M{ //first level is unchanged - already in lowercase
					"key": "value1",
				},
			},
		},
		{
			name:      "alters keys on all nested levels with case-insensitive matching",
			from:      "level1_field1.level2_field1.level3_field1",
			mode:      CaseInsensitiveMode,
			alterFunc: lower,
			m: M{
				"level1_Field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "Value2",
						"level3_Field1": "Value3",
					},
				},
			},
			exp: M{
				"level1_field1": M{
					"Key": "value1",
					"level2_field1": M{
						"Key":           "Value2",
						"level3_field1": "Value3",
					},
				},
			},
		},
		{
			name:      "alters keys on all nested levels with case-sensitive matchig",
			from:      "level1_Field1.level2_Field1.level3_Field1",
			mode:      CaseSensitiveMode,
			alterFunc: exclamation,
			m: M{
				"level1_Field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "Value2",
						"level3_Field1": "Value3",
					},
				},
			},
			exp: M{
				"level1_Field1!": M{
					"Key": "value1",
					"level2_Field1!": M{
						"Key":            "Value2",
						"level3_Field1!": "Value3",
					},
				},
			},
		},
		{
			name:      "errors if the source does not exist",
			from:      "level1_Field1.NOT_EXIST.level3_Field1",
			mode:      CaseInsensitiveMode,
			alterFunc: lower,
			m: M{
				"level1_Field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "value2",
						"level3_Field1": "value3",
					},
				},
			},
			expErr: "key not found",
		},
		{
			name:      "errors if the casing does not match",
			from:      "level1_Field1.level2_field1.level3_Field1",
			mode:      CaseSensitiveMode,
			alterFunc: lower,
			m: M{
				"level1_Field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "value2",
						"level3_Field1": "value3",
					},
				},
			},
			expErr: "key not found",
		},
		{
			name:      "errors if the last segment does not match",
			from:      "level1_Field1.level2_Field1.level3_field1",
			mode:      CaseSensitiveMode,
			alterFunc: lower,
			m: M{
				"level1_Field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "value2",
						"level3_Field1": "value3",
					},
				},
			},
			expErr: "key not found",
		},
		{
			name:      "errors if the new name already exists",
			from:      "level1_Field1.level2_Field1.level3_Field1",
			mode:      CaseInsensitiveMode,
			alterFunc: lower,
			m: M{
				"level1_Field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "value2",
						"level3_Field1": "value3",
					},
					"Level2_field1": M{
						"Key":           "value4",
						"level3_Field2": "value5",
					},
				},
			},
			expErr: "key collision",
		},
		{
			name:      "errors if alter function returns empty string",
			from:      "level1_Field1.level2_Field1.level3_Field1",
			mode:      CaseInsensitiveMode,
			alterFunc: empty,
			m: M{
				"level1_Field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "value2",
						"level3_Field1": "value3",
					},
				},
			},
			expErr: "cannot be empty",
		},
		{
			name:      "errors if alter function returns error",
			from:      "level1_Field1.level2_Field1.level3_Field1",
			mode:      CaseInsensitiveMode,
			alterFunc: errorFunc,
			m: M{
				"level1_Field1": M{
					"Key": "value1",
					"level2_Field1": M{
						"Key":           "value2",
						"level3_Field1": "value3",
					},
				},
			},
			expErr: "oops",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloned := tc.m.Clone() // we need to preserve the initial state

			err := cloned.AlterPath(tc.from, tc.mode, tc.alterFunc)
			if tc.expErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.exp.StringToPrint(), cloned.StringToPrint())
		})
	}
}
