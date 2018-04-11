package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestIncludeFields(t *testing.T) {

	var tests = []struct {
		Fields []string
		Input  common.MapStr
		Output common.MapStr
	}{
		{
			Fields: []string{"test"},
			Input: common.MapStr{
				"hello": "world",
				"test":  17,
			},
			Output: common.MapStr{
				"test": 17,
			},
		},
		{
			Fields: []string{"test", "a.b"},
			Input: common.MapStr{
				"a.b":  "b",
				"a.c":  "c",
				"test": 17,
			},
			Output: common.MapStr{
				"test": 17,
				"a": common.MapStr{
					"b": "b",
				},
			},
		},
	}

	for _, test := range tests {
		p := includeFields{
			Fields: test.Fields,
		}

		event := &beat.Event{
			Fields: test.Input,
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)

		assert.Equal(t, test.Output, newEvent.Fields)
	}
}
