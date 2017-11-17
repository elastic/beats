package template

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"

	"github.com/stretchr/testify/assert"
)

func TestConfigsMapping(t *testing.T) {
	config, _ := common.NewConfigFrom(map[string]interface{}{
		"correct": "config",
	})

	tests := []struct {
		mapping  string
		event    bus.Event
		expected []*common.Config
	}{
		// No match
		{
			mapping: `
- condition.equals:
    foo: 3
  config:
  - type: config1`,
			event: bus.Event{
				"foo": "no match",
			},
			expected: nil,
		},
		// Match config
		{
			mapping: `
- condition.equals:
    foo: 3
  config:
  - correct: config`,
			event: bus.Event{
				"foo": 3,
			},
			expected: []*common.Config{config},
		},
	}

	for _, test := range tests {
		var mappings MapperSettings
		config, err := common.NewConfigWithYAML([]byte(test.mapping), "")
		if err != nil {
			t.Fatal(err)
		}

		if err := config.Unpack(&mappings); err != nil {
			t.Fatal(err)
		}

		mapper, err := NewConfigMapper(mappings)
		if err != nil {
			t.Fatal(err)
		}

		res := mapper.GetConfig(test.event)
		assert.Equal(t, test.expected, res)
	}
}
