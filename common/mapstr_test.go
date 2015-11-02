package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
		io{
			Input: MapStr{},
			Output: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:56.123Z"),
			},
		},
		// should convert from string to Time
		io{
			Input: MapStr{"@timestamp": "2015-03-01T12:34:57.123Z"},
			Output: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:57.123Z"),
			},
		},
		// should convert from time.Time to Time
		io{
			Input: MapStr{
				"@timestamp": time.Date(2015, time.March, 01,
					12, 34, 57, 123*1e6, time.UTC),
			},
			Output: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:57.123Z"),
			},
		},
		// should leave a Time alone
		io{
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
		MapStr{
			"@timestamp": "2015-03-01T12:34:57.123456Z",
		},
		// should error when the @timestamp is an integer
		MapStr{
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
		io{
			Input: MapStr{
				"a": "b",
			},
			Output: MapStr{
				"a":     "b",
				"count": 1,
			},
		},

		// should do nothing if there is already a count
		io{
			Input: MapStr{
				"count": 1,
			},
			Output: MapStr{
				"count": 1,
			},
		},

		// should add count on an empty dict
		io{
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
		io{
			Input: MapStr{
				"a": "b",
			},
			Output: `{"a":"b"}`,
		},
		io{
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
