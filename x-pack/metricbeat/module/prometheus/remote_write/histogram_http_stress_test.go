// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package remote_write

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	rw "github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
)

func histogramIdentityWriteRequestWithLabels(tsMillis int64, zone, le string, count float64) *prompb.WriteRequest {
	return &prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{{
			Labels: []prompb.Label{
				{Name: "__name__", Value: "http_request_duration_seconds_bucket"},
				{Name: "runtime", Value: "linux"},
				{Name: "zone", Value: zone},
				{Name: "le", Value: le},
			},
			Samples: []prompb.Sample{{
				Value:     count,
				Timestamp: tsMillis,
			}},
		}},
	}
}

func TestHTTPConcurrentRemoteWriteCapacityStress(t *testing.T) {
	const (
		tsMillis      = int64(1_700_000_100_000)
		workers       = 12
		maxHistograms = 4
		maxBuckets    = 12
	)

	cfg := map[string]interface{}{
		"module":     "prometheus",
		"metricsets": []string{"remote_write"},
		"use_types":  true,
		"period":     "60s",
		"histogram_assembly": map[string]interface{}{
			"enabled":                true,
			"quiet_period":           "500ms",
			"hard_timeout":           "5s",
			"max_pending_histograms": maxHistograms,
			"max_pending_buckets":    maxBuckets,
		},
	}
	ms := mbtest.NewMetricSet(t, cfg)
	m, ok := ms.(*rw.MetricSet)
	require.True(t, ok)

	gen, err := remoteWriteEventsGeneratorFactory(m.BaseMetricSet)
	require.NoError(t, err)
	typed, ok := gen.(*remoteWriteTypedGenerator)
	require.True(t, ok)
	start := time.UnixMilli(tsMillis)
	nowFn, setNow := testClock(start)
	typed.now = nowFn
	m.SetPromEventsGeneratorForTest(gen)
	useOwnerLoop, hasEvents, hasBatches := m.FlowModeForTest()
	require.True(t, useOwnerLoop, "assembler stress test must exercise owner-loop mode")
	require.False(t, hasEvents, "owner-loop mode must not allocate the events channel")
	require.True(t, hasBatches, "assembler stress test must submit through the batches channel")

	ticker := newXpackFakeTicker()
	rw.ConfigureOwnerLoopForTest(m, func(time.Duration) rw.OwnerLoopTickSource {
		return ticker
	}, true)
	t.Cleanup(func() { rw.ClearOwnerLoopTestConfiguration(m) })

	reporter := newHTTPTestPushReporter(nil)
	runDone := make(chan struct{})
	go func() {
		m.Run(reporter)
		close(runDone)
	}()
	select {
	case <-rw.WaitOwnerLoopReady(m):
	case <-time.After(2 * time.Second):
		t.Fatal("owner loop should become ready")
	}
	t.Cleanup(reporter.stop)

	handle := &httpMetricSetHandle{
		m: m, typed: typed, setNow: setNow, ticker: ticker, reporter: reporter, runDone: runDone,
	}
	handler := handle.m.HTTPHandler()
	setNow(start)

	var (
		accepted atomic.Int32
		rejected atomic.Int32
		startBar sync.WaitGroup
		doneBar  sync.WaitGroup
	)
	startBar.Add(1)
	doneBar.Add(workers)

	for i := range workers {
		go func(idx int) {
			defer doneBar.Done()
			startBar.Wait()
			zone := fmt.Sprintf("z%d", idx%6)
			req := histogramIdentityWriteRequestWithLabels(tsMillis, zone, "0.25", float64(idx+1))
			rec := postSnappyWriteRequest(t, handler, req)
			switch rec.Code {
			case http.StatusAccepted:
				accepted.Add(1)
			case http.StatusServiceUnavailable:
				rejected.Add(1)
			default:
				t.Errorf("worker %d unexpected status %d", idx, rec.Code)
			}
		}(i)
	}

	startBar.Done()
	done := make(chan struct{})
	go func() {
		doneBar.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent remote_write handlers deadlocked")
	}

	assert.Equal(t, int32(workers), accepted.Load()+rejected.Load(), "every worker must get 202 or 503")
	assert.Positive(t, rejected.Load(), "some requests must be rejected when over max_pending_histograms")
	assert.Positive(t, accepted.Load(), "some requests must be accepted under capacity")

	stats := typed.HistogramAssemblyStats()
	assert.LessOrEqual(t, stats.PendingHistograms, maxHistograms, "pending histograms must respect limit")
	assert.LessOrEqual(t, stats.PendingBuckets, maxBuckets, "pending buckets must respect limit")

	flushAt := start.Add(600 * time.Millisecond)
	setNow(flushAt)
	ticker.Tick(flushAt)
	_ = drainReporterEvents(reporter)
	handle.shutdown(t)
}
