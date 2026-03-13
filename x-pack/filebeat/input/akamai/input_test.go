// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// mockBeatClient implements beat.Client for testing.
type mockBeatClient struct {
	mu     sync.Mutex
	events []beat.Event
}

func (m *mockBeatClient) Publish(e beat.Event) {
	m.PublishAll([]beat.Event{e})
}

func (m *mockBeatClient) PublishAll(es []beat.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, es...)
}

func (m *mockBeatClient) Close() error { return nil }

func (m *mockBeatClient) eventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

func baseTestConfig(serverURL string) config {
	cfg := defaultConfig()
	u, _ := url.Parse(serverURL)
	cfg.Resource.URL = &urlConfig{URL: u}
	cfg.ConfigIDs = "1"
	cfg.Auth.EdgeGrid = &edgeGridConfig{
		ClientToken:  "test-client-token",
		ClientSecret: "test-client-secret",
		AccessToken:  "test-access-token",
	}
	cfg.EventLimit = 2
	cfg.NumberOfWorkers = 2
	cfg.BatchSize = 1
	cfg.InvalidTimestampRetries = 1
	cfg.StreamBufferSize = 4
	return cfg
}

// --- StreamEvents unit tests ---

func TestStreamEventsSeparatesOffsetContext(t *testing.T) {
	raw := ndjson(
		`{"event":"a","offset":"off-a"}`,
		`{"event":"b","offset":"off-b"}`,
		`{"total":2,"offset":"next-off","limit":2}`,
	)

	eventCh := make(chan json.RawMessage, 10)
	pageCtx, count, err := StreamEvents(context.Background(), strings.NewReader(raw), eventCh)
	close(eventCh)

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, "next-off", pageCtx.Offset)
	assert.Equal(t, 2, pageCtx.Total)

	var events []string
	for raw := range eventCh {
		events = append(events, string(raw))
	}
	assert.Len(t, events, 2)
	assert.Contains(t, events[0], `"event":"a"`)
	assert.Contains(t, events[1], `"event":"b"`)
}

func TestStreamEventsFallbackToEvent(t *testing.T) {
	raw := ndjson(
		`{"event":"a","offset":"off-a"}`,
		`{"event":"b","offset":"off-b"}`,
	)

	eventCh := make(chan json.RawMessage, 10)
	pageCtx, count, err := StreamEvents(context.Background(), strings.NewReader(raw), eventCh)
	close(eventCh)

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Empty(t, pageCtx.Offset, "no offset context line, last line should be sent as event")
}

func TestStreamEventsEmptyBody(t *testing.T) {
	eventCh := make(chan json.RawMessage, 10)
	pageCtx, count, err := StreamEvents(context.Background(), strings.NewReader(""), eventCh)
	close(eventCh)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, pageCtx.Offset)
}

func TestStreamEventsSingleOffsetContextOnly(t *testing.T) {
	raw := `{"total":0,"offset":"empty-off","limit":2}` + "\n"

	eventCh := make(chan json.RawMessage, 10)
	pageCtx, count, err := StreamEvents(context.Background(), strings.NewReader(raw), eventCh)
	close(eventCh)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, "empty-off", pageCtx.Offset)
}

// --- Mock server infrastructure ---

type mockResponseStep struct {
	status int
	body   string
}

type mockInputServer struct {
	mu       sync.Mutex
	steps    []mockResponseStep
	requests []url.Values
}

func (s *mockInputServer) setScenario(steps []mockResponseStep) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.steps = steps
	s.requests = nil
}

func (s *mockInputServer) snapshotRequests() []url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]url.Values, len(s.requests))
	copy(out, s.requests)
	return out
}

func (s *mockInputServer) handler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	q := cloneValues(r.URL.Query())
	s.requests = append(s.requests, q)
	idx := len(s.requests) - 1
	if idx >= len(s.steps) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"detail":"unexpected request"}`))
		return
	}

	step := s.steps[idx]
	if step.status == 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(step.status)
	}
	_, _ = w.Write([]byte(step.body))
}

func cloneValues(in url.Values) url.Values {
	out := make(url.Values, len(in))
	for k, values := range in {
		cp := make([]string, len(values))
		copy(cp, values)
		out[k] = cp
	}
	return out
}

// --- Input scenario tests ---

type inputScenario struct {
	name                string
	steps               []mockResponseStep
	initial             cursor
	retries             int
	maxAttempts         *int
	maxRecoveryAttempts *int
	wantOffset          string
	wantCaughtUp        bool
	wantPublishedEvents int
	verifyReqs          func(t *testing.T, reqs []url.Values)
}

func TestInput(t *testing.T) {
	serverState := &mockInputServer{}
	srv := httptest.NewServer(http.HandlerFunc(serverState.handler))
	defer srv.Close()

	now := time.Now().Unix()
	chainFrom := now - 3600
	chainTo := now - apiSafetyBuffer

	tests := []inputScenario{
		{
			name: "paginates from time mode to offset mode",
			steps: []mockResponseStep{
				{body: ndjson(`{"message":"e1"}`, `{"message":"e2"}`, `{"total":2,"offset":"off-page-1","limit":2}`)},
				{body: ndjson(`{"message":"e3"}`, `{"total":1,"offset":"off-page-2","limit":2}`)},
			},
			wantOffset:          "off-page-2",
			wantCaughtUp:        true,
			wantPublishedEvents: 3,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 2)
				assert.Equal(t, "2", reqs[0].Get("limit"))
				assert.Empty(t, reqs[0].Get("offset"))
				assert.NotEmpty(t, reqs[0].Get("from"))
				assert.NotEmpty(t, reqs[0].Get("to"))

				assert.Equal(t, "off-page-1", reqs[1].Get("offset"))
				assert.Empty(t, reqs[1].Get("from"))
			},
		},
		{
			name: "drops expired offset and replays chain",
			steps: []mockResponseStep{
				{status: http.StatusRequestedRangeNotSatisfiable, body: `{"detail":"offset expired","status":416}`},
				{body: ndjson(`{"message":"recovered"}`, `{"total":1,"offset":"off-recovered","limit":2}`)},
			},
			initial: cursor{
				ChainFrom:        chainFrom,
				ChainTo:          chainTo,
				LastOffset:       "expired-offset",
				OffsetObtainedAt: time.Now(),
			},
			wantOffset:          "off-recovered",
			wantCaughtUp:        true,
			wantPublishedEvents: 1,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 2)
				assert.Equal(t, "expired-offset", reqs[0].Get("offset"))
				assert.Empty(t, reqs[1].Get("offset"))
				assert.NotEmpty(t, reqs[1].Get("from"))
				assert.NotEmpty(t, reqs[1].Get("to"))
			},
		},
		{
			name: "retries invalid timestamp and continues",
			steps: []mockResponseStep{
				{status: http.StatusBadRequest, body: `{"detail":"invalid timestamp","status":400}`},
				{body: ndjson(`{"message":"retry-success"}`, `{"total":1,"offset":"off-r1","limit":2}`)},
			},
			retries:             1,
			wantOffset:          "off-r1",
			wantCaughtUp:        true,
			wantPublishedEvents: 1,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 2)
			},
		},
		{
			name: "invalid timestamp retries exhausted triggers chain replay",
			steps: []mockResponseStep{
				{status: http.StatusBadRequest, body: `{"detail":"invalid timestamp","status":400}`},
				{status: http.StatusBadRequest, body: `{"detail":"invalid timestamp","status":400}`},
				{body: ndjson(`{"message":"recovered"}`, `{"total":1,"offset":"off-r2","limit":2}`)},
			},
			initial: cursor{
				ChainFrom:        chainFrom,
				ChainTo:          chainTo,
				LastOffset:       "stale-offset",
				OffsetObtainedAt: time.Now(),
			},
			retries:             1,
			wantOffset:          "off-r2",
			wantCaughtUp:        true,
			wantPublishedEvents: 1,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 3)
				assert.Equal(t, "stale-offset", reqs[0].Get("offset"))
				assert.Equal(t, "stale-offset", reqs[1].Get("offset"))
				assert.Empty(t, reqs[2].Get("offset"))
				assert.NotEmpty(t, reqs[2].Get("from"))
			},
		},
		{
			name: "stops on non recoverable 400",
			steps: []mockResponseStep{
				{status: http.StatusBadRequest, body: `{"detail":"invalid request payload","status":400}`},
			},
			wantOffset:          "",
			wantCaughtUp:        false,
			wantPublishedEvents: 0,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 1)
			},
		},
		{
			name: "stops when next page offset is missing",
			steps: []mockResponseStep{
				{body: ndjson(`{"event":"a"}`, `{"event":"b"}`)},
			},
			wantOffset:          "",
			wantCaughtUp:        false,
			wantPublishedEvents: 2,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 1)
			},
		},
		{
			name: "ends cycle on empty response",
			steps: []mockResponseStep{
				{body: ""},
			},
			wantOffset:          "",
			wantCaughtUp:        false,
			wantPublishedEvents: 0,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 1)
			},
		},
		{
			name: "handles server error response",
			steps: []mockResponseStep{
				{status: http.StatusInternalServerError, body: `{"detail":"server exploded","status":500}`},
			},
			maxAttempts:         intPtr(0),
			wantOffset:          "",
			wantCaughtUp:        false,
			wantPublishedEvents: 0,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 1)
			},
		},
		{
			name: "from too old triggers chain replay with clamp",
			steps: []mockResponseStep{
				{status: http.StatusBadRequest, body: `{"detail":"from parameter is out of range","status":400}`},
				{body: ndjson(`{"message":"clamped"}`, `{"total":1,"offset":"off-clamped","limit":2}`)},
			},
			wantOffset:          "off-clamped",
			wantCaughtUp:        true,
			wantPublishedEvents: 1,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 2)
				assert.NotEmpty(t, reqs[0].Get("from"))
				assert.NotEmpty(t, reqs[1].Get("from"), "retry should use time-based mode")
			},
		},
		{
			name: "terminates after max recovery attempts",
			steps: []mockResponseStep{
				{status: http.StatusRequestedRangeNotSatisfiable, body: `{"detail":"offset expired","status":416}`},
				{status: http.StatusRequestedRangeNotSatisfiable, body: `{"detail":"offset expired","status":416}`},
				{status: http.StatusRequestedRangeNotSatisfiable, body: `{"detail":"offset expired","status":416}`},
			},
			initial: cursor{
				ChainFrom:        chainFrom,
				ChainTo:          chainTo,
				LastOffset:       "expired-offset",
				OffsetObtainedAt: time.Now(),
			},
			maxRecoveryAttempts: intPtr(2),
			wantOffset:          "",
			wantCaughtUp:        false,
			wantPublishedEvents: 0,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				assert.Len(t, reqs, 2, "should stop after 2 recovery attempts")
			},
		},
		{
			name: "proactive offset TTL drop triggers chain replay",
			steps: []mockResponseStep{
				{body: ndjson(`{"message":"replayed"}`, `{"total":1,"offset":"off-ttl","limit":2}`)},
			},
			initial: cursor{
				ChainFrom:        chainFrom,
				ChainTo:          chainTo,
				LastOffset:       "old-offset",
				OffsetObtainedAt: time.Now().Add(-5 * time.Minute),
			},
			wantOffset:          "off-ttl",
			wantCaughtUp:        true,
			wantPublishedEvents: 1,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 1)
				assert.Empty(t, reqs[0].Get("offset"), "stale offset should not be used")
				assert.NotEmpty(t, reqs[0].Get("from"))
				assert.NotEmpty(t, reqs[0].Get("to"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runInputScenario(t, srv.URL, serverState, tc)
		})
	}
}

func intPtr(v int) *int {
	return &v
}

func runInputScenario(t *testing.T, serverURL string, serverState *mockInputServer, tc inputScenario) {
	t.Helper()

	serverState.setScenario(tc.steps)
	cfg := baseTestConfig(serverURL)
	cfg.InvalidTimestampRetries = tc.retries
	if tc.maxAttempts != nil {
		cfg.Resource.Retry.MaxAttempts = tc.maxAttempts
	}
	if tc.maxRecoveryAttempts != nil {
		cfg.MaxRecoveryAttempts = *tc.maxRecoveryAttempts
	}

	httpClient, err := NewClient(cfg, logp.NewNopLogger(), monitoring.NewRegistry())
	require.NoError(t, err)
	defer httpClient.Close()

	beatClient := &mockBeatClient{}
	acks := newACKHandler()
	defer acks.Close()

	poller := &siemPoller{
		cfg:        cfg,
		httpClient: httpClient,
		log:        logp.NewNopLogger(),
		client:     beatClient,
		acks:       acks,
		cursor:     tc.initial,
		env:        v2.Context{},
	}

	err = poller.poll(context.Background())
	require.NoError(t, err)

	reqs := serverState.snapshotRequests()
	tc.verifyReqs(t, reqs)
	assert.Equal(t, tc.wantOffset, poller.cursor.LastOffset)
	assert.Equal(t, tc.wantCaughtUp, poller.cursor.CaughtUp)
	assert.Equal(t, tc.wantPublishedEvents, beatClient.eventCount())
}

func ndjson(lines ...string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// --- StreamEvents context cancellation test ---

func TestStreamEventsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	eventCh := make(chan json.RawMessage, 1)

	raw := ndjson(
		`{"event":"a"}`,
		`{"event":"b"}`,
		`{"event":"c"}`,
		`{"total":3,"offset":"off","limit":3}`,
	)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, count, err := StreamEvents(ctx, strings.NewReader(raw), eventCh)
	close(eventCh)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, count, 3, "should not have sent all events before cancellation")
}

// --- Cursor method tests ---

func TestCursorIsOffsetStale(t *testing.T) {
	tests := []struct {
		name      string
		cursor    cursor
		ttl       time.Duration
		wantStale bool
	}{
		{
			name:      "no offset",
			cursor:    cursor{},
			ttl:       120 * time.Second,
			wantStale: false,
		},
		{
			name:      "ttl disabled",
			cursor:    cursor{LastOffset: "off", OffsetObtainedAt: time.Now().Add(-5 * time.Minute)},
			ttl:       0,
			wantStale: false,
		},
		{
			name:      "fresh offset",
			cursor:    cursor{LastOffset: "off", OffsetObtainedAt: time.Now()},
			ttl:       120 * time.Second,
			wantStale: false,
		},
		{
			name:      "stale offset",
			cursor:    cursor{LastOffset: "off", OffsetObtainedAt: time.Now().Add(-5 * time.Minute)},
			ttl:       120 * time.Second,
			wantStale: true,
		},
		{
			name:      "zero obtained at",
			cursor:    cursor{LastOffset: "off"},
			ttl:       120 * time.Second,
			wantStale: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantStale, tt.cursor.isOffsetStale(tt.ttl))
		})
	}
}

// --- buildFetchParams tests ---

func TestBuildFetchParamsBranches(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name       string
		cursor     cursor
		wantOffset bool
		wantTime   bool
	}{
		{
			name: "branch 1: chain draining with valid offset",
			cursor: cursor{
				ChainFrom:        now - 3600,
				ChainTo:          now - apiSafetyBuffer,
				LastOffset:       "valid-off",
				OffsetObtainedAt: time.Now(),
			},
			wantOffset: true,
		},
		{
			name: "branch 2: chain replay with stale offset",
			cursor: cursor{
				ChainFrom:        now - 3600,
				ChainTo:          now - apiSafetyBuffer,
				LastOffset:       "stale-off",
				OffsetObtainedAt: time.Now().Add(-5 * time.Minute),
			},
			wantTime: true,
		},
		{
			name: "branch 2: chain replay with no offset",
			cursor: cursor{
				ChainFrom: now - 3600,
				ChainTo:   now - apiSafetyBuffer,
			},
			wantTime: true,
		},
		{
			name:     "branch 3: first run (empty cursor)",
			cursor:   cursor{},
			wantTime: true,
		},
		{
			name: "branch 3: caught up starts new chain",
			cursor: cursor{
				ChainFrom: now - 3600,
				ChainTo:   now - apiSafetyBuffer,
				CaughtUp:  true,
			},
			wantTime: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poller := &siemPoller{
				cfg:    baseTestConfig("http://localhost"),
				log:    logp.NewNopLogger(),
				cursor: tt.cursor,
			}
			params := poller.buildFetchParams()

			if tt.wantOffset {
				assert.NotEmpty(t, params.Offset)
				assert.Zero(t, params.From)
				assert.Zero(t, params.To)
			}
			if tt.wantTime {
				assert.Empty(t, params.Offset)
				assert.NotZero(t, params.From)
				assert.NotZero(t, params.To)
			}
			assert.Equal(t, poller.cfg.EventLimit, params.Limit)
		})
	}
}

// --- Zero-copy event test ---

func TestCreateBeatEventZeroCopy(t *testing.T) {
	raw := json.RawMessage(`{"attackData":{"rule":"1234"},"httpMessage":{"host":"example.com"}}`)
	event := createBeatEvent(raw)

	msg, ok := event.Fields["message"].(string)
	require.True(t, ok)
	assert.Equal(t, string(raw), msg)
	assert.Len(t, event.Fields, 1, "only message field should exist, no unmarshal")
}

// --- Batch mode tests ---

func TestProcessPageBatchPublish(t *testing.T) {
	tests := []struct {
		name       string
		batchSize  int
		workers    int
		events     int
		wantEvents int
	}{
		{name: "batch_size=1", batchSize: 1, workers: 1, events: 3, wantEvents: 3},
		{name: "batch_size=2", batchSize: 2, workers: 1, events: 3, wantEvents: 3},
		{name: "batch_size=10 larger than events", batchSize: 10, workers: 1, events: 3, wantEvents: 3},
		{name: "multiple workers", batchSize: 1, workers: 3, events: 5, wantEvents: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beatClient := &mockBeatClient{}
			acks := newACKHandler()
			defer acks.Close()

			cfg := baseTestConfig("http://localhost")
			cfg.BatchSize = tt.batchSize
			cfg.NumberOfWorkers = tt.workers
			cfg.StreamBufferSize = tt.workers * 4

			poller := &siemPoller{
				cfg:    cfg,
				log:    logp.NewNopLogger(),
				client: beatClient,
				acks:   acks,
				cursor: cursor{ChainFrom: 100, ChainTo: 200},
				env:    v2.Context{},
			}

			lines := make([]string, tt.events)
			for i := range lines {
				lines[i] = `{"event":"` + string(rune('a'+i)) + `"}`
			}
			lines = append(lines, `{"total":3,"offset":"test-off","limit":10}`)
			body := strings.NewReader(ndjson(lines...))

			eventCount, pageCtx, err := poller.processPage(context.Background(), nopReadCloser{body})
			require.NoError(t, err)
			assert.Equal(t, tt.events, eventCount)
			assert.Equal(t, "test-off", pageCtx.Offset)
			assert.Equal(t, tt.wantEvents, beatClient.eventCount())
		})
	}
}

// nopReadCloser wraps an io.Reader to satisfy io.ReadCloser.
type nopReadCloser struct {
	r interface{ Read([]byte) (int, error) }
}

func (n nopReadCloser) Read(p []byte) (int, error) { return n.r.Read(p) }
func (n nopReadCloser) Close() error               { return nil }
