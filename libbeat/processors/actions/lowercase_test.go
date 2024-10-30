package actions

import (
	"fmt"
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
		FullPath      bool
		Input         mapstr.M
		Output        mapstr.M
		Error         bool
	}{
		{
			Name:          "Lowercase Fields",
			Fields:        []string{"a.b.c", "Field1"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      true,
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
			Name:          "Lowercase Fields when full_path is false", // searches only the most nested key 'case insensitively'
			Fields:        []string{"a.B.c", "field1"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      false,
			Input: mapstr.M{
				"field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"C": "D",
					},
				},
				"A": mapstr.M{
					"b": mapstr.M{
						"C": "D",
					},
				},
			},
			Output: mapstr.M{
				"field1": mapstr.M{"Field2": "Value"},
				"Field3": "Value",
				"a": mapstr.M{
					"B": mapstr.M{
						"c": "D", // c is lowercased
					},
				},
				"A": mapstr.M{
					"b": mapstr.M{
						"C": "D",
					},
				},
			},

			Error: false,
		},
		{
			Name:          "Lowercase Fields with colliding keys",
			Fields:        []string{"ab"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      true,
			Input: mapstr.M{
				"ab": "first",
				"Ab": "second",
			},
			Output: mapstr.M{
				"ab":    "first",
				"Ab":    "second",
				"error": mapstr.M{"message": "could not fetch value for key: ab, Error: key collision on the same path \"ab\", previous match - \"ab\", another subkey - \"Ab\": key collision"},
			},
			Error: true,
		},
		{
			Name:          "Lowercase Fields when full_path is false",
			Fields:        []string{"ab"},
			IgnoreMissing: false,
			FailOnError:   true,
			FullPath:      false,
			Input: mapstr.M{
				"ab": "first",
				"Ab": "second",
			},
			Output: mapstr.M{
				"ab": "first",
				"Ab": "second",
			},
			Error: false,
		},
		{
			Name:          "Ignore Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: true,
			FailOnError:   true,
			FullPath:      true,
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
			FullPath:      true,
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
			FullPath:      true,
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
				FullPath:      test.FullPath,
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

func BenchmarkLowerCaseProcessorRun(b *testing.B) {
	tests := []struct {
		Name   string
		Events []beat.Event
	}{
		{
			Name:   "5000 events with 5 fields on each level with 3 level depth without collisions",
			Events: GenerateEvents(5000, 5, 3, false),
		},
		{
			Name:   "5000 events with 5 fields on each level with 3 level depth with collisions",
			Events: GenerateEvents(5000, 5, 3, true),
		},
		{
			Name:   "500 events with 50 fields on each level with 5 level depth without collisions",
			Events: GenerateEvents(500, 50, 3, false),
		},
		{
			Name:   "500 events with 50 fields on each level with 5 level depth with collisions",
			Events: GenerateEvents(500, 50, 3, true),
		},
		// Add more test cases as needed for benchmarking
	}

	for _, tt := range tests {
		b.Run(tt.Name, func(b *testing.B) {
			p := &alterFieldProcessor{
				Fields:        []string{"level1field1.level1field2.level3.field3"},
				alterFunc:     lowerCase,
				FullPath:      true,
				IgnoreMissing: false,
				FailOnError:   true,
			}
			for i := 0; i < b.N; i++ {
				//Run the function with the input
				for _, e := range tt.Events {
					p.Run(&e)
				}

			}
		})
	}
}

func GenerateEvents(numEvents, fieldsPerLevel, depth int, withCollisions bool) []beat.Event {
	events := make([]beat.Event, numEvents)
	for i := 0; i < numEvents; i++ {
		event := &beat.Event{Fields: mapstr.M{}}
		generateFields(event, fieldsPerLevel, depth, withCollisions)
		events[i] = *event
	}
	return events
}

// generateFields recursively generates fields for the event
func generateFields(event *beat.Event, fieldsPerLevel, depth int, withCollisions bool) {
	if depth == 0 {
		return
	}

	for j := 1; j <= fieldsPerLevel; j++ {
		var key string
		for d := 1; d < depth; d++ {
			key += fmt.Sprintf("level%dfield%d", d, j)
			key += "."
		}
		if withCollisions {
			key = fmt.Sprintf("Level%dField%d", depth, j) // Creating a collision (Level is capitalized)
		} else {
			key += fmt.Sprintf("level%dfield%d", depth, j)
		}
		event.Fields.Put(key, "value")
		key = ""
	}

}
