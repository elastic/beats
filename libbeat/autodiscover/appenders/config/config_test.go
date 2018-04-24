package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

func TestGenerateAppender(t *testing.T) {
	tests := []struct {
		eventConfig common.MapStr
		event       bus.Event
		result      common.MapStr
		config      string
	}{
		// Appender without a condition should apply the config regardless
		{
			event: bus.Event{},
			result: common.MapStr{
				"test":  "bar",
				"test1": "foo",
				"test2": "foo",
			},
			eventConfig: common.MapStr{
				"test": "bar",
			},
			config: `
- config: 
    "test1": foo 
- config: 
    "test2": foo 
`,
		},
		// Appender with a condition check that fails. Only appender with no condition should pass
		{
			event: bus.Event{
				"foo": "bar",
			},
			result: common.MapStr{
				"test":  "bar",
				"test1": "foo",
			},
			eventConfig: common.MapStr{
				"test": "bar",
			},
			config: `
- config: 
    "test1": foo 
- config: 
    "test2": foo 
  condition.equals:
    "foo": "bar1"
`,
		},
		// Appender with a condition check that passes. It should get appended
		{
			event: bus.Event{
				"foo": "bar",
			},
			result: common.MapStr{
				"test":  "bar",
				"test1": "foo",
				"test2": "foo",
			},
			eventConfig: common.MapStr{
				"test": "bar",
			},
			config: `
- config: 
    "test1": foo 
- config: 
    "test2": foo 
  condition.equals:
    "foo": "bar"
`,
		},
	}
	for _, test := range tests {
		config, err := common.NewConfigWithYAML([]byte(test.config), "")
		if err != nil {
			t.Fatal(err)
		}

		appender, err := NewConfigAppender(config)
		assert.Nil(t, err)
		assert.NotNil(t, appender)

		eveConfig, err := common.NewConfigFrom(&test.eventConfig)
		assert.Nil(t, err)

		test.event["config"] = []*common.Config{eveConfig}
		appender.Append(test.event)

		cfgs, _ := test.event["config"].([]*common.Config)
		assert.Equal(t, len(cfgs), 1)

		out := common.MapStr{}
		cfgs[0].Unpack(&out)

		assert.Equal(t, out, test.result)

	}
}
