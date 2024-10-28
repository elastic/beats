package actions

import (
	"reflect"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

func TestNewLowerCaseProcessor(t *testing.T) {
	c := conf.MustNewConfigFrom(
		mapstr.M{
			"fields":         []string{"field1", "type", "field2"},
			"ignore_missing": true,
			"fail_on_error":  false,
		},
	)

	procInt, err := NewLowerCaseProcessor(c)
	assert.NoError(t, err)

	processor, ok := procInt.(*alterFieldProcessor)
	assert.True(t, ok)
	assert.Equal(t, []string{"field1", "field2"}, processor.Fields)
	assert.True(t, processor.IgnoreMissing)
	assert.False(t, processor.FailOnError)
}

func TestLowerCaseProcessorRun(t *testing.T) {
	tests := []struct {
		Name          string
		Fields        []string
		IgnoreMissing bool
		FailOnError   bool
		Input         mapstr.M
		Output        mapstr.M
		Error         bool
	}{
		{
			Name:          "Lowercase Fields",
			Fields:        []string{"a.'b.c'", "Field1"},
			IgnoreMissing: false,
			FailOnError:   true,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"C": "D",
					},
				},
			},
			Output: mapstr.M{
				"field1": mapstr.M{"Field2": "Value"}, // field1 is lowercased
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"c": "D", // c is lowercased
					},
				},
			},
			Error: false,
		},
		{
			Name:          "Lowercase Fields with multiple matching keys",
			Fields:        []string{"a.b.c"},
			IgnoreMissing: false,
			FailOnError:   true,
			Input: mapstr.M{
				"a": mapstr.M{
					"B": mapstr.M{
						"c": "first",
					},
				},
				"A": mapstr.M{
					"B": mapstr.M{
						"C": "second",
					},
				},
			},
			Output: mapstr.M{
				"A": mapstr.M{
					"B": mapstr.M{
						"c": "second", // c is lowercased
					},
				},
				"a": mapstr.M{
					"B": mapstr.M{
						"c": "first",
					},
				},
			},
			Error: false,
		},
		{
			Name:          "Lowercase Fields with colliding keys", // preserver the value of the last match
			Fields:        []string{"ab"},
			IgnoreMissing: false,
			FailOnError:   true,
			Input: mapstr.M{
				"ab": "first",
				"Ab": "second",
			},
			Output: mapstr.M{
				"ab": "second",
			},
			Error: false,
		},
		{
			Name:          "Ignore Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: true,
			FailOnError:   true,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Output: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Error: false,
		},
		{
			Name:          "Do Not Fail On Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: false,
			FailOnError:   false,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Output: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Error: false,
		},
		{
			Name:          "Fail On Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: false,
			FailOnError:   true,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
			},
			Output: mapstr.M{
				"Field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
				"error":  mapstr.M{"message": "could not fetch value for key: Field4, Error: key not found"},
			},
			Error: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			p := &alterFieldProcessor{
				Fields:        test.Fields,
				IgnoreMissing: test.IgnoreMissing,
				FailOnError:   test.FailOnError,
				alterFunc:     lowerCase,
			}

			event, err := p.Run(&beat.Event{Fields: test.Input})

			if !test.Error {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.True(t, reflect.DeepEqual(event.Fields, test.Output), event.Fields)
		})
	}
}
