// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	monitoringlog "github.com/elastic/beats/v7/libbeat/monitoring/report/log"
	"github.com/elastic/beats/v7/metricbeat/mb"
	rw "github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
	xcollector "github.com/elastic/beats/v7/x-pack/metricbeat/module/prometheus/collector"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

var histogramAssemblerMonitoringMetricNames = []string{
	"pending_gauge",
	"pending_buckets_gauge",
	"quiet_flushes_total",
	"hard_timeout_flushes_total",
	"partial_flushes_total",
	"capacity_rejections_total",
	"late_buckets_total",
	"late_buckets_dropped_total",
	"shutdown_dropped_total",
}

func histogramAssemblerMonitoringPrefix() string {
	return "histogram_assembler."
}

func histogramAssemblerMetricSnapshot(t *testing.T, msReg *monitoring.Registry) map[string]int64 {
	t.Helper()
	require.NotNil(t, msReg, "metricset registry must be set for snapshot tests")

	snap := monitoring.CollectFlatSnapshot(msReg, monitoring.Full, false)
	out := make(map[string]int64, len(histogramAssemblerMonitoringMetricNames))
	prefix := histogramAssemblerMonitoringPrefix()
	for _, name := range histogramAssemblerMonitoringMetricNames {
		key := prefix + name
		val, ok := snap.Ints[key]
		require.True(t, ok, "expected monitoring metric %q to be registered", key)
		out[name] = val
	}

	for key := range snap.Ints {
		if len(key) <= len(prefix) || key[:len(prefix)] != prefix {
			continue
		}
		suffix := key[len(prefix):]
		assert.Contains(t, histogramAssemblerMonitoringMetricNames, suffix,
			"unexpected dynamic or extra histogram_assembler monitoring key %q", key)
	}
	return out
}

func newTestTypedGeneratorWithMonitoring(t *testing.T, cfg histogramAssemblyConfig, start time.Time, msReg *monitoring.Registry) (*remoteWriteTypedGenerator, func(time.Time)) {
	t.Helper()
	nowFn, setNow := testClock(start)
	counters := xcollector.NewCounterCache(time.Minute)
	mon := registerHistogramAssemblerMonitoring(msReg)
	g := &remoteWriteTypedGenerator{
		counterCache:    counters,
		rateCounters:    true,
		assemblyConfig:  cfg,
		now:             nowFn,
		retainedFlushes: make(map[string]mb.Event),
		histogramMon:    mon,
	}
	g.assembler = newHistogramAssembler(cfg, mon)
	counters.Start()
	t.Cleanup(counters.Stop)
	t.Cleanup(func() {
		if g.histogramMon != nil {
			g.histogramMon.unregister()
			g.histogramMon = nil
		}
	})
	return g, setNow
}

func TestRegisterHistogramAssemblerMonitoringIdempotent(t *testing.T) {
	msReg := monitoring.NewRegistry()
	first := registerHistogramAssemblerMonitoring(msReg)
	second := registerHistogramAssemblerMonitoring(msReg)
	require.Same(t, first, second, "re-registering on the same metricset registry must not duplicate metrics")
	histogramAssemblerMetricSnapshot(t, msReg)
	t.Cleanup(first.unregister)
}

func TestHistogramAssemblerPendingMetricsAreReportedAsGauges(t *testing.T) {
	msReg := monitoring.NewRegistry()
	mon := registerHistogramAssemblerMonitoring(msReg)
	t.Cleanup(mon.unregister)
	snap := monitoring.CollectFlatSnapshot(msReg, monitoring.Full, false)
	for _, name := range []string{"pending_gauge", "pending_buckets_gauge"} {
		metricKey := "histogram_assembler." + name
		assert.Contains(t, snap.Ints, metricKey, "%q must be registered", metricKey)
		reporterKey := "dataset.instance." + metricKey
		assert.True(t, monitoringlog.IsGauge(reporterKey), "%q must be reported as a gauge, not a delta", reporterKey)
	}
}

func TestHistogramAssemblerMonitoringIngestUpdatesPendingGauges(t *testing.T) {
	msReg := monitoring.NewRegistry()
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(100, 0)
	g, setNow := newTestTypedGeneratorWithMonitoring(t, cfg, start, msReg)
	ts := model.TimeFromUnix(100)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.50"}, 2, ts),
	})

	metrics := histogramAssemblerMetricSnapshot(t, msReg)
	assert.Equal(t, int64(1), metrics["pending_gauge"], "ingest must track one pending histogram")
	assert.Equal(t, int64(2), metrics["pending_buckets_gauge"], "ingest must track pending bucket count in assembler memory only")
}

func TestHistogramAssemblerMonitoringQuietFlush(t *testing.T) {
	msReg := monitoring.NewRegistry()
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(200, 0)
	g, setNow := newTestTypedGeneratorWithMonitoring(t, cfg, start, msReg)
	ts := model.TimeFromUnix(200)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 1, ts),
	})
	setNow(ts.Time().Add(cfg.QuietPeriod + time.Millisecond))
	g.FlushExpired(g.now())

	metrics := histogramAssemblerMetricSnapshot(t, msReg)
	assert.Equal(t, int64(0), metrics["pending_gauge"], "quiet flush clears pending histogram gauge")
	assert.Equal(t, int64(0), metrics["pending_buckets_gauge"], "quiet flush clears pending bucket gauge")
	assert.Equal(t, int64(1), metrics["quiet_flushes_total"], "quiet flush must increment quiet counter")
	assert.Equal(t, int64(0), metrics["hard_timeout_flushes_total"], "quiet flush must not increment hard timeout counter")
	assert.Equal(t, int64(0), metrics["partial_flushes_total"], "quiet flush must not increment partial counter")
}

func TestHistogramAssemblerMonitoringHardTimeoutPartialFlush(t *testing.T) {
	msReg := monitoring.NewRegistry()
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(300, 0)
	g, setNow := newTestTypedGeneratorWithMonitoring(t, cfg, start, msReg)
	ts := model.TimeFromUnix(300)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
	})
	g.FlushExpired(ts.Time().Add(cfg.HardTimeout))

	metrics := histogramAssemblerMetricSnapshot(t, msReg)
	assert.Equal(t, int64(1), metrics["hard_timeout_flushes_total"], "hard timeout flush must increment hard timeout counter")
	assert.Equal(t, int64(1), metrics["partial_flushes_total"], "hard timeout flush must increment partial counter")
	assert.Equal(t, int64(0), metrics["quiet_flushes_total"], "partial hard flush must not increment quiet counter")
}

func TestHistogramAssemblerMonitoringCapacityRejection(t *testing.T) {
	msReg := monitoring.NewRegistry()
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Second,
		HardTimeout:          time.Second,
		MaxPendingHistograms: 1,
		MaxPendingBuckets:    100,
	}
	start := time.Unix(400, 0)
	g, setNow := newTestTypedGeneratorWithMonitoring(t, cfg, start, msReg)
	ts := model.TimeFromUnix(400)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("metric_a_bucket", map[string]string{"le": "0.1"}, 1, ts),
	})
	require.ErrorIs(t, g.CheckCapacity(model.Samples{
		promBucketSample("metric_b_bucket", map[string]string{"le": "0.1"}, 1, ts),
	}), rw.ErrRemoteWriteCapacityExceeded, "second histogram must be rejected")

	metrics := histogramAssemblerMetricSnapshot(t, msReg)
	assert.Equal(t, int64(1), metrics["capacity_rejections_total"], "capacity rejection must increment once per rejected request")
	assert.Equal(t, int64(1), metrics["pending_gauge"], "rejected batch must not mutate pending histogram gauge")
}

func TestHistogramAssemblerMonitoringLateBucketAfterFlush(t *testing.T) {
	msReg := monitoring.NewRegistry()
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(500, 0)
	g, setNow := newTestTypedGeneratorWithMonitoring(t, cfg, start, msReg)
	ts := model.TimeFromUnix(500)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 1, ts),
	})
	flushAt := ts.Time().Add(cfg.QuietPeriod + time.Millisecond)
	setNow(flushAt)
	g.FlushExpired(flushAt)

	setNow(flushAt.Add(time.Millisecond))
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.5"}, 2, ts),
	})

	metrics := histogramAssemblerMetricSnapshot(t, msReg)
	assert.Equal(t, int64(1), metrics["late_buckets_total"], "post-close bucket observation must increment late_buckets_total")
	assert.Equal(t, int64(1), metrics["late_buckets_dropped_total"], "dropped late bucket must increment late_buckets_dropped_total")
}

func TestHistogramAssemblerMonitoringShutdownDropped(t *testing.T) {
	msReg := monitoring.NewRegistry()
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(600, 0)
	g, setNow := newTestTypedGeneratorWithMonitoring(t, cfg, start, msReg)
	ts := model.TimeFromUnix(600)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
	})
	g.retainedFlushes = map[string]mb.Event{"k1": {}, "k2": {}}

	g.Stop()

	metrics := histogramAssemblerMetricSnapshot(t, msReg)
	assert.Equal(t, int64(0), metrics["pending_gauge"], "shutdown clears pending gauge")
	assert.Equal(t, int64(3), metrics["shutdown_dropped_total"], "shutdown must count pending histograms plus retained flush entries")
}

func TestHistogramAssemblerMonitoringUnregisterOnStop(t *testing.T) {
	msReg := monitoring.NewRegistry()
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(700, 0)
	g, _ := newTestTypedGeneratorWithMonitoring(t, cfg, start, msReg)

	_, ok := histogramAssemblerMonitoringSlots.Load(msReg)
	require.True(t, ok, "register must store a slot keyed by metricset registry")

	g.Stop()

	_, ok = histogramAssemblerMonitoringSlots.Load(msReg)
	assert.False(t, ok, "Stop must free the process-wide monitoring slot for restart cycles")

	// A subsequent register on the same registry (receiver restart) must succeed.
	again := registerHistogramAssemblerMonitoring(msReg)
	require.NotNil(t, again)
	t.Cleanup(again.unregister)
	histogramAssemblerMetricSnapshot(t, msReg)
}

func TestRemoteWriteFactoryUseTypesFalseDoesNotRegisterHistogramAssemblerMonitoring(t *testing.T) {
	base := newFactoryBaseMetricSet(t, map[string]any{
		"use_types": false,
	})
	gen, err := remoteWriteEventsGeneratorFactory(base)
	require.NoError(t, err)
	require.IsType(t, &rw.RemoteWriteEventGenerator{}, gen)

	snap := monitoring.CollectFlatSnapshot(base.Metrics(), monitoring.Full, false)
	for key := range snap.Ints {
		assert.NotContains(t, key, "histogram_assembler.", "use_types=false must not register histogram assembler metrics")
	}
}

func TestRemoteWriteFactoryAssemblerDisabledDoesNotRegisterHistogramAssemblerMonitoring(t *testing.T) {
	base := newFactoryBaseMetricSet(t, nil)
	gen, err := remoteWriteEventsGeneratorFactory(base)
	require.NoError(t, err)
	require.IsType(t, &remoteWriteTypedGenerator{}, gen)

	snap := monitoring.CollectFlatSnapshot(base.Metrics(), monitoring.Full, false)
	for key := range snap.Ints {
		assert.NotContains(t, key, "histogram_assembler.", "disabled assembler must not register histogram assembler metrics")
	}
}
