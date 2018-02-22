package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

var (
	prevSnap = monitoring.FlatSnapshot{
		Ints: map[string]int64{
			"count": 10,
			"gone":  1,
		},
		Floats: map[string]float64{
			"system.load.1": 2.0,
			"float_counter": 1,
		},
	}
	curSnap = monitoring.FlatSnapshot{
		Ints: map[string]int64{
			"count": 20,
			"new":   1,
		},
		Floats: map[string]float64{
			"system.load.1": 1.2,
			"float_counter": 3,
		},
	}
)

// Smoke test.
func TestStartStop(t *testing.T) {
	r, err := MakeReporter(beat.Info{}, common.NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	r.Stop()
}

func TestMakeDeltaSnapshot(t *testing.T) {
	delta := makeDeltaSnapshot(prevSnap, curSnap)
	assert.EqualValues(t, 10, delta.Ints["count"])
	assert.EqualValues(t, 1, delta.Ints["new"])
	assert.EqualValues(t, 1.2, delta.Floats["system.load.1"])
	assert.EqualValues(t, 2, delta.Floats["float_counter"])
	assert.NotContains(t, delta.Ints, "gone")
}

func TestReporterLog(t *testing.T) {
	logp.DevelopmentSetup(logp.ToObserverOutput())
	reporter := reporter{period: 30 * time.Second, logger: logp.NewLogger("monitoring")}

	reporter.logSnapshot(monitoring.FlatSnapshot{})
	logs := logp.ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, "No non-zero metrics in the last 30s", logs[0].Message)
	}

	reporter.logSnapshot(
		monitoring.FlatSnapshot{
			Bools: map[string]bool{
				"running": true,
			},
		},
	)
	logs = logp.ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, "Non-zero metrics in the last 30s", logs[0].Message)
		assertMapHas(t, logs[0].ContextMap(), "monitoring.metrics.running", true)
	}

	reporter.logTotals(curSnap)
	logs = logp.ObserverLogs().TakeAll()
	if assert.Len(t, logs, 2) {
		assert.Equal(t, "Total non-zero metrics", logs[0].Message)
		assertMapHas(t, logs[0].ContextMap(), "monitoring.metrics.count", 20)
		assertMapHas(t, logs[0].ContextMap(), "monitoring.metrics.new", 1)
		assert.Contains(t, logs[1].Message, "Uptime: ")
	}
}

func assertMapHas(t *testing.T, m map[string]interface{}, key string, expectedValue interface{}) {
	t.Helper()
	v, err := common.MapStr(m).GetValue(key)
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, expectedValue, v)
}
