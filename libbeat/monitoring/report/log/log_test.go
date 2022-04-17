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

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/monitoring"
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
	assert.EqualValues(t, 5, delta.Ints["active_gauge"])
	assert.EqualValues(t, 4.1, delta.Floats["foo.histogram.p99"])
	assert.NotContains(t, delta.Ints, "gone")
}

func TestReporterLog(t *testing.T) {
	logp.DevelopmentSetup(logp.ToObserverOutput())
	reporter := reporter{config: defaultConfig(), logger: logp.NewLogger("monitoring")}

	reporter.logSnapshot(map[string]monitoring.FlatSnapshot{})
	logs := logp.ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, "No non-zero metrics in the last 30s", logs[0].Message)
	}

	reporter.logSnapshot(
		map[string]monitoring.FlatSnapshot{
			"metrics": monitoring.FlatSnapshot{
				Bools: map[string]bool{
					"running": true,
				},
			},
		},
	)
	logs = logp.ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, "Non-zero metrics in the last 30s", logs[0].Message)
		assertMapHas(t, logs[0].ContextMap(), "monitoring.metrics.running", true)
	}

	reporter.logTotals(map[string]monitoring.FlatSnapshot{"metrics": curSnap})
	logs = logp.ObserverLogs().TakeAll()
	if assert.Len(t, logs, 2) {
		assert.Equal(t, "Total metrics", logs[0].Message)
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
