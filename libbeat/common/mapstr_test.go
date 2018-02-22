// +build !integration

package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/libbeat/logp"
)

func TestMapStrUpdate(t *testing.T) {
	assert := assert.New(t)

	a := MapStr{
		"a": 1,
		"b": 2,
	}
	b := MapStr{
		"b": 3,
		"c": 4,
	}

	a.Update(b)

	assert.Equal(a, MapStr{"a": 1, "b": 3, "c": 4})
}

func TestMapStrDeepUpdate(t *testing.T) {
	tests := []struct {
		a, b, expected MapStr
	}{
		{
			MapStr{"a": 1},
			MapStr{"b": 2},
			MapStr{"a": 1, "b": 2},
		},
		{
			MapStr{"a": 1},
			MapStr{"a": 2},
			MapStr{"a": 2},
		},
		{
			MapStr{"a": 1},
			MapStr{"a": MapStr{"b": 1}},
			MapStr{"a": MapStr{"b": 1}},
		},
		{
			MapStr{"a": MapStr{"b": 1}},
			MapStr{"a": MapStr{"c": 2}},
			MapStr{"a": MapStr{"b": 1, "c": 2}},
		},
		{
			MapStr{"a": MapStr{"b": 1}},
			MapStr{"a": 1},
			MapStr{"a": 1},
		},
		{
			MapStr{"a.b": 1},
			MapStr{"a": 1},
			MapStr{"a": 1, "a.b": 1},
		},
		{
			MapStr{"a": 1},
			MapStr{"a.b": 1},
			MapStr{"a": 1, "a.b": 1},
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

	a := MapStr{
		"a": 1,
		"b": 2,
	}
	b := MapStr{
		"b": 3,
		"c": 4,
	}

	c := MapStrUnion(a, b)

	assert.Equal(c, MapStr{"a": 1, "b": 3, "c": 4})
}

func TestMapStrCopyFieldsTo(t *testing.T) {
	assert := assert.New(t)

	m := MapStr{
		"a": MapStr{
			"a1": 2,
			"a2": 3,
		},
		"b": 2,
		"c": MapStr{
			"c1": 1,
			"c2": 2,
			"c3": MapStr{
				"c31": 1,
				"c32": 2,
			},
		},
	}
	c := MapStr{}

	err := m.CopyFieldsTo(c, "dd")
	assert.Error(err)
	assert.Equal(MapStr{}, c)

	err = m.CopyFieldsTo(c, "a")
	assert.Equal(nil, err)
	assert.Equal(MapStr{"a": MapStr{"a1": 2, "a2": 3}}, c)

	err = m.CopyFieldsTo(c, "c.c1")
	assert.Equal(nil, err)
	assert.Equal(MapStr{"a": MapStr{"a1": 2, "a2": 3}, "c": MapStr{"c1": 1}}, c)

	err = m.CopyFieldsTo(c, "b")
	assert.Equal(nil, err)
	assert.Equal(MapStr{"a": MapStr{"a1": 2, "a2": 3}, "c": MapStr{"c1": 1}, "b": 2}, c)

	err = m.CopyFieldsTo(c, "c.c3.c32")
	assert.Equal(nil, err)
	assert.Equal(MapStr{"a": MapStr{"a1": 2, "a2": 3}, "c": MapStr{"c1": 1, "c3": MapStr{"c32": 2}}, "b": 2}, c)
}

func TestMapStrDelete(t *testing.T) {
	assert := assert.New(t)

	m := MapStr{
		"c": MapStr{
			"c1": 1,
			"c2": 2,
			"c3": MapStr{
				"c31": 1,
				"c32": 2,
			},
		},
	}

	err := m.Delete("c.c2")
	assert.Equal(nil, err)
	assert.Equal(MapStr{"c": MapStr{"c1": 1, "c3": MapStr{"c31": 1, "c32": 2}}}, m)

	err = m.Delete("c.c2.c21")
	assert.NotEqual(nil, err)
	assert.Equal(MapStr{"c": MapStr{"c1": 1, "c3": MapStr{"c31": 1, "c32": 2}}}, m)

	err = m.Delete("c.c3.c31")
	assert.Equal(nil, err)
	assert.Equal(MapStr{"c": MapStr{"c1": 1, "c3": MapStr{"c32": 2}}}, m)

	err = m.Delete("c")
	assert.Equal(nil, err)
	assert.Equal(MapStr{}, m)
}

func TestHasKey(t *testing.T) {
	assert := assert.New(t)

	m := MapStr{
		"c": MapStr{
			"c1": 1,
			"c2": 2,
			"c3": MapStr{
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

func TestMapStrPut(t *testing.T) {
	m := MapStr{
		"subMap": MapStr{
			"a": 1,
		},
	}

	// Add new value to the top-level.
	v, err := m.Put("a", "ok")
	assert.NoError(t, err)
	assert.Nil(t, v)
	assert.Equal(t, MapStr{"a": "ok", "subMap": MapStr{"a": 1}}, m)

	// Add new value to subMap.
	v, err = m.Put("subMap.b", 2)
	assert.NoError(t, err)
	assert.Nil(t, v)
	assert.Equal(t, MapStr{"a": "ok", "subMap": MapStr{"a": 1, "b": 2}}, m)

	// Overwrite a value in subMap.
	v, err = m.Put("subMap.a", 2)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)
	assert.Equal(t, MapStr{"a": "ok", "subMap": MapStr{"a": 2, "b": 2}}, m)

	// Add value to map that does not exist.
	m = MapStr{}
	v, err = m.Put("subMap.newMap.a", 1)
	assert.NoError(t, err)
	assert.Nil(t, v)
	assert.Equal(t, MapStr{"subMap": MapStr{"newMap": MapStr{"a": 1}}}, m)
}

func TestMapStrGetValue(t *testing.T) {

	tests := []struct {
		input  MapStr
		key    string
		output interface{}
		error  bool
	}{
		{
			MapStr{"a": 1},
			"a",
			1,
			false,
		},
		{
			MapStr{"a": MapStr{"b": 1}},
			"a",
			MapStr{"b": 1},
			false,
		},
		{
			MapStr{"a": MapStr{"b": 1}},
			"a.b",
			1,
			false,
		},
		{
			MapStr{"a": MapStr{"b.c": 1}},
			"a",
			MapStr{"b.c": 1},
			false,
		},
		{
			MapStr{"a": MapStr{"b.c": 1}},
			"a.b",
			nil,
			true,
		},
		{
			MapStr{"a.b": MapStr{"c": 1}},
			"a.b",
			MapStr{"c": 1},
			false,
		},
		{
			MapStr{"a.b": MapStr{"c": 1}},
			"a.b.c",
			nil,
			true,
		},
		{
			MapStr{"a": MapStr{"b.c": 1}},
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

	m := MapStr{
		"c1": 1,
		"c2": 2,
		"c3": MapStr{
			"c31": 1,
			"c32": 2,
		},
	}

	c := m.Clone()
	assert.Equal(MapStr{"c31": 1, "c32": 2}, c["c3"])
}

func TestString(t *testing.T) {
	type io struct {
		Input  MapStr
		Output string
	}
	tests := []io{
		{
			Input: MapStr{
				"a": "b",
			},
			Output: `{"a":"b"}`,
		},
		{
			Input: MapStr{
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
	m := MapStr{}

	assert.Equal(t, "{}", m.StringToPrint())
	assert.Equal(t, true, len(m.StringToPrint()) > 0)
}

func TestMergeFields(t *testing.T) {
	type io struct {
		UnderRoot bool
		Event     MapStr
		Fields    MapStr
		Output    MapStr
		Err       string
	}
	tests := []io{
		// underRoot = true, merges
		{
			UnderRoot: true,
			Event: MapStr{
				"a": "1",
			},
			Fields: MapStr{
				"b": 2,
			},
			Output: MapStr{
				"a": "1",
				"b": 2,
			},
		},

		// underRoot = true, overwrites existing
		{
			UnderRoot: true,
			Event: MapStr{
				"a": "1",
			},
			Fields: MapStr{
				"a": 2,
			},
			Output: MapStr{
				"a": 2,
			},
		},

		// underRoot = false, adds new 'fields' when it doesn't exist
		{
			UnderRoot: false,
			Event: MapStr{
				"a": "1",
			},
			Fields: MapStr{
				"a": 2,
			},
			Output: MapStr{
				"a": "1",
				"fields": MapStr{
					"a": 2,
				},
			},
		},

		// underRoot = false, merge with existing 'fields' and overwrites existing keys
		{
			UnderRoot: false,
			Event: MapStr{
				"fields": MapStr{
					"a": "1",
					"b": 2,
				},
			},
			Fields: MapStr{
				"a": 3,
				"c": 4,
			},
			Output: MapStr{
				"fields": MapStr{
					"a": 3,
					"b": 2,
					"c": 4,
				},
			},
		},

		// underRoot = false, error when 'fields' is wrong type
		{
			UnderRoot: false,
			Event: MapStr{
				"fields": "not a MapStr",
			},
			Fields: MapStr{
				"a": 3,
			},
			Output: MapStr{
				"fields": "not a MapStr",
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

func TestAddTag(t *testing.T) {
	type io struct {
		Event  MapStr
		Tags   []string
		Output MapStr
		Err    string
	}
	tests := []io{
		// No existing tags, creates new tag array
		{
			Event: MapStr{},
			Tags:  []string{"json"},
			Output: MapStr{
				"tags": []string{"json"},
			},
		},
		// Existing tags is a []string, appends
		{
			Event: MapStr{
				"tags": []string{"json"},
			},
			Tags: []string{"docker"},
			Output: MapStr{
				"tags": []string{"json", "docker"},
			},
		},
		// Existing tags is a []interface{}, appends
		{
			Event: MapStr{
				"tags": []interface{}{"json"},
			},
			Tags: []string{"docker"},
			Output: MapStr{
				"tags": []interface{}{"json", "docker"},
			},
		},
		// Existing tags is not a []string or []interface{}
		{
			Event: MapStr{
				"tags": "not a slice",
			},
			Tags: []string{"docker"},
			Output: MapStr{
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

func TestFlatten(t *testing.T) {
	type data struct {
		Event    MapStr
		Expected MapStr
	}
	tests := []data{
		{
			Event: MapStr{
				"hello": MapStr{
					"world": 15,
				},
			},
			Expected: MapStr{
				"hello.world": 15,
			},
		},
		{
			Event: MapStr{
				"test": 15,
			},
			Expected: MapStr{
				"test": 15,
			},
		},
		{
			Event: MapStr{
				"test": 15,
				"hello": MapStr{
					"world": MapStr{
						"ok": "test",
					},
				},
				"elastic": MapStr{
					"for": "search",
				},
			},
			Expected: MapStr{
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

func BenchmarkMapStrFlatten(b *testing.B) {
	m := MapStr{
		"test": 15,
		"hello": MapStr{
			"world": MapStr{
				"ok": "test",
			},
		},
		"elastic": MapStr{
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
	logp.DevelopmentSetup(logp.ToObserverOutput())

	m := MapStr{
		"test": 15,
		"hello": MapStr{
			"world": MapStr{
				"ok": "test",
			},
		},
		"elastic": MapStr{
			"for": "search",
		},
	}

	data, err := json.Marshal(MapStr{"m": m})
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

		assert.Equal(t, string(expectedJSON), actualJSON)
	}
}

func BenchmarkMapStrLogging(b *testing.B) {
	logp.DevelopmentSetup(logp.ToDiscardOutput())
	logger := logp.NewLogger("benchtest")

	m := MapStr{
		"test": 15,
		"hello": MapStr{
			"world": MapStr{
				"ok": "test",
			},
		},
		"elastic": MapStr{
			"for": "search",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infow("test", "mapstr", m)
	}
}

func BenchmarkWalkMap(b *testing.B) {

	globalM := MapStr{
		"hello": MapStr{
			"world": MapStr{
				"ok": "test",
			},
		},
	}

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			globalM.GetValue("test.world.ok")
		}
	})

	b.Run("Put", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			m := MapStr{
				"hello": MapStr{
					"world": MapStr{
						"ok": "test",
					},
				},
			}

			m.Put("hello.world.new", 17)
		}
	})

	b.Run("PutMissing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			m := MapStr{}

			m.Put("a.b.c", 17)
		}
	})

	b.Run("HasKey", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			globalM.HasKey("hello.world.ok")
			globalM.HasKey("hello.world.no_ok")
		}
	})

	b.Run("HasKeyFirst", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			globalM.HasKey("hello")
		}
	})

	b.Run("Delete", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			m := MapStr{
				"hello": MapStr{
					"world": MapStr{
						"ok": "test",
					},
				},
			}
			m.Put("hello.world.test", 17)
		}
	})
}
