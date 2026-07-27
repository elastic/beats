// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

var (
	prevSnap = monitoring.FlatSnapshot{
		Ints: map[string]int64{
			"count":        10,
			"gone":         1,
			"active_gauge": 6,
		},
		Floats: map[string]float64{
			"system.load.1":     2.0,
			"float_counter":     1,
			"foo.histogram.p99": 4.0,
		},
	}
	curSnap = monitoring.FlatSnapshot{
		Ints: map[string]int64{
			"count":        20,
			"new":          1,
			"active_gauge": 5,
		},
		Floats: map[string]float64{
			"system.load.1":     1.2,
			"float_counter":     3,
			"foo.histogram.p99": 4.1,
		},
	}
)

// Smoke test.
func TestStartStop(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	r, err := MakeReporter(beat.Info{Logger: logger}, conf.NewConfig(), beatmonitoring.NewGlobalMonitoring())
	if err != nil {
		t.Fatal(err)
	}
	r.Stop()
}

func TestMakeDeltaSnapshot(t *testing.T) {
	delta := makeDeltaSnapshot(prevSnap, curSnap)
	assert.EqualValues(t, 10, delta.Ints["count"])
	assert.EqualValues(t, 1, delta.Ints["new"])
	assert.InDelta(t, 1.2, delta.Floats["system.load.1"], 0.001)
	assert.InDelta(t, 2, delta.Floats["float_counter"], 0.001)
	assert.EqualValues(t, 5, delta.Ints["active_gauge"])
	assert.InDelta(t, 4.1, delta.Floats["foo.histogram.p99"], 0.001)
	assert.NotContains(t, delta.Ints, "gone")
}

func TestReporterLog(t *testing.T) {
	logger, zapLogs := logptest.NewTestingLoggerWithObserver(t, "")

	reporter := Reporter{config: defaultConfig(), logger: logger.Named("monitoring")}

	reporter.logSnapshot(map[string]monitoring.FlatSnapshot{})
	logs := zapLogs.TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, "No non-zero metrics in the last 30s", logs[0].Message)
	}

	reporter.logSnapshot(
		map[string]monitoring.FlatSnapshot{
			"metrics": {
				Bools: map[string]bool{
					"running": true,
				},
			},
		},
	)
	logs = zapLogs.TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, "Non-zero metrics in the last 30s", logs[0].Message)
		assertMapHas(t, logs[0].ContextMap(), "monitoring.metrics.running", true)
	}

	reporter.logTotals(map[string]monitoring.FlatSnapshot{"metrics": curSnap})
	logs = zapLogs.TakeAll()
	if assert.Len(t, logs, 2) {
		assert.Equal(t, "Total metrics", logs[0].Message)
		assertMapHas(t, logs[0].ContextMap(), "monitoring.metrics.count", 20)
		assertMapHas(t, logs[0].ContextMap(), "monitoring.metrics.new", 1)
		assert.Contains(t, logs[1].Message, "Uptime: ")
	}
}

func TestZeroPeriodSkipsLogging(t *testing.T) {
	logger, zapLogs := logptest.NewTestingLoggerWithObserver(t, "")

	r := &Reporter{
		config:     config{Period: 0},
		done:       make(chan struct{}),
		logger:     logger.Named("monitoring"),
		registries: map[string]*monitoring.Registry{},
	}

	r.wg.Go(func() {
		r.snapshotLoop()
	})

	// The goroutine should exit immediately when Period == 0.
	exited := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(exited)
	}()
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("snapshotLoop goroutine did not exit within 5s for zero period")
	}

	// No periodic metrics log lines should have been emitted.
	for _, log := range zapLogs.TakeAll() {
		assert.NotContains(t, log.Message, "Starting metrics logging")
		assert.NotContains(t, log.Message, "Non-zero metrics")
		assert.NotContains(t, log.Message, "No non-zero metrics")
		assert.NotContains(t, log.Message, "Total metrics")
		assert.Contains(t, log.Message, "Skipping metrics logging")
	}
}

// TestZeroPeriodConfig verifies that a config with period=0 does not panic
// (time.NewTicker panics on a zero duration) and that Period is parsed as 0.
func TestZeroPeriodConfig(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	cfg, err := conf.NewConfigFrom(map[string]any{
		"period": "0s",
	})
	if err != nil {
		t.Fatal(err)
	}

	rep, err := MakeReporter(beat.Info{Logger: logger}, cfg, beatmonitoring.NewGlobalMonitoring())
	if err != nil {
		t.Fatal(err)
	}
	defer rep.Stop()

	reporter, ok := rep.(*Reporter)
	if !ok {
		t.Fatal("MakeReporter did not return a *Reporter")
	}
	assert.Equal(t, time.Duration(0), reporter.Period)
}

func assertMapHas(t *testing.T, m map[string]any, key string, expectedValue any) {
	t.Helper()
	v, err := mapstr.M(m).GetValue(key)
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, expectedValue, v)
}
