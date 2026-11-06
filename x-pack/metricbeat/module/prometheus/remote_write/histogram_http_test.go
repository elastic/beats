// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package remote_write

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	rw "github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type httpTestPushReporter struct {
	ctx      context.Context
	cancel   context.CancelFunc
	doneCh   chan struct{}
	stopOnce sync.Once
	events   chan mb.Event
	mu       sync.Mutex
	seen     []mb.Event
}

func newHTTPTestPushReporter(ctx context.Context) *httpTestPushReporter {
	parent := ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	return &httpTestPushReporter{
		ctx:    ctx,
		cancel: cancel,
		doneCh: make(chan struct{}),
		events: make(chan mb.Event, 64),
	}
}

func (r *httpTestPushReporter) Done() <-chan struct{} {
	return r.ctx.Done()
}

func (r *httpTestPushReporter) stop() {
	r.stopOnce.Do(func() {
		close(r.doneCh)
		r.cancel()
	})
}

func (r *httpTestPushReporter) Event(event mb.Event) bool {
	select {
	case <-r.ctx.Done():
		return false
	case <-r.doneCh:
		return false
	default:
	}
	r.mu.Lock()
	r.seen = append(r.seen, event)
	r.mu.Unlock()
	select {
	case r.events <- event:
	default:
	}
	return true
}

func (r *httpTestPushReporter) Error(err error) bool {
	return r.Event(mb.Event{Error: err})
}

type xpackFakeTicker struct {
	ch     chan time.Time
	stopCh chan struct{}
}

func newXpackFakeTicker() *xpackFakeTicker {
	return &xpackFakeTicker{
		ch:     make(chan time.Time, 1),
		stopCh: make(chan struct{}),
	}
}

func (t *xpackFakeTicker) C() <-chan time.Time {
	return t.ch
}

func (t *xpackFakeTicker) Stop() {
	close(t.stopCh)
}

func (t *xpackFakeTicker) Tick(now time.Time) {
	select {
	case t.ch <- now:
	default:
	}
}

func histogramBucketWriteRequest(tsMillis int64, buckets map[string]float64) *prompb.WriteRequest {
	series := make([]prompb.TimeSeries, 0, len(buckets))
	for le, count := range buckets {
		series = append(series, prompb.TimeSeries{
			Labels: []prompb.Label{
				{Name: "__name__", Value: "http_request_duration_seconds_bucket"},
				{Name: "runtime", Value: "linux"},
				{Name: "le", Value: le},
			},
			Samples: []prompb.Sample{{
				Value:     count,
				Timestamp: tsMillis,
			}},
		})
	}
	return &prompb.WriteRequest{Timeseries: series}
}

func encodeSnappyWriteRequest(t *testing.T, req *prompb.WriteRequest) []byte {
	t.Helper()
	data, err := proto.Marshal(req)
	require.NoError(t, err, "prompb marshal must succeed")
	return snappy.Encode(nil, data)
}

func postSnappyWriteRequest(t *testing.T, handler http.HandlerFunc, req *prompb.WriteRequest) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(encodeSnappyWriteRequest(t, req))))
	return rec
}

func countHistogramEvents(events []mb.Event) int {
	n := 0
	for _, ev := range events {
		if _, ok := ev.ModuleFields["http_request_duration_seconds"]; ok {
			n++
		}
	}
	return n
}

func histogramEventsFromReporter(reporter *httpTestPushReporter) []mb.Event {
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	out := make([]mb.Event, len(reporter.seen))
	copy(out, reporter.seen)
	return out
}

type httpMetricSetHandle struct {
	m        *rw.MetricSet
	typed    *remoteWriteTypedGenerator
	setNow   func(time.Time)
	ticker   *xpackFakeTicker
	reporter *httpTestPushReporter
	runDone  chan struct{}
}

func (h *httpMetricSetHandle) shutdown(t *testing.T) {
	t.Helper()
	h.reporter.stop()
	select {
	case <-h.runDone:
	case <-time.After(5 * time.Second):
		t.Fatal("MetricSet.Run must exit after reporter shutdown")
	}
}

func newXPackRemoteWriteHTTPMetricSet(t *testing.T, quietPeriod time.Duration) *httpMetricSetHandle {
	t.Helper()

	cfg := map[string]interface{}{
		"module":     "prometheus",
		"metricsets": []string{"remote_write"},
		"use_types":  true,
		"period":     "60s",
		"histogram_assembly": map[string]interface{}{
			"enabled":                true,
			"quiet_period":           quietPeriod.String(),
			"hard_timeout":           (10 * quietPeriod).String(),
			"max_pending_histograms": 100,
			"max_pending_buckets":    1000,
		},
	}
	ms := mbtest.NewMetricSet(t, cfg)
	m, ok := ms.(*rw.MetricSet)
	require.True(t, ok, "metricset must be OSS remote_write MetricSet")

	gen, err := remoteWriteEventsGeneratorFactory(m.BaseMetricSet)
	require.NoError(t, err, "typed generator factory must succeed")
	typed, ok := gen.(*remoteWriteTypedGenerator)
	require.True(t, ok, "use_types must install typed generator")

	start := time.Unix(1_700_000_000, 0)
	nowFn, setNow := testClock(start)
	typed.now = nowFn
	m.SetPromEventsGeneratorForTest(gen)
	useOwnerLoop, hasEvents, hasBatches := m.FlowModeForTest()
	require.True(t, useOwnerLoop, "assembler HTTP tests must exercise owner-loop mode")
	require.False(t, hasEvents, "owner-loop mode must not allocate the events channel")
	require.True(t, hasBatches, "assembler HTTP tests must submit through the batches channel")

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
	t.Cleanup(reporter.stop) // idempotent; tests must call shutdown to join Run

	return &httpMetricSetHandle{
		m: m, typed: typed, setNow: setNow, ticker: ticker, reporter: reporter, runDone: runDone,
	}
}

func drainReporterEvents(reporter *httpTestPushReporter) []mb.Event {
	var out []mb.Event
	for {
		select {
		case ev := <-reporter.events:
			out = append(out, ev)
		default:
			return out
		}
	}
}

func TestMetricSetRunExitsAfterReporterStopWithoutExtraTick(t *testing.T) {
	const tsMillis = int64(1_700_000_000_000)
	h := newXPackRemoteWriteHTTPMetricSet(t, time.Second)
	handler := h.m.HTTPHandler()

	h.setNow(time.UnixMilli(tsMillis))
	require.Equal(t, http.StatusAccepted, postSnappyWriteRequest(t, handler, histogramBucketWriteRequest(tsMillis, map[string]float64{
		"0.25": 10, "0.50": 20, "+Inf": 30,
	})).Code)

	flushAt := time.UnixMilli(tsMillis).Add(time.Second + time.Millisecond)
	h.setNow(flushAt)
	h.ticker.Tick(flushAt)
	require.Eventually(t, func() bool {
		return countHistogramEvents(histogramEventsFromReporter(h.reporter)) == 1
	}, time.Second, 5*time.Millisecond, "flush must complete before idle shutdown test")

	h.shutdown(t)
}

func TestHTTPTypedHistogramMergesAcrossSnappyWriteRequests(t *testing.T) {
	const tsMillis = int64(1_700_000_000_000)
	quietPeriod := time.Second

	h := newXPackRemoteWriteHTTPMetricSet(t, quietPeriod)
	defer h.shutdown(t)
	handler := h.m.HTTPHandler()
	typed := h.typed
	setNow := h.setNow
	ticker := h.ticker
	reporter := h.reporter

	setNow(time.UnixMilli(tsMillis))

	rec1 := postSnappyWriteRequest(t, handler, histogramBucketWriteRequest(tsMillis, map[string]float64{
		"0.25": 10,
		"0.50": 20,
	}))
	assert.Equal(t, http.StatusAccepted, rec1.Code, "first bucket batch must be accepted")
	firstEvents := drainReporterEvents(reporter)
	assert.Equal(t, 0, countHistogramEvents(firstEvents), "no histogram event must be published before quiet flush")

	rec2 := postSnappyWriteRequest(t, handler, histogramBucketWriteRequest(tsMillis, map[string]float64{
		"+Inf": 30,
	}))
	assert.Equal(t, http.StatusAccepted, rec2.Code, "second bucket batch must be accepted")
	midEvents := drainReporterEvents(reporter)
	assert.Equal(t, 0, countHistogramEvents(midEvents), "partial histogram must not publish before quiet flush")

	flushAt := time.UnixMilli(tsMillis).Add(quietPeriod + time.Millisecond)
	setNow(flushAt)
	ticker.Tick(flushAt)

	require.Eventually(t, func() bool {
		return countHistogramEvents(histogramEventsFromReporter(reporter)) == 1
	}, time.Second, 5*time.Millisecond, "quiet flush must publish exactly one merged histogram event")

	published := histogramEventsFromReporter(reporter)
	require.Equal(t, 1, countHistogramEvents(published), "exactly one histogram event after flush")
	var histEv mb.Event
	for _, ev := range published {
		if _, ok := ev.ModuleFields["http_request_duration_seconds"]; ok {
			histEv = ev
			break
		}
	}
	hist := histEv.ModuleFields["http_request_duration_seconds"].(mapstr.M)["histogram"].(mapstr.M)
	assert.Equal(t, []float64{0.125, 0.375, 0.5}, hist["values"], "merged centroids must include all bucket bounds")
	assert.Equal(t, []uint64{0, 0, 0}, hist["counts"], "first-scrape histogram counts use counter-cache deltas (zero until a repeat scrape)")

	stats := typed.HistogramAssemblyStats()
	assert.Equal(t, uint64(1), stats.FlushesComplete, "quiet flush must complete one histogram")
	assert.Equal(t, 0, stats.PendingHistograms, "assembler must have no pending histograms after flush")
}
