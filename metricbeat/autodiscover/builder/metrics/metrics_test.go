package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		event  bus.Event
		len    int
		result common.MapStr
	}{
		// Empty event hints should return empty config
		{
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"docker": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
			},
			len:    0,
			result: common.MapStr{},
		},
		// Hints without host should return nothing
		{
			event: bus.Event{
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "prometheus",
					},
				},
			},
			len:    0,
			result: common.MapStr{},
		},
		// Only module hint should return empty config
		{
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "prometheus",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module":     "prometheus",
				"metricsets": []interface{}{"collector"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
			},
		},
		// Only module, namespace hint should return empty config
		{
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "prometheus",
						"namespace": "test",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module":     "prometheus",
				"namespace":  "test",
				"metricsets": []interface{}{"collector"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
			},
		},
		// Module, namespace, host hint should return valid config without port should not return hosts
		{
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "prometheus",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module":     "prometheus",
				"namespace":  "test",
				"metricsets": []interface{}{"collector"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
			},
		},
		// Module, namespace, host hint should return valid config
		{
			event: bus.Event{
				"host": "1.2.3.4",
				"port": int64(9090),
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "prometheus",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module":     "prometheus",
				"namespace":  "test",
				"metricsets": []interface{}{"collector"},
				"hosts":      []interface{}{"1.2.3.4:9090"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
			},
		},
	}
	for _, test := range tests {
		cfg := defaultConfig()

		m := metricAnnotations{
			Key: cfg.Key,
		}
		cfgs := m.CreateConfig(test.event)
		assert.Equal(t, len(cfgs), test.len)

		if test.len != 0 {
			config := common.MapStr{}
			err := cfgs[0].Unpack(&config)
			assert.Nil(t, err)

			assert.Equal(t, config, test.result)
		}

	}
}
