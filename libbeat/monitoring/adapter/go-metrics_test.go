package adapter

import (
	"strings"
	"testing"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/monitoring"
)

func TestGoMetricsAdapter(t *testing.T) {
	filters := []MetricFilter{
		WhitelistIf(func(name string) bool {
			return strings.HasPrefix(name, "mon")
		}),
		ApplyIf(
			func(name string) bool {
				return strings.HasPrefix(name, "ign")
			},
			GoMetricsNilify,
		),
	}

	counters := map[string]int64{
		"mon-counter": 42,
		"ign-counter": 0,
		"counter":     42,
	}
	meters := map[string]int64{
		"mon-meter": 23,
		"ign-meter": 0,
		"meter":     23,
	}

	monReg := monitoring.NewRegistry()
	var reg metrics.Registry = GetGoMetrics(monReg, "test", filters...)

	// register some metrics and check they're satisfying the go-metrics interface
	// no matter if owned by monitoring or go-metrics
	for name := range counters {
		cnt := reg.GetOrRegister(name, func() interface{} {
			return metrics.NewCounter()
		}).(metrics.Counter)
		cnt.Clear()
	}

	for name := range meters {
		meter := reg.GetOrRegister(name, func() interface{} {
			return metrics.NewMeter()
		}).(metrics.Meter)
		meter.Count()
	}

	// get and increase registered metrics
	for name := range counters {
		cnt := reg.Get(name).(metrics.Counter)
		cnt.Inc(21)
		cnt.Inc(21)
	}
	for name := range meters {
		meter := reg.Get(name).(metrics.Meter)
		meter.Mark(11)
		meter.Mark(12)
	}

	// compare metric values to expected values
	for name, value := range counters {
		cnt := reg.Get(name).(metrics.Counter)
		assert.Equal(t, value, cnt.Count())
	}
	for name, value := range meters {
		meter := reg.Get(name).(metrics.Meter)
		assert.Equal(t, value, meter.Count())
	}

	// check Each only returns metrics not registered with monitoring.Registry
	reg.Each(func(name string, v interface{}) {
		if strings.HasPrefix(name, "mon") {
			t.Errorf("metric %v should not have been reported by each", name)
		}
	})
	monReg.Do(monitoring.Full, func(name string, v interface{}) {
		if !strings.HasPrefix(name, "test.mon") {
			t.Errorf("metric %v should not have been reported by each", name)
		}
	})
}
