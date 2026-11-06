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

package outputs

import (
	"sync"
	"testing"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/monitoring"
	"github.com/elastic/beats/v7/pkg/monitoring/adapter"
)

func TestLazySampleEmpty(t *testing.T) {
	s := newLazySample(16)

	assert.Nil(t, s.get(), "reservoir must not be allocated before the first Update")
	assert.Zero(t, s.Count(), "empty sample count")
	assert.Zero(t, s.Size(), "empty sample size")
	assert.Zero(t, s.Max(), "empty sample max")
	assert.Zero(t, s.Min(), "empty sample min")
	assert.Zero(t, s.Sum(), "empty sample sum")
	assert.Zero(t, s.Mean(), "empty sample mean")
	assert.Zero(t, s.StdDev(), "empty sample standard deviation")
	assert.Zero(t, s.Variance(), "empty sample variance")
	assert.Empty(t, s.Values(), "empty sample values")
	assert.Equal(t, []float64{0, 0}, s.Percentiles([]float64{0.5, 0.99}), "empty sample percentiles")
	assert.Zero(t, s.Percentile(0.5), "empty sample percentile")
	assert.Zero(t, s.Snapshot().Count(), "empty snapshot count")

	s.Clear()
	assert.Nil(t, s.get(), "Clear on an empty sample must not allocate the reservoir")
}

func TestLazySampleRecords(t *testing.T) {
	s := newLazySample(16)
	for _, v := range []int64{1, 2, 3, 4} {
		s.Update(v)
	}

	assert.NotNil(t, s.get(), "reservoir must exist after Update")
	assert.Equal(t, int64(4), s.Count(), "count")
	assert.Equal(t, int64(4), s.Max(), "max")
	assert.Equal(t, int64(1), s.Min(), "min")
	assert.Equal(t, int64(10), s.Sum(), "sum")
	assert.InDelta(t, 2.5, s.Mean(), 0.0001, "mean")
	assert.Equal(t, int64(4), s.Snapshot().Count(), "snapshot count")

	s.Clear()
	assert.Zero(t, s.Count(), "count after Clear")
}

// A lazy sample must report exactly what an eagerly allocated one does.
func TestLazySampleMatchesUniformSample(t *testing.T) {
	lazy := newLazySample(8)
	eager := metrics.NewUniformSample(8)

	same := func(stage string) {
		assert.Equal(t, eager.Count(), lazy.Count(), "%s: count", stage)
		assert.Equal(t, eager.Size(), lazy.Size(), "%s: size", stage)
		assert.Equal(t, eager.Max(), lazy.Max(), "%s: max", stage)
		assert.Equal(t, eager.Min(), lazy.Min(), "%s: min", stage)
		assert.Equal(t, eager.Sum(), lazy.Sum(), "%s: sum", stage)
		assert.Equal(t, eager.Mean(), lazy.Mean(), "%s: mean", stage)
		assert.Equal(t, eager.StdDev(), lazy.StdDev(), "%s: standard deviation", stage)
		assert.Equal(t, eager.Variance(), lazy.Variance(), "%s: variance", stage)
	}

	same("empty")
	for _, v := range []int64{5, 1, 9, 3} { // fewer values than the reservoir, so sampling is deterministic
		lazy.Update(v)
		eager.Update(v)
	}
	same("after updates")
	lazy.Clear()
	eager.Clear()
	same("after clear")
}

func TestLazySampleConcurrent(t *testing.T) {
	s := newLazySample(64)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				s.Update(int64(i))
				_ = s.Snapshot().Count()
				_ = s.Count()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(8*200), s.Count(), "every update must be counted")
}

// The write.latency histograms must stay in the registry, with the same keys
// and values, whether or not anything reported a latency.
func TestStatsLatencyMetricsUnchanged(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	snapshotKeys := func(report bool) map[string]float64 {
		reg := monitoring.NewRegistry()
		stats := NewStats(reg, logger)
		if report {
			stats.ReportLatency(5 * time.Millisecond)
			stats.ReportLatency(15 * time.Millisecond)
		}
		snap := monitoring.CollectFlatSnapshot(reg, monitoring.Full, false)
		out := map[string]float64{}
		for k, v := range snap.Floats {
			out[k] = v
		}
		for k, v := range snap.Ints {
			out[k] = float64(v)
		}
		return out
	}

	// What an eagerly allocated sample would have reported.
	eagerKeys := func(report bool) map[string]float64 {
		reg := monitoring.NewRegistry()
		lifetime, delta := metrics.NewUniformSample(1024), metrics.NewUniformSample(1024)
		require.NoError(t, adapter.NewGoMetrics(reg, "write.latency", logger, adapter.Accept).Register("histogram", metrics.NewHistogram(lifetime)))
		require.NoError(t, adapter.NewGoMetrics(reg, "write.latency_delta", logger, adapter.Accept).Register("histogram", adapter.NewClearOnVisitHistogram(delta)))
		if report {
			for _, v := range []int64{5, 15} {
				lifetime.Update(v)
				delta.Update(v)
			}
		}
		snap := monitoring.CollectFlatSnapshot(reg, monitoring.Full, false)
		out := map[string]float64{}
		for k, v := range snap.Floats {
			out[k] = v
		}
		for k, v := range snap.Ints {
			out[k] = float64(v)
		}
		return out
	}

	for _, report := range []bool{false, true} {
		got := snapshotKeys(report)
		want := eagerKeys(report)
		for k, v := range want {
			if len(k) < len("write.latency") || k[:len("write.latency")] != "write.latency" {
				continue
			}
			assert.Contains(t, got, k, "report=%v: metric %s must still exist", report, k)
			assert.InDelta(t, v, got[k], 0.0001, "report=%v: metric %s", report, k)
		}
	}
}

func BenchmarkNewStats(b *testing.B) {
	logger := logptest.NewTestingLogger(b, "")
	b.ReportAllocs()
	for b.Loop() {
		_ = NewStats(monitoring.NewRegistry(), logger)
	}
}
