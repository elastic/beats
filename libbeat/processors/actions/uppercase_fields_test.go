package actions

import (
	"reflect"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

func TestNewUpperCaseProcessor(t *testing.T) {
	c := conf.MustNewConfigFrom(
		mapstr.M{
			"fields":         []string{"field1", "type", "field2"},
			"ignore_missing": true,
			"fail_on_error":  false,
		},
	)

	procInt, err := NewUpperCaseProcessor(c)
	assert.NoError(t, err)

	processor, ok := procInt.(*upperCaseProcessor)
	assert.True(t, ok)
	assert.Equal(t, []string{"field1", "field2"}, processor.Fields)
	assert.True(t, processor.IgnoreMissing)
	assert.False(t, processor.FailOnError)
}

func TestUpperCaseProcessorRun(t *testing.T) {
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
			Name:          "Uppercase Fields",
			Fields:        []string{"field1"},
			IgnoreMissing: false,
			FailOnError:   true,
			Input: mapstr.M{
				"field1": mapstr.M{"field2": "value"},
				"field3": "value",
			},
			Output: mapstr.M{
				"FIELD1": mapstr.M{"field2": "value"},
				"field3": "value",
			},
			Error: false,
		},
		{
			Name:          "Uppercase Nested Fields",
			Fields:        []string{"field1.field2"},
			IgnoreMissing: false,
			FailOnError:   true,
			Input: mapstr.M{
				"field1": mapstr.M{"field2": "value"},
				"field3": "value",
			},
			Output: mapstr.M{
				"field1": mapstr.M{"FIELD2": "value"},
				"field3": "value",
			},
			Error: false,
		},
		{
			Name:          "Ignore Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: true,
			FailOnError:   true,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "value"},
				"Field3": "value",
			},
			Output: mapstr.M{
				"Field1": mapstr.M{"Field2": "value"},
				"Field3": "value",
			},
			Error: false,
		},
		{
			Name:          "Do Not Fail On Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: false,
			FailOnError:   false,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "value"},
				"Field3": "value",
			},
			Output: mapstr.M{
				"Field1": mapstr.M{"Field2": "value"},
				"Field3": "value",
			},
			Error: false,
		},
		{
			Name:          "Fail On Missing Key Error",
			Fields:        []string{"Field4"},
			IgnoreMissing: false,
			FailOnError:   true,
			Input: mapstr.M{
				"Field1": mapstr.M{"Field2": "value"},
				"Field3": "value",
			},
			Output: mapstr.M{
				"Field1": mapstr.M{"Field2": "value"},
				"Field3": "value",
				"error":  mapstr.M{"message": "could not fetch value for key: Field4, Error: key not found"},
			},
			Error: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			p := &upperCaseProcessor{
				Fields:        test.Fields,
				IgnoreMissing: test.IgnoreMissing,
				FailOnError:   test.FailOnError,
			}

			event, err := p.Run(&beat.Event{Fields: test.Input})

			if !test.Error {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.True(t, reflect.DeepEqual(event.Fields, test.Output))
		})
	}
}
