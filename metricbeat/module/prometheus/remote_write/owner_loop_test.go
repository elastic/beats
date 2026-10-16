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

package remote_write

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type testPushReporter struct {
	ctx      context.Context
	cancel   context.CancelFunc
	doneCh   chan struct{}
	stopOnce sync.Once
	events   chan mb.Event
	eventFn  func(mb.Event) bool
}

func newTestPushReporter(parent context.Context) *testPushReporter {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	return &testPushReporter{
		ctx:    ctx,
		cancel: cancel,
		doneCh: make(chan struct{}),
		events: make(chan mb.Event, 256),
	}
}

func (r *testPushReporter) Done() <-chan struct{} {
	return r.ctx.Done()
}

func (r *testPushReporter) stop() {
	r.stopOnce.Do(func() {
		close(r.doneCh)
		r.cancel()
	})
}

func (r *testPushReporter) Event(event mb.Event) bool {
	if r.eventFn != nil {
		return r.eventFn(event)
	}
	select {
	case <-r.ctx.Done():
		return false
	case <-r.doneCh:
		return false
	case r.events <- event:
		return true
	}
}

func (r *testPushReporter) Error(err error) bool {
	return r.Event(mb.Event{Error: err})
}

func waitForOwnerLoop(t *testing.T, m *MetricSet) {
	t.Helper()
	select {
	case <-waitRunReady(m):
	case <-time.After(2 * time.Second):
		t.Fatal("owner loop should signal readiness")
	}
}

func startOwnerLoop(t *testing.T, m *MetricSet, reporter *testPushReporter) {
	t.Helper()
	runDone := make(chan struct{})
	go func() {
		m.Run(reporter)
		close(runDone)
	}()
	waitForOwnerLoop(t, m)
	t.Cleanup(func() {
		reporter.stop()
		select {
		case <-runDone:
		case <-time.After(5 * time.Second):
			t.Error("MetricSet.Run must exit after reporter shutdown")
		}
	})
}

type fakeTicker struct {
	ch     chan time.Time
	stopCh chan struct{}
}

func newFakeTicker() *fakeTicker {
	return &fakeTicker{
		ch:     make(chan time.Time, 1),
		stopCh: make(chan struct{}),
	}
}

func (t *fakeTicker) C() <-chan time.Time {
	return t.ch
}

func (t *fakeTicker) Stop() {
	close(t.stopCh)
}

func (t *fakeTicker) Tick(now time.Time) {
	select {
	case t.ch <- now:
	default:
	}
}

type fakeGenerator struct {
	mu                  sync.Mutex
	inGenerate          int
	maxConcurrent       int
	generateCalls       atomic.Int32
	checkCapacityCalls  atomic.Int32
	startCalled         bool
	stopCalled          bool
	startCalls          int
	stopCalls           int
	generateHook        func(model.Samples) map[string]mb.Event
	checkCapacityFn     func(model.Samples) error
	flushHook           func(time.Time) map[string]mb.Event
	flushInterval       time.Duration
	retainedFlushEvents map[string]mb.Event
}

func (f *fakeGenerator) Start() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.startCalled = true
	f.startCalls++
}

func (f *fakeGenerator) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalled = true
	f.stopCalls++
}

func (f *fakeGenerator) GenerateEvents(metrics model.Samples) map[string]mb.Event {
	f.mu.Lock()
	f.inGenerate++
	if f.inGenerate > f.maxConcurrent {
		f.maxConcurrent = f.inGenerate
	}
	f.generateCalls.Add(1)
	f.mu.Unlock()

	defer func() {
		f.mu.Lock()
		f.inGenerate--
		f.mu.Unlock()
	}()

	if f.generateHook != nil {
		return f.generateHook(metrics)
	}
	return map[string]mb.Event{
		"key": {RootFields: mapstr.M{"samples": len(metrics)}},
	}
}

func (f *fakeGenerator) CheckCapacity(samples model.Samples) error {
	f.checkCapacityCalls.Add(1)
	if f.checkCapacityFn != nil {
		return f.checkCapacityFn(samples)
	}
	return nil
}

func (f *fakeGenerator) FlushExpired(now time.Time) map[string]mb.Event {
	if f.flushHook != nil {
		return f.flushHook(now)
	}
	return nil
}

func (f *fakeGenerator) NextFlushInterval() time.Duration {
	if f.flushInterval == 0 {
		return time.Second
	}
	return f.flushInterval
}

func (f *fakeGenerator) RetainUnpublishedFlushEvents(events map[string]mb.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.retainedFlushEvents = events
}

func (f *fakeGenerator) RequiresOwnerLoop() bool {
	return true
}

type fakeBatchProcessorGenerator struct {
	*fakeGenerator
	processCalls atomic.Int32
	processFn    func(model.Samples) (map[string]mb.Event, error)
}

func (f *fakeBatchProcessorGenerator) ProcessOwnerLoopBatch(samples model.Samples) (map[string]mb.Event, error) {
	f.processCalls.Add(1)
	if f.processFn != nil {
		return f.processFn(samples)
	}
	return nil, nil
}

func newMetricSetWithFakeGenerator(t *testing.T, gen *fakeGenerator) (*MetricSet, *fakeTicker, *testPushReporter) {
	t.Helper()
	return newMetricSetWithGenerator(t, gen)
}

func newMetricSetWithGenerator(t *testing.T, gen RemoteWriteEventsGenerator) (*MetricSet, *fakeTicker, *testPushReporter) {
	t.Helper()
	m := newTestMetricSetBase(t, 1024*1024, 10*1024*1024)
	m.setPromEventsGenerator(gen)
	ticker := newFakeTicker()
	setOwnerLoopTestSeams(m, ownerLoopSeams{
		newTickSource:  func(time.Duration) tickSource { return ticker },
		skipHTTPServer: true,
	})
	t.Cleanup(func() { clearOwnerLoopTestSeams(m) })
	reporter := newTestPushReporter(nil)
	startOwnerLoop(t, m, reporter)
	return m, ticker, reporter
}

func TestSetPromEventsGeneratorForTestRecomputesModeAndChannels(t *testing.T) {
	m := newTestMetricSetBase(t, 1024*1024, 10*1024*1024)

	m.SetPromEventsGeneratorForTest(&fakeGenerator{})
	assert.True(t, m.useOwnerLoop, "owner-capable replacement must select owner mode")
	assert.Nil(t, m.events, "owner mode must clear the events channel")
	assert.NotNil(t, m.batches, "owner mode must allocate the batches channel")

	m.SetPromEventsGeneratorForTest(newLegacyFlowGenerator())
	assert.False(t, m.useOwnerLoop, "legacy replacement must recompute legacy mode")
	assert.NotNil(t, m.events, "legacy mode must allocate the events channel")
	assert.Nil(t, m.batches, "legacy mode must clear the batches channel")

	require.NoError(t, m.Close(), "legacy replacement must close")
	assert.False(t, m.startLegacyGenerator(), "closed legacy mode must be terminal")

	replacement := newLegacyFlowGenerator()
	m.SetPromEventsGeneratorForTest(replacement)
	assert.True(t, m.startLegacyGenerator(), "replacement before Run must reset closed state")
	require.NoError(t, m.Close(), "replacement must remain closable")
}

func postWriteRequest(t *testing.T, m *MetricSet, numSamples int) *httptest.ResponseRecorder {
	t.Helper()
	writeReq := createTestWriteRequest(numSamples)
	body, err := encodeWriteRequest(writeReq)
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	m.handleFunc(rec, req)
	return rec
}

func TestOwnerLoopSerializesConcurrentGenerateEvents(t *testing.T) {
	gen := &fakeGenerator{
		generateHook: func(samples model.Samples) map[string]mb.Event {
			time.Sleep(50 * time.Millisecond)
			return map[string]mb.Event{"k": {}}
		},
	}
	m, _, _ := newMetricSetWithFakeGenerator(t, gen)

	const workers = 8
	var wg sync.WaitGroup
	statuses := make([]int, workers)
	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rec := postWriteRequest(t, m, 1)
			statuses[idx] = rec.Code
		}(i)
	}
	wg.Wait()

	for i, code := range statuses {
		assert.Equal(t, http.StatusAccepted, code, "worker %d should get 202", i)
	}
	assert.Equal(t, 1, gen.maxConcurrent, "GenerateEvents must not run concurrently")
	assert.Equal(t, workers, int(gen.generateCalls.Load()), "every batch should be processed")
}

func TestOwnerLoopAcceptedAfterReporterEvent(t *testing.T) {
	reported := make(chan struct{}, 1)
	gen := &fakeGenerator{}
	m, _, reporter := newMetricSetWithFakeGenerator(t, gen)
	reporter.eventFn = func(event mb.Event) bool {
		select {
		case reported <- struct{}{}:
		default:
		}
		select {
		case reporter.events <- event:
			return true
		default:
			return false
		}
	}

	rec := postWriteRequest(t, m, 2)
	assert.Equal(t, http.StatusAccepted, rec.Code, "successful batch should return 202")
	select {
	case <-reported:
	default:
		t.Fatal("reporter should have received events for accepted batch")
	}
}

func TestOwnerLoopPrefersBatchProcessor(t *testing.T) {
	gen := &fakeBatchProcessorGenerator{
		fakeGenerator: &fakeGenerator{},
		processFn: func(samples model.Samples) (map[string]mb.Event, error) {
			require.Len(t, samples, 2)
			return map[string]mb.Event{
				"processed": {RootFields: mapstr.M{"source": "processor"}},
			}, nil
		},
	}
	m, _, reporter := newMetricSetWithGenerator(t, gen)

	rec := postWriteRequest(t, m, 2)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, int32(1), gen.processCalls.Load(), "batch processor must run exactly once")
	assert.Zero(t, gen.checkCapacityCalls.Load(), "processor must bypass fallback preflight")
	assert.Zero(t, gen.generateCalls.Load(), "processor must bypass fallback generation")
	select {
	case event := <-reporter.events:
		assert.Equal(t, "processor", event.RootFields["source"])
	default:
		t.Fatal("processor events must be published")
	}
}

func TestOwnerLoopFallbackChecksCapacityThenGeneratesEvents(t *testing.T) {
	gen := &fakeGenerator{}
	m, _, _ := newMetricSetWithFakeGenerator(t, gen)

	rec := postWriteRequest(t, m, 1)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, int32(1), gen.checkCapacityCalls.Load(), "fallback must check capacity once")
	assert.Equal(t, int32(1), gen.generateCalls.Load(), "fallback must generate events once")
}

func TestOwnerLoopBatchProcessorCapacityErrorReturns503(t *testing.T) {
	gen := &fakeBatchProcessorGenerator{
		fakeGenerator: &fakeGenerator{},
		processFn: func(model.Samples) (map[string]mb.Event, error) {
			return nil, ErrRemoteWriteCapacityExceeded
		},
	}
	m, _, _ := newMetricSetWithGenerator(t, gen)

	rec := postWriteRequest(t, m, 1)

	assertHTTPErrorBody(t, rec, http.StatusServiceUnavailable, batchOutcomeMsgCapacityExceeded)
	assert.Equal(t, int32(1), gen.processCalls.Load(), "batch processor must run exactly once")
	assert.Zero(t, gen.generateCalls.Load(), "GenerateEvents must not run after processor capacity errors")
}

func TestOwnerLoopBatchProcessorUnexpectedErrorReturns500(t *testing.T) {
	gen := &fakeBatchProcessorGenerator{
		fakeGenerator: &fakeGenerator{},
		processFn: func(model.Samples) (map[string]mb.Event, error) {
			return nil, errors.New("processor internal failure")
		},
	}
	m, _, _ := newMetricSetWithGenerator(t, gen)

	rec := postWriteRequest(t, m, 1)

	assertHTTPErrorBody(t, rec, http.StatusInternalServerError, batchOutcomeMsgPreflightFailed)
	assert.Equal(t, int32(1), gen.processCalls.Load(), "batch processor must run exactly once")
	assert.Zero(t, gen.generateCalls.Load(), "GenerateEvents must not run after processor errors")
}

func TestOwnerLoopBatchProcessorPipelineRejectionReturns503(t *testing.T) {
	gen := &fakeBatchProcessorGenerator{
		fakeGenerator: &fakeGenerator{},
		processFn: func(model.Samples) (map[string]mb.Event, error) {
			return map[string]mb.Event{
				"processed": {RootFields: mapstr.M{"source": "processor"}},
			}, nil
		},
	}
	m, _, reporter := newMetricSetWithGenerator(t, gen)
	var eventCount atomic.Int32
	reporter.eventFn = func(mb.Event) bool {
		eventCount.Add(1)
		return false
	}

	rec := postWriteRequest(t, m, 1)

	assertHTTPErrorBody(t, rec, http.StatusServiceUnavailable, batchOutcomeMsgPipelineRejected)
	assert.Equal(t, int32(1), gen.processCalls.Load(), "batch processor must run exactly once")
	assert.Equal(t, int32(1), eventCount.Load(), "processor event must use existing reporter path")
	assert.Zero(t, gen.generateCalls.Load(), "GenerateEvents must not run for processor batches")
}

func TestOwnerLoopCapacityRejectionReturns503(t *testing.T) {
	gen := &fakeGenerator{
		checkCapacityFn: func(model.Samples) error {
			return ErrRemoteWriteCapacityExceeded
		},
	}
	m, _, _ := newMetricSetWithFakeGenerator(t, gen)

	rec := postWriteRequest(t, m, 3)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "capacity rejection must be retryable 503")
	assert.Contains(t, rec.Body.String(), batchOutcomeMsgCapacityExceeded)
	assert.Equal(t, 0, int(gen.generateCalls.Load()), "GenerateEvents must not run when capacity rejects")
}

func TestOwnerLoopRequestCancellationUnblocks(t *testing.T) {
	block := make(chan struct{})
	gen := &fakeGenerator{
		generateHook: func(model.Samples) map[string]mb.Event {
			<-block
			return map[string]mb.Event{"k": {}}
		},
	}
	m, _, _ := newMetricSetWithFakeGenerator(t, gen)

	ctx, cancel := context.WithCancel(context.Background())
	writeReq := createTestWriteRequest(1)
	body, err := encodeWriteRequest(writeReq)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		m.handleFunc(rec, req)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return gen.generateCalls.Load() == 1
	}, time.Second, 5*time.Millisecond, "owner loop should start processing the batch")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancelled request should unblock handler")
	}
	close(block)
}

func TestOwnerLoopTimerFlushUsesFakeTicker(t *testing.T) {
	var flushCount atomic.Int32
	gen := &fakeGenerator{
		flushHook: func(time.Time) map[string]mb.Event {
			flushCount.Add(1)
			return map[string]mb.Event{"flush": {RootFields: mapstr.M{"flushed": true}}}
		},
		flushInterval: time.Minute,
	}
	_, ticker, reporter := newMetricSetWithFakeGenerator(t, gen)
	var reported atomic.Int32
	reporter.eventFn = func(event mb.Event) bool {
		reported.Add(1)
		select {
		case reporter.events <- event:
			return true
		default:
			return false
		}
	}

	ticker.Tick(time.Unix(100, 0))

	require.Eventually(t, func() bool {
		return flushCount.Load() == 1
	}, time.Second, 5*time.Millisecond, "flush tick should invoke FlushExpired")
	require.Eventually(t, func() bool {
		return reported.Load() >= 1
	}, time.Second, 5*time.Millisecond, "flush events should reach reporter")
}

func TestOwnerLoopShutdownUnblocksBlockedHandler(t *testing.T) {
	block := make(chan struct{})
	gen := &fakeGenerator{
		generateHook: func(model.Samples) map[string]mb.Event {
			<-block
			return map[string]mb.Event{"k": {}}
		},
	}
	m, _, _ := newMetricSetWithFakeGenerator(t, gen)

	firstDone := make(chan struct{})
	go func() {
		postWriteRequest(t, m, 1)
		close(firstDone)
	}()

	require.Eventually(t, func() bool {
		return gen.generateCalls.Load() == 1
	}, time.Second, 5*time.Millisecond, "first batch should be processing")

	secondDone := make(chan struct{})
	var secondRec *httptest.ResponseRecorder
	go func() {
		secondRec = postWriteRequest(t, m, 1)
		close(secondDone)
	}()

	require.Eventually(t, func() bool {
		return m.handlersInFlight.Load() >= 2
	}, time.Second, 5*time.Millisecond, "second handler should block waiting for owner")

	m.shutdownIntake()
	require.Eventually(t, func() bool {
		select {
		case <-secondDone:
			return true
		default:
			return false
		}
	}, 2*time.Second, 5*time.Millisecond, "shutdown must unblock blocked handler")

	assertHTTPErrorBody(t, secondRec, http.StatusServiceUnavailable, "remote write intake stopped")
	close(block)
	<-firstDone
}

func assertHTTPErrorBody(t *testing.T, rec *httptest.ResponseRecorder, code int, fragment string) {
	t.Helper()
	require.Equal(t, code, rec.Code)
	require.Contains(t, rec.Body.String(), fragment)
}

func TestOwnerLoopGeneratorLifecycle(t *testing.T) {
	gen := &fakeGenerator{}
	m := newTestMetricSetBase(t, 1024*1024, 10*1024*1024)
	m.setPromEventsGenerator(gen)
	reporter := newTestPushReporter(nil)

	runDone := make(chan struct{})
	go func() {
		m.Run(reporter)
		close(runDone)
	}()
	waitForOwnerLoop(t, m)
	t.Cleanup(func() {
		reporter.stop()
		select {
		case <-runDone:
		case <-time.After(5 * time.Second):
			t.Error("MetricSet.Run must exit after test cleanup")
		}
	})

	require.Eventually(t, func() bool {
		gen.mu.Lock()
		defer gen.mu.Unlock()
		return gen.startCalled
	}, time.Second, 5*time.Millisecond, "Run should start generator once")

	require.NoError(t, m.Close(), "owner Close must be a no-op while Run owns lifecycle")
	gen.mu.Lock()
	assert.Zero(t, gen.stopCalls, "owner Close must not stop the active generator")
	gen.mu.Unlock()

	rec := postWriteRequest(t, m, 1)
	assert.Equal(t, http.StatusAccepted, rec.Code)

	reporter.stop()
	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Run should exit after reporter shutdown")
	}

	gen.mu.Lock()
	defer gen.mu.Unlock()
	assert.True(t, gen.startCalled, "generator Start must run exactly once per Run")
	assert.True(t, gen.stopCalled, "Run should stop generator after intake stops")
	assert.Equal(t, 1, gen.startCalls, "owner Run must start generator exactly once")
	assert.Equal(t, 1, gen.stopCalls, "owner Run must stop generator exactly once")
}

func TestMetricSetCloseWithoutRunIsNoOp(t *testing.T) {
	gen := &fakeGenerator{}
	m := newTestMetricSetBase(t, 1024*1024, 10*1024*1024)
	m.setPromEventsGenerator(gen)
	require.NoError(t, m.Close())
	gen.mu.Lock()
	defer gen.mu.Unlock()
	assert.False(t, gen.startCalled, "Close must not start generator")
	assert.False(t, gen.stopCalled, "Close must not stop generator without Run")
	assert.Zero(t, gen.startCalls, "Close must not call Start")
	assert.Zero(t, gen.stopCalls, "Close must not call Stop for owner mode")
}

func TestOwnerLoopPreflightUnexpectedErrorReturns500(t *testing.T) {
	gen := &fakeGenerator{
		checkCapacityFn: func(model.Samples) error {
			return errors.New("preflight internal failure")
		},
	}
	m, _, _ := newMetricSetWithFakeGenerator(t, gen)

	rec := postWriteRequest(t, m, 1)
	assertHTTPErrorBody(t, rec, http.StatusInternalServerError, batchOutcomeMsgPreflightFailed)
	assert.Equal(t, int32(0), gen.generateCalls.Load(), "GenerateEvents must not run on any preflight error")
}

func TestOwnerLoopBatchReporterEventFalseReturns503(t *testing.T) {
	gen := &fakeGenerator{
		generateHook: func(model.Samples) map[string]mb.Event {
			return map[string]mb.Event{
				"a": {RootFields: mapstr.M{"n": 1}},
				"b": {RootFields: mapstr.M{"n": 2}},
			}
		},
	}
	m, _, reporter := newMetricSetWithFakeGenerator(t, gen)
	var eventCount atomic.Int32
	reporter.eventFn = func(event mb.Event) bool {
		if eventCount.Add(1) == 1 {
			return true
		}
		return false
	}

	rec := postWriteRequest(t, m, 1)
	assertHTTPErrorBody(t, rec, http.StatusServiceUnavailable, batchOutcomeMsgPipelineRejected)
	assert.Equal(t, int32(2), eventCount.Load(), "owner loop should attempt to publish until failure")
}

func TestOwnerLoopTimerFlushRetainsOnReporterFailure(t *testing.T) {
	flushEvents := map[string]mb.Event{
		"flush-a": {RootFields: mapstr.M{"id": "a"}},
		"flush-b": {RootFields: mapstr.M{"id": "b"}},
	}
	gen := &fakeGenerator{
		flushHook: func(time.Time) map[string]mb.Event {
			out := make(map[string]mb.Event, len(flushEvents))
			maps.Copy(out, flushEvents)
			return out
		},
		flushInterval: time.Minute,
	}
	_, ticker, reporter := newMetricSetWithFakeGenerator(t, gen)
	var published atomic.Int32
	reporter.eventFn = func(mb.Event) bool {
		published.Add(1)
		return false
	}

	ticker.Tick(time.Unix(200, 0))

	require.Eventually(t, func() bool {
		return published.Load() == 1
	}, time.Second, 5*time.Millisecond, "flush should stop publishing after first pipeline rejection")

	require.Eventually(t, func() bool {
		gen.mu.Lock()
		defer gen.mu.Unlock()
		return len(gen.retainedFlushEvents) == 2
	}, time.Second, 5*time.Millisecond, "entire flush batch must be retained when publish fails")

	gen.mu.Lock()
	retained := gen.retainedFlushEvents
	gen.mu.Unlock()
	assert.Contains(t, retained, "flush-a", "retained set must include events not delivered to the pipeline")
	assert.Contains(t, retained, "flush-b", "retained set must include remaining events after publish failure")
}

func TestOwnerLoopTimerFlushRetainsPartialOnMidFlushFailure(t *testing.T) {
	const (
		keyAlpha = "alpha"
		keyBeta  = "beta"
		keyGamma = "gamma"
	)
	allKeys := []string{keyAlpha, keyBeta, keyGamma}
	flushEvents := map[string]mb.Event{
		keyAlpha: {RootFields: mapstr.M{"id": keyAlpha}},
		keyBeta:  {RootFields: mapstr.M{"id": keyBeta}},
		keyGamma: {RootFields: mapstr.M{"id": keyGamma}},
	}
	gen := &fakeGenerator{
		flushHook: func(time.Time) map[string]mb.Event {
			out := make(map[string]mb.Event, len(flushEvents))
			maps.Copy(out, flushEvents)
			return out
		},
		flushInterval: time.Minute,
	}
	_, ticker, reporter := newMetricSetWithFakeGenerator(t, gen)
	var attempt atomic.Int32
	published := sync.Map{}
	reporter.eventFn = func(e mb.Event) bool {
		id, ok := e.RootFields["id"].(string)
		require.True(t, ok, "test events must carry id for deterministic assertions")
		if attempt.Add(1) == 2 {
			return false
		}
		published.Store(id, struct{}{})
		return true
	}

	ticker.Tick(time.Unix(300, 0))

	require.Eventually(t, func() bool {
		gen.mu.Lock()
		defer gen.mu.Unlock()
		return len(gen.retainedFlushEvents) == 2
	}, time.Second, 5*time.Millisecond, "two events should remain unpublished after mid-flush rejection")

	publishedKeys := make(map[string]struct{})
	published.Range(func(k, _ interface{}) bool {
		key, ok := k.(string)
		require.True(t, ok, "published event keys must be strings")
		publishedKeys[key] = struct{}{}
		return true
	})

	gen.mu.Lock()
	retained := gen.retainedFlushEvents
	gen.mu.Unlock()

	assert.Len(t, publishedKeys, 1, "exactly one event should be published before rejection")
	assert.Len(t, retained, 2, "failed and not-yet-attempted events must be retained")

	for _, key := range allKeys {
		_, wasPublished := publishedKeys[key]
		_, wasRetained := retained[key]
		assert.NotEqual(t, wasPublished, wasRetained, "key %q must be exclusively published or retained", key)
		assert.True(t, wasPublished || wasRetained, "key %q must appear in published or retained union", key)
	}
}
