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

package adapter

import (
	"strings"
	"testing"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
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
	var reg metrics.Registry = GetGoMetrics(monReg, "test", logp.NewNopLogger(), filters...)

	// register some metrics and check they're satisfying the go-metrics interface
	// no matter if owned by monitoring or go-metrics
	for name := range counters {
		cnt, ok := reg.GetOrRegister(name, func() any {
			return metrics.NewCounter()
		}).(metrics.Counter)
		require.True(t, ok)
		cnt.Clear()
	}

	for name := range meters {
		meter, ok := reg.GetOrRegister(name, func() any {
			return metrics.NewMeter()
		}).(metrics.Meter)
		require.True(t, ok)
		meter.Count()
	}

	// get and increase registered metrics
	for name := range counters {
		cnt, ok := reg.Get(name).(metrics.Counter)
		require.True(t, ok)
		cnt.Inc(21)
		cnt.Inc(21)
	}
	for name := range meters {
		meter, ok := reg.Get(name).(metrics.Meter)
		require.True(t, ok)
		meter.Mark(11)
		meter.Mark(12)
	}

	// compare metric values to expected values
	for name, value := range counters {
		cnt, ok := reg.Get(name).(metrics.Counter)
		require.True(t, ok)
		assert.Equal(t, value, cnt.Count())
	}
	for name, value := range meters {
		meter, ok := reg.Get(name).(metrics.Meter)
		require.True(t, ok)
		assert.Equal(t, value, meter.Count())
	}

	// check Each only returns metrics not registered with monitoring.Registry
	reg.Each(func(name string, v any) {
		if strings.HasPrefix(name, "mon") {
			t.Errorf("metric %v should not have been reported by each", name)
		}
	})
	monReg.Do(monitoring.Full, func(name string, v any) {
		if !strings.HasPrefix(name, "test.mon") {
			t.Errorf("metric %v should not have been reported by each", name)
		}
	})
}

func TestGoMetricsHistogramClearOnVisit(t *testing.T) {
	monReg := monitoring.NewRegistry()
	histogramSample := metrics.NewUniformSample(10)
	clearedHistogramSample := metrics.NewUniformSample(10)
	_ = NewGoMetrics(monReg, "original", logp.NewNopLogger(), Accept).Register("histogram", metrics.NewHistogram(histogramSample))
	_ = NewGoMetrics(monReg, "cleared", logp.NewNopLogger(), Accept).Register("histogram", NewClearOnVisitHistogram(clearedHistogramSample))
	dataPoints := [...]int{2, 4, 8, 4, 2}
	dataPointsMedian := 4.0
	for _, i := range dataPoints {
		histogramSample.Update(int64(i))
		clearedHistogramSample.Update(int64(i))
	}

	preSnapshot := []struct {
		expected any
		actual   any
		msg      string
	}{
		{
			actual:   histogramSample.Count(),
			expected: int64(len(dataPoints)),
			msg:      "histogram sample count incorrect",
		},
		{
			actual:   clearedHistogramSample.Count(),
			expected: int64(len(dataPoints)),
			msg:      "cleared histogram sample count incorrect",
		},
		{
			actual:   histogramSample.Percentiles([]float64{0.5}),
			expected: []float64{dataPointsMedian},
			msg:      "histogram median incorrect",
		},
		{
			actual:   clearedHistogramSample.Percentiles([]float64{0.5}),
			expected: []float64{dataPointsMedian},
			msg:      "cleared histogram median incorrect",
		},
	}

	for _, tc := range preSnapshot {
		require.Equal(t, tc.expected, tc.actual, tc.msg)
	}

	// collecting the snapshot triggers the visit of each histogram
	flatSnapshot := monitoring.CollectFlatSnapshot(monReg, monitoring.Full, false)

	// Check to make sure after the snapshot the samples in
	// clearedHistogramSample have been reset to zero, but the
	// snapshot reports the values before the clear

	postSnapshot := []struct {
		expected any
		actual   any
		msg      string
	}{
		{
			actual:   histogramSample.Count(),
			expected: int64(len(dataPoints)),
			msg:      "histogram sample count incorrect",
		},
		{
			actual:   clearedHistogramSample.Count(),
			expected: int64(0),
			msg:      "cleared histogram sample count incorrect",
		},
		{
			actual:   histogramSample.Percentiles([]float64{0.5}),
			expected: []float64{dataPointsMedian},
			msg:      "histogram median incorrect",
		},
		{
			actual:   clearedHistogramSample.Percentiles([]float64{0.5}),
			expected: []float64{0},
			msg:      "cleared histogram median incorrect",
		},
		{
			actual:   flatSnapshot.Ints["original.histogram.count"],
			expected: int64(len(dataPoints)),
			msg:      "visited histogram count is wrong",
		},
		{
			actual:   flatSnapshot.Ints["cleared.histogram.count"],
			expected: int64(len(dataPoints)),
			msg:      "visited cleared histogram count is wrong",
		},
		{
			actual:   flatSnapshot.Floats["original.histogram.median"],
			expected: float64(dataPointsMedian),
			msg:      "visited histogram median is wrong",
		},
		{
			actual:   flatSnapshot.Floats["cleared.histogram.median"],
			expected: float64(dataPointsMedian),
			msg:      "visited cleared histogram median is wrong",
		},
	}

	for _, tc := range postSnapshot {
		require.Equal(t, tc.expected, tc.actual, tc.msg)
	}
}
