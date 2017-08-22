// +build !integration

package server

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func GetMetricProcessor() *metricProcessor {
	templates := []TemplateConfig{
		{
			Namespace: "foo",
			Filter:    "test.localhost.*",
			Template:  ".host.shell.metric",
			Delimiter: ".",
		},
		{
			Namespace: "foo",
			Filter:    "test.xyz.*",
			Template:  ".host.metric*",
			Delimiter: "_",
			Tags: map[string]string{
				"a": "b",
			},
		},
	}

	defaultTemplate := DefaultGraphiteCollectorConfig().DefaultTemplate
	return NewMetricProcessor(templates, defaultTemplate)
}

func TestMetricProcessorAddTemplate(t *testing.T) {
	processor := GetMetricProcessor()
	temp := TemplateConfig{
		Namespace: "xyz",
		Filter:    "a.b.*",
		Template:  ".host.shell.metric",
		Delimiter: ".",
	}
	processor.AddTemplate(temp)
	out := processor.templates.Search([]string{"a", "b", "c"})
	assert.NotNil(t, out)
	assert.Equal(t, out.Namespace, temp.Namespace)
}

func TestMetricProcessorDeleteTemplate(t *testing.T) {
	processor := GetMetricProcessor()
	temp := TemplateConfig{
		Namespace: "xyz",
		Filter:    "a.b.*",
		Template:  ".host.shell.metric",
		Delimiter: ".",
	}
	processor.AddTemplate(temp)
	processor.RemoveTemplate(temp)
	out := processor.templates.Search([]string{"a", "b", "c"})
	assert.Nil(t, out)

}

func TestMetricProcessorProcess(t *testing.T) {
	processor := GetMetricProcessor()
	event, err := processor.Process("test.localhost.bash.stats 42 1500934723")
	assert.Nil(t, err)
	assert.NotNil(t, event)

	tag := event["tag"].(common.MapStr)
	assert.Equal(t, len(tag), 2)
	assert.Equal(t, tag["host"], "localhost")
	assert.Equal(t, tag["shell"], "bash")

	assert.NotNil(t, event["stats"])
	assert.Equal(t, event["stats"], float64(42))

	ts := float64(1500934723)
	timestamp := common.Time(time.Unix(int64(ts), int64((ts-math.Floor(ts))*float64(time.Second))))

	assert.Equal(t, event["@timestamp"], timestamp)

	event, err = processor.Process("test.localhost.bash.stats 42")
	assert.Nil(t, err)
	assert.NotNil(t, event)

	assert.NotNil(t, event["stats"])
	assert.Equal(t, event["stats"], float64(42))
}
