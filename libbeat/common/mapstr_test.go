// +build !integration

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

func TestMergeFields(t *testing.T) {
	type io struct {
		UnderRoot bool
		Event     MapStr
		Fields    MapStr
		Output    MapStr
		Err       error
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
			Err: ErrorFieldsIsNotMapStr,
		},
	}

	for _, test := range tests {
		err := MergeFields(test.Event, test.Fields, test.UnderRoot)
		assert.Equal(t, test.Err, err)
		assert.Equal(t, test.Output, test.Event)
	}
}

func TestAddTag(t *testing.T) {
	type io struct {
		Event  MapStr
		Tags   []string
		Output MapStr
		Err    error
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
		// Existing tags, appends
		{
			Event: MapStr{
				"tags": []string{"json"},
			},
			Tags: []string{"docker"},
			Output: MapStr{
				"tags": []string{"json", "docker"},
			},
		},
		// Existing tags is not a []string
		{
			Event: MapStr{
				"tags": "not a slice",
			},
			Tags: []string{"docker"},
			Output: MapStr{
				"tags": "not a slice",
			},
			Err: ErrorTagsIsNotStringArray,
		},
	}

	for _, test := range tests {
		err := AddTags(test.Event, test.Tags)
		assert.Equal(t, test.Err, err)
		assert.Equal(t, test.Output, test.Event)
	}
}
