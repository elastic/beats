package hints

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/metricbeat/mb"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		message string
		event   bus.Event
		len     int
		result  common.MapStr
	}{
		{
			message: "Empty event hints should return empty config",
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
		{
			message: "Hints without host should return nothing",
			event: bus.Event{
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "mockmodule",
					},
				},
			},
			len:    0,
			result: common.MapStr{},
		},
		{
			message: "Only module hint should return all metricsets",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "mockmodule",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module":     "mockmodule",
				"metricsets": []interface{}{"one", "two"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
			},
		},
		{
			message: "metricsets hint works",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":     "mockmodule",
						"metricsets": "one",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module":     "mockmodule",
				"metricsets": []interface{}{"one"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
			},
		},
		{
			message: "Only module, it should return defaults",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "mockmoduledefaults",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module":     "mockmoduledefaults",
				"metricsets": []interface{}{"default"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
			},
		},
		{
			message: "Module, namespace, host hint should return valid config without port should not return hosts",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module":     "mockmoduledefaults",
				"namespace":  "test",
				"metricsets": []interface{}{"default"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
			},
		},
		{
			message: "Module, namespace, host hint should return valid config",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module":     "mockmoduledefaults",
				"namespace":  "test",
				"metricsets": []interface{}{"default"},
				"hosts":      []interface{}{"1.2.3.4:9090"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
			},
		},
	}
	for _, test := range tests {
		mockRegister := mb.NewRegister()
		mockRegister.MustAddMetricSet("mockmodule", "one", NewMockMetricSet, mb.DefaultMetricSet())
		mockRegister.MustAddMetricSet("mockmodule", "two", NewMockMetricSet, mb.DefaultMetricSet())
		mockRegister.MustAddMetricSet("mockmoduledefaults", "default", NewMockMetricSet, mb.DefaultMetricSet())
		mockRegister.MustAddMetricSet("mockmoduledefaults", "other", NewMockMetricSet)

		m := metricHints{
			Key:      defaultConfig().Key,
			Registry: mockRegister,
		}
		cfgs := m.CreateConfig(test.event)
		assert.Equal(t, len(cfgs), test.len)

		if test.len != 0 {
			config := common.MapStr{}
			err := cfgs[0].Unpack(&config)
			assert.Nil(t, err, test.message)

			assert.Equal(t, config, test.result, test.message)
		}

	}
}

type MockMetricSet struct {
	mb.BaseMetricSet
}

func NewMockMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MockMetricSet{}, nil
}

func (ms *MockMetricSet) Fetch(report mb.Reporter) {

}
