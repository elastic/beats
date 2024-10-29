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
		Fields        map[string]struct{}
		IgnoreMissing bool
		FailOnError   bool
		Input         mapstr.M
		Output        mapstr.M
		Error         bool
	}{
		{
			Name: "Lowercase Fields",
			Fields: map[string]struct{}{
				"a.b.c":  {},
				"field1": {},
			},
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
		// {
		// 	Name: "Lowercase Fields with multiple matching keys",
		// 	Fields: map[string]struct{}{
		// 		"a.b.c": {},
		// 	},
		// 	IgnoreMissing: false,
		// 	FailOnError:   true,
		// 	Input: mapstr.M{
		// 		"a": mapstr.M{
		// 			"B": mapstr.M{
		// 				"c": "first",
		// 			},
		// 		},
		// 		"A": mapstr.M{
		// 			"B": mapstr.M{
		// 				"C": "second",
		// 			},
		// 		},
		// 	},
		// 	Output: mapstr.M{
		// 		"A": mapstr.M{
		// 			"B": mapstr.M{
		// 				"c": "second", // c is lowercased
		// 			},
		// 		},
		// 		"a": mapstr.M{
		// 			"B": mapstr.M{
		// 				"c": "first",
		// 			},
		// 		},
		// 	},
		// 	Error: false,
		// },
		// {
		// 	Name: "Lowercase Fields with colliding keys", // preserver the value of the last match
		// 	Fields: map[string]struct{}{
		// 		"ab": {},
		// 	},
		// 	IgnoreMissing: false,
		// 	FailOnError:   true,
		// 	Input: mapstr.M{
		// 		"ab": "first",
		// 		"Ab": "second",
		// 	},
		// 	Output: mapstr.M{
		// 		"ab": "second",
		// 	},
		// 	Error: false,
		// },
		// {
		// 	Name:          "Ignore Missing Key Error",
		// 	Fields:        []string{"Field4"},
		// 	IgnoreMissing: true,
		// 	FailOnError:   true,
		// 	Input: mapstr.M{
		// 		"Field1": mapstr.M{"Field2": "Value"},
		// 		"Field3": "Value",
		// 	},
		// 	Output: mapstr.M{
		// 		"Field1": mapstr.M{"Field2": "Value"},
		// 		"Field3": "Value",
		// 	},
		// 	Error: false,
		// },
		// {
		// 	Name:          "Do Not Fail On Missing Key Error",
		// 	Fields:        []string{"Field4"},
		// 	IgnoreMissing: false,
		// 	FailOnError:   false,
		// 	Input: mapstr.M{
		// 		"Field1": mapstr.M{"Field2": "Value"},
		// 		"Field3": "Value",
		// 	},
		// 	Output: mapstr.M{
		// 		"Field1": mapstr.M{"Field2": "Value"},
		// 		"Field3": "Value",
		// 	},
		// 	Error: false,
		// },
		// {
		// 	Name:          "Fail On Missing Key Error",
		// 	Fields:        []string{"Field4"},
		// 	IgnoreMissing: false,
		// 	FailOnError:   true,
		// 	Input: mapstr.M{
		// 		"Field1": mapstr.M{"Field2": "Value"},
		// 		"Field3": "Value",
		// 	},
		// 	Output: mapstr.M{
		// 		"Field1": mapstr.M{"Field2": "Value"},
		// 		"Field3": "Value",
		// 		"error":  mapstr.M{"message": "could not fetch value for key: Field4, Error: key not found"},
		// 	},
		// 	Error: true,
		// },
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

func BenchmarkLowerCaseProcessorRun(b *testing.B) {
	tests := []struct {
		Name   string
		Events []beat.Event
	}{
		{
			Name:   "5000 events with 5 fields on each level with 3 level depth without collisions",
			Events: GenerateEvents(5000, 5, 3, false),
		},
		// Add more test cases as needed for benchmarking
	}

	for _, tt := range tests {
		b.Run(tt.Name, func(b *testing.B) {
			p := &alterFieldProcessor{
				Fields: map[string]struct{}{
					"level1field1": {},
				},
				alterFunc: lowerCase,
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
