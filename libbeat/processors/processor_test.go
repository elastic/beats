package processors_test

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	_ "github.com/elastic/beats/libbeat/processors/actions"
	"github.com/stretchr/testify/assert"
)

func GetProcessors(t *testing.T, yml []map[string]interface{}) *processors.Processors {

	config := processors.PluginConfig{}

	for _, action := range yml {
		c := map[string]common.Config{}

		for name, actionYml := range action {
			actionConfig, err := common.NewConfigFrom(actionYml)
			assert.Nil(t, err)

			c[name] = *actionConfig
		}
		config = append(config, c)

	}

	list, err := processors.New(config)
	assert.Nil(t, err)

	return list

}

func TestBadConfig(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		map[string]interface{}{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"contains": map[string]string{
						"proc.name": "test",
					},
				},
				"fields": []string{"proc.cpu.total_p", "proc.mem", "dd"},
			},
			"drop_fields": map[string]interface{}{
				"fields": []string{"proc.cpu"},
			},
		},
	}

	config := processors.PluginConfig{}

	for _, action := range yml {
		c := map[string]common.Config{}

		for name, actionYml := range action {
			actionConfig, err := common.NewConfigFrom(actionYml)
			assert.Nil(t, err)

			c[name] = *actionConfig
		}
		config = append(config, c)
	}

	_, err := processors.New(config)
	assert.NotNil(t, err)

}

func TestIncludeFields(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		map[string]interface{}{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"contains": map[string]string{
						"proc.name": "test",
					},
				},
				"fields": []string{"proc.cpu.total_p", "proc.mem", "dd"},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"name":    "test-1",
			"cmdline": "/sbin/launchd",
			"mem": common.MapStr{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  2555572224,
			},
		},
		"type": "process",
	}

	processedEvent := processors.Run(event)

	expectedEvent := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"total_p": 0,
			},
			"mem": common.MapStr{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  2555572224,
			},
		},
		"type": "process",
	}

	assert.Equal(t, expectedEvent, processedEvent)
}

func TestIncludeFields1(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		map[string]interface{}{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"regexp": map[string]string{
						"proc.cmdline": "launchd",
					},
				},
				"fields": []string{"proc.cpu.total_add"},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},

		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"cmdline": "/sbin/launchd",
			"mem": common.MapStr{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  2555572224,
			},
		},
		"type": "process",
	}

	processedEvent := processors.Run(event)

	expectedEvent := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"type":       "process",
	}

	assert.Equal(t, expectedEvent, processedEvent)
}

func TestDropFields(t *testing.T) {

	yml := []map[string]interface{}{
		map[string]interface{}{
			"drop_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"equals": map[string]string{
						"beat.hostname": "mar",
					},
				},
				"fields": []string{"proc.cpu.start_time", "mem", "proc.cmdline", "beat", "dd"},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},

		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"cmdline": "/sbin/launchd",
		},
		"mem": common.MapStr{
			"rss":   11194368,
			"rss_p": 0,
			"share": 0,
			"size":  2555572224,
		},
		"type": "process",
	}

	processedEvent := processors.Run(event)

	expectedEvent := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"system":  26027,
				"total":   79390,
				"total_p": 0,
				"user":    53363,
			},
		},
		"type": "process",
	}

	assert.Equal(t, expectedEvent, processedEvent)
}

func TestMultipleIncludeFields(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		map[string]interface{}{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"contains": map[string]string{
						"beat.name": "my-shipper",
					},
				},
				"fields": []string{"proc"},
			},
		},
		map[string]interface{}{
			"include_fields": map[string]interface{}{
				"fields": []string{"proc.cpu.start_time", "proc.cpu.total_p", "proc.mem.rss_p", "proc.cmdline"},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event1 := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},

		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"cmdline": "/sbin/launchd",
		},
		"mem": common.MapStr{
			"rss":   11194368,
			"rss_p": 0,
			"share": 0,
			"size":  2555572224,
		},
		"type": "process",
	}

	event2 := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},
		"fs": common.MapStr{
			"device_name": "devfs",
			"total":       198656,
			"used":        198656,
			"used_p":      1,
			"free":        0,
			"avail":       0,
			"files":       677,
			"free_files":  0,
			"mount_point": "/dev",
		},
		"type": "process",
	}

	expected1 := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"total_p":    0,
			},
			"cmdline": "/sbin/launchd",
		},

		"type": "process",
	}

	expected2 := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"type":       "process",
	}

	actual1 := processors.Run(event1)
	actual2 := processors.Run(event2)

	assert.Equal(t, expected1, actual1)
	assert.Equal(t, expected2, actual2)
}

func TestDropEvent(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		map[string]interface{}{
			"drop_event": map[string]interface{}{
				"when": map[string]interface{}{
					"range": map[string]interface{}{
						"proc.cpu.total_p": map[string]float64{
							"lt": 0.5,
						},
					},
				},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"name":    "test-1",
			"cmdline": "/sbin/launchd",
			"mem": common.MapStr{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  2555572224,
			},
		},
		"type": "process",
	}

	processedEvent := processors.Run(event)

	assert.Nil(t, processedEvent)
}

func TestEmptyCondition(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		map[string]interface{}{
			"drop_event": map[string]interface{}{},
		},
	}

	processors := GetProcessors(t, yml)

	event := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"name":    "test-1",
			"cmdline": "/sbin/launchd",
			"mem": common.MapStr{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  2555572224,
			},
		},
		"type": "process",
	}

	processedEvent := processors.Run(event)

	assert.Nil(t, processedEvent)
}

func TestBadCondition(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		map[string]interface{}{
			"drop_event": map[string]interface{}{
				"when": map[string]interface{}{
					"equal": map[string]string{
						"type": "process",
					},
				},
			},
		},
	}

	config := processors.PluginConfig{}

	for _, action := range yml {
		c := map[string]common.Config{}

		for name, actionYml := range action {
			actionConfig, err := common.NewConfigFrom(actionYml)
			if err != nil {
				t.Fatal(err)
			}

			c[name] = *actionConfig
		}
		config = append(config, c)
	}

	_, err := processors.New(config)
	assert.Error(t, err)

}

func TestMissingFields(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		map[string]interface{}{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"equals": map[string]string{
						"type": "process",
					},
				},
			},
		},
	}

	config := processors.PluginConfig{}

	for _, action := range yml {
		c := map[string]common.Config{}

		for name, actionYml := range action {
			actionConfig, err := common.NewConfigFrom(actionYml)
			assert.Nil(t, err)

			c[name] = *actionConfig
		}
		config = append(config, c)
	}

	_, err := processors.New(config)
	assert.NotNil(t, err)

}

func TestBadConditionConfig(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		map[string]interface{}{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"fake": map[string]string{
						"type": "process",
					},
				},
				"fields": []string{"proc.cpu.start_time", "proc.cpu.total_p", "proc.mem.rss_p", "proc.cmdline"},
			},
		},
	}

	config := processors.PluginConfig{}

	for _, action := range yml {
		c := map[string]common.Config{}

		for name, actionYml := range action {
			actionConfig, err := common.NewConfigFrom(actionYml)
			assert.Nil(t, err)

			c[name] = *actionConfig
		}
		config = append(config, c)
	}

	_, err := processors.New(config)
	assert.NotNil(t, err)

}
