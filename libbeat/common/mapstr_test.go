package common

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
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

func TestEnsureTimestampField(t *testing.T) {

	type io struct {
		Input  MapStr
		Output MapStr
	}

	tests := []io{
		// should add a @timestamp field if it doesn't exists.
		{
			Input: MapStr{},
			Output: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:56.123Z"),
			},
		},
		// should convert from string to Time
		{
			Input: MapStr{"@timestamp": "2015-03-01T12:34:57.123Z"},
			Output: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:57.123Z"),
			},
		},
		// should convert from time.Time to Time
		{
			Input: MapStr{
				"@timestamp": time.Date(2015, time.March, 01,
					12, 34, 57, 123*1e6, time.UTC),
			},
			Output: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:57.123Z"),
			},
		},
		// should leave a Time alone
		{
			Input: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:57.123Z"),
			},
			Output: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:57.123Z"),
			},
		},
	}

	now := func() time.Time {
		return time.Date(2015, time.March, 01, 12, 34, 56, 123*1e6, time.UTC)
	}

	for _, test := range tests {
		m := test.Input
		err := m.EnsureTimestampField(now)
		assert.Nil(t, err)
		assert.Equal(t, test.Output, m)
	}
}

func TestEnsureTimestampFieldNegative(t *testing.T) {

	inputs := []MapStr{
		// should error on invalid string layout (microseconds)
		{
			"@timestamp": "2015-03-01T12:34:57.123456Z",
		},
		// should error when the @timestamp is an integer
		{
			"@timestamp": 123456678,
		},
	}

	now := func() time.Time {
		return time.Date(2015, time.March, 01, 12, 34, 56, 123*1e6, time.UTC)
	}

	for _, input := range inputs {
		m := input
		err := m.EnsureTimestampField(now)
		assert.NotNil(t, err)
	}
}

func TestEnsureCountFiled(t *testing.T) {
	type io struct {
		Input  MapStr
		Output MapStr
	}
	tests := []io{
		// should add a count field if there is none
		{
			Input: MapStr{
				"a": "b",
			},
			Output: MapStr{
				"a":     "b",
				"count": 1,
			},
		},

		// should do nothing if there is already a count
		{
			Input: MapStr{
				"count": 1,
			},
			Output: MapStr{
				"count": 1,
			},
		},

		// should add count on an empty dict
		{
			Input:  MapStr{},
			Output: MapStr{"count": 1},
		},
	}

	for _, test := range tests {
		m := test.Input
		err := m.EnsureCountField()
		assert.Nil(t, err)
		assert.Equal(t, test.Output, m)
	}
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

func TestUnmarshalYAML(t *testing.T) {
	type io struct {
		InputLines []string
		Output     MapStr
	}
	tests := []io{
		// should return nil for empty document
		{
			InputLines: []string{},
			Output:     nil,
		},
		// should handle scalar values
		{
			InputLines: []string{
				"a: b",
				"c: true",
				"123: 456",
			},
			Output: MapStr{
				"a":   "b",
				"c":   "true",
				"123": "456",
			},
		},
		// should handle array with scalar values
		{
			InputLines: []string{
				"a:",
				"  - b",
				"  - true",
				"  - 123",
			},
			Output: MapStr{
				"a": []interface{}{"b", "true", "123"},
			},
		},
		// should handle array with nested map
		{
			InputLines: []string{
				"a:",
				"  - b: c",
				"    d: true",
				"    123: 456",
			},
			Output: MapStr{
				"a": []interface{}{
					MapStr{
						"b":   "c",
						"d":   "true",
						"123": "456",
					},
				},
			},
		},
		// should handle nested map
		{
			InputLines: []string{
				"a: ",
				"  b: c",
				"  d: true",
				"  123: 456",
			},
			Output: MapStr{
				"a": MapStr{
					"b":   "c",
					"d":   "true",
					"123": "456",
				},
			},
		},
	}
	for _, test := range tests {
		var actual MapStr
		if err := yaml.Unmarshal([]byte(strings.Join(test.InputLines, "\n")), &actual); err != nil {
			assert.Fail(t, "YAML unmarshaling unexpectedly failed: %s", err)
			continue
		}
		assert.Equal(t, test.Output, actual)
	}
}
