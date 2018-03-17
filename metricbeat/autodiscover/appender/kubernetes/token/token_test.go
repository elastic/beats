package token

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

func TestTokenAppender(t *testing.T) {
	tests := []struct {
		eventConfig string
		event       bus.Event
		result      common.MapStr
		config      string
	}{
		// Appender without a condition should apply the config regardless
		// Empty event config should return a config with only the headers
		{
			event: bus.Event{},
			result: common.MapStr{
				"headers": map[string]interface{}{
					"Authorization": "Bearer foo bar",
				},
			},
			eventConfig: "",
			config: `
token_path: "test"
`,
		},
		// Metricbeat module config should return a config that has headers section
		{
			event: bus.Event{},
			result: common.MapStr{
				"module": "prometheus",
				"hosts":  []interface{}{"1.2.3.4:8080"},
				"headers": map[string]interface{}{
					"Authorization": "Bearer foo bar",
				},
			},
			eventConfig: `
module: prometheus
hosts: ["1.2.3.4:8080"]
`,
			config: `
token_path: "test"
`,
		},
	}

	for _, test := range tests {
		config, err := common.NewConfigWithYAML([]byte(test.config), "")
		if err != nil {
			t.Fatal(err)
		}

		eConfig, err := common.NewConfigWithYAML([]byte(test.eventConfig), "")
		if err != nil {
			t.Fatal(err)
		}

		test.event["config"] = []*common.Config{eConfig}
		writeFile("test", "foo bar")

		appender, err := NewTokenAppender(config)
		assert.Nil(t, err)
		assert.NotNil(t, appender)

		appender.Append(test.event)
		cfgs, _ := test.event["config"].([]*common.Config)
		assert.Equal(t, len(cfgs), 1)

		out := common.MapStr{}
		cfgs[0].Unpack(&out)

		assert.Equal(t, out, test.result)
		deleteFile("test")
	}
}

func writeFile(name, message string) {
	ioutil.WriteFile(name, []byte(message), os.ModePerm)
}

func deleteFile(name string) {
	os.Remove(name)
}
