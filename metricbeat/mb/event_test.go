// +build !integration

package mb

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestEventConversionToBeatEvent(t *testing.T) {
	var (
		timestamp = time.Now()
		module    = "docker"
		metricSet = "uptime"
	)

	t.Run("all levels", func(t *testing.T) {
		e := (&Event{
			Timestamp: timestamp,
			RootFields: common.MapStr{
				"type": "docker",
			},
			ModuleFields: common.MapStr{
				"container": common.MapStr{
					"name": "wordpress",
				},
			},
			MetricSetFields: common.MapStr{
				"ms": 1000,
			},
		}).BeatEvent(module, metricSet)

		assert.Equal(t, timestamp, e.Timestamp)
		assert.Equal(t, common.MapStr{
			"type": "docker",
			"docker": common.MapStr{
				"container": common.MapStr{
					"name": "wordpress",
				},
				"uptime": common.MapStr{
					"ms": 1000,
				},
			},
		}, e.Fields)
	})

	t.Run("idempotent", func(t *testing.T) {
		mbEvent := &Event{
			Timestamp: timestamp,
			RootFields: common.MapStr{
				"type": "docker",
			},
			ModuleFields: common.MapStr{
				"container": common.MapStr{
					"name": "wordpress",
				},
			},
			MetricSetFields: common.MapStr{
				"ms": 1000,
			},
		}
		e := mbEvent.BeatEvent(module, metricSet)
		e = mbEvent.BeatEvent(module, metricSet)

		assert.Equal(t, timestamp, e.Timestamp)
		assert.Equal(t, common.MapStr{
			"type": "docker",
			"docker": common.MapStr{
				"container": common.MapStr{
					"name": "wordpress",
				},
				"uptime": common.MapStr{
					"ms": 1000,
				},
			},
		}, e.Fields)
	})

	t.Run("with event modifiers", func(t *testing.T) {
		modifier := func(m, ms string, e *Event) {
			e.RootFields.Put("module", m)
			e.RootFields.Put("metricset", ms)
		}

		e := (&Event{}).BeatEvent(module, metricSet, modifier)
		assert.Equal(t, common.MapStr{
			"module":    module,
			"metricset": metricSet,
		}, e.Fields)
	})

	t.Run("error message", func(t *testing.T) {
		msg := "something failed"
		e := (&Event{
			Error: errors.New(msg),
		}).BeatEvent(module, metricSet)

		errorMessage, err := e.Fields.GetValue("error.message")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, msg, errorMessage)
	})
}

func TestAddMetricSetInfo(t *testing.T) {
	const (
		host    = "localhost"
		elapsed = time.Duration(500 * time.Millisecond)
	)

	t.Run("all fields", func(t *testing.T) {
		e := Event{
			Host: host,
			Took: elapsed,
		}

		AddMetricSetInfo(moduleName, metricSetName, &e)

		assert.Equal(t, common.MapStr{
			"metricset": common.MapStr{
				"host":   host,
				"module": moduleName,
				"name":   metricSetName,
				"rtt":    time.Duration(500000),
			},
		}, e.RootFields)
	})

	t.Run("no optional fields", func(t *testing.T) {
		e := Event{}

		AddMetricSetInfo(moduleName, metricSetName, &e)

		assert.Equal(t, common.MapStr{
			"metricset": common.MapStr{
				"module": moduleName,
				"name":   metricSetName,
			},
		}, e.RootFields)
	})
}

func TestTransformMapStrToEvent(t *testing.T) {
	var (
		timestamp  = time.Now()
		took       = time.Duration(1)
		moduleData = common.MapStr{
			"container_id": "busybox",
		}
		metricSetData = common.MapStr{
			"uptime": "1 day",
		}
		failure = errors.New("failed")
	)

	m := common.MapStr{
		TimestampKey:  timestamp,
		RTTKey:        took,
		ModuleDataKey: moduleData,
	}
	m.DeepUpdate(metricSetData)

	t.Run("normal", func(t *testing.T) {
		m := m.Clone()
		e := TransformMapStrToEvent("module", m, failure)

		assert.Equal(t, timestamp, e.Timestamp)
		assert.Equal(t, took, e.Took)
		assert.Empty(t, e.RootFields)
		assert.Equal(t, moduleData, e.ModuleFields)
		assert.Equal(t, metricSetData, e.MetricSetFields)
		assert.Equal(t, failure, e.Error)
	})

	t.Run("namespace", func(t *testing.T) {
		const namespace = "foo.bar"

		mapWithNamespace := m.Clone()
		mapWithNamespace.Put(NamespaceKey, namespace)

		e := TransformMapStrToEvent("module", mapWithNamespace, nil)

		assert.Equal(t, timestamp, e.Timestamp)
		assert.Equal(t, took, e.Took)
		assert.Empty(t, e.RootFields)
		assert.Equal(t, moduleData, e.ModuleFields)
		assert.Equal(t, "module."+namespace, e.Namespace)
		assert.Equal(t, metricSetData, e.MetricSetFields)
	})
}
