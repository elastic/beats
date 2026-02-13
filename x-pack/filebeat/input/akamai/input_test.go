// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type mockPublisher struct {
	mu           sync.Mutex
	events       []beat.Event
	cursors      []interface{}
	failMessages map[string]bool
	failCount    int
}

// Publish captures successful events and optionally injects publish failures
// based on event.message so partial-failure paths can be exercised.
func (m *mockPublisher) Publish(event beat.Event, cursor interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if msg, ok := event.Fields["message"].(string); ok && m.failMessages[msg] {
		m.failCount++
		return errors.New("mock publish failure")
	}
	m.events = append(m.events, event)
	m.cursors = append(m.cursors, cursor)
	return nil
}

// baseTestConfig returns a minimal valid Akamai config for mock-server tests.
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
	cfg.InvalidTimestampRetries = 1
	return cfg
}

// TestParseResponseSeparatesOffsetContext verifies that event lines are kept as
// events and the trailing metadata line is parsed as pagination context.
func TestParseResponseSeparatesOffsetContext(t *testing.T) {
	raw := strings.Join([]string{
		`{"event":"a","offset":"off-a"}`,
		`{"event":"b","offset":"off-b"}`,
		`{"total":2,"offset":"next-off","limit":2}`,
	}, "\n")

	client := &Client{log: logp.NewNopLogger()}
	resp, err := client.parseResponse(strings.NewReader(raw), 2)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.Events, 2)
	assert.Equal(t, "next-off", resp.LastOffset)
	assert.True(t, resp.HasMore)
	assert.Contains(t, string(resp.Events[0].Raw), `"event":"a"`)
	assert.Contains(t, string(resp.Events[1].Raw), `"event":"b"`)
}

// TestParseResponseFallbackToLastEventOffset verifies fallback pagination when
// response metadata is missing and only event offsets are present.
func TestParseResponseFallbackToLastEventOffset(t *testing.T) {
	raw := strings.Join([]string{
		`{"event":"a","offset":"off-a"}`,
		`{"event":"b","offset":"off-b"}`,
	}, "\n")

	client := &Client{log: logp.NewNopLogger()}
	resp, err := client.parseResponse(strings.NewReader(raw), 10)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.Events, 2)
	assert.Equal(t, "off-b", resp.LastOffset)
	assert.False(t, resp.HasMore)
}

// mockResponseStep represents one HTTP response emitted by the mock server in
// strict request order for a scenario.
type mockResponseStep struct {
	status int
	body   string
}

// mockInputServer serves scripted responses and records query params so tests
// can assert time-mode vs offset-mode fetch transitions.
type mockInputServer struct {
	mu       sync.Mutex
	steps    []mockResponseStep
	requests []url.Values
}

// setScenario resets captured requests and loads the ordered response script.
func (s *mockInputServer) setScenario(steps []mockResponseStep) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.steps = steps
	s.requests = nil
}

// snapshotRequests returns a copy of captured query params for assertions.
func (s *mockInputServer) snapshotRequests() []url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]url.Values, len(s.requests))
	copy(out, s.requests)
	return out
}

// handler emits the next scripted response; extra calls are treated as test
// harness errors and return 500.
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

// cloneValues deep-copies query params to avoid mutation across assertions.
func cloneValues(in url.Values) url.Values {
	out := make(url.Values, len(in))
	for k, values := range in {
		cp := make([]string, len(values))
		copy(cp, values)
		out[k] = cp
	}
	return out
}

// inputScenario defines one end-to-end polling flow.
type inputScenario struct {
	name                string
	steps               []mockResponseStep
	initial             cursor
	retries             int
	maxAttempts         *int
	failMessages        map[string]bool
	wantOffset          string
	wantRecovery        bool
	wantPublishedEvents int
	wantPublishError    bool
	verifyReqs          func(t *testing.T, reqs []url.Values)
}

// TestInput validates end-to-end poll behavior with a shared mock server using
// table-driven scenarios for pagination, recovery, and failure paths.
func TestInput(t *testing.T) {
	serverState := &mockInputServer{}
	srv := httptest.NewServer(http.HandlerFunc(serverState.handler))
	defer srv.Close()

	tests := []inputScenario{
		{
			name: "paginates from time mode to offset mode",
			steps: []mockResponseStep{
				{body: ndjson(`{"message":"e1","offset":"off-e1"}`, `{"message":"e2","offset":"off-e2"}`, `{"total":2,"offset":"off-page-1","limit":2}`)},
				{body: ndjson(`{"message":"e3","offset":"off-e3"}`, `{"total":1,"offset":"off-page-2","limit":2}`)},
			},
			wantOffset:          "off-page-2",
			wantRecovery:        false,
			wantPublishedEvents: 3,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 2)
				assert.Equal(t, "2", reqs[0].Get("limit"))
				assert.Empty(t, reqs[0].Get("offset"))
				assert.NotEmpty(t, reqs[0].Get("from"))
				assert.NotEmpty(t, reqs[0].Get("to"))

				assert.Equal(t, "2", reqs[1].Get("limit"))
				assert.Equal(t, "off-page-1", reqs[1].Get("offset"))
				assert.Empty(t, reqs[1].Get("from"))
				assert.Empty(t, reqs[1].Get("to"))
			},
		},
		{
			name: "drops expired offset and recovers in time mode",
			steps: []mockResponseStep{
				{status: http.StatusRequestedRangeNotSatisfiable, body: `{"detail":"offset expired","status":416}`},
				{body: ndjson(`{"message":"recovered","offset":"off-recovered"}`, `{"total":1,"offset":"off-page-recovered","limit":2}`)},
			},
			initial:             cursor{LastOffset: "expired-offset"},
			wantOffset:          "off-page-recovered",
			wantRecovery:        false,
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
				{body: ndjson(`{"message":"retry-success","offset":"off-r1"}`, `{"total":1,"offset":"off-page-r1","limit":2}`)},
			},
			retries:             1,
			wantOffset:          "off-page-r1",
			wantRecovery:        false,
			wantPublishedEvents: 1,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 2)
				assert.Equal(t, reqs[0].Get("offset"), reqs[1].Get("offset"))
			},
		},
		{
			name: "enters recovery mode when invalid timestamp retries are exhausted",
			steps: []mockResponseStep{
				{status: http.StatusBadRequest, body: `{"detail":"invalid timestamp","status":400}`},
				{status: http.StatusBadRequest, body: `{"detail":"invalid timestamp","status":400}`},
				{body: ndjson(`{"message":"recovered-after-invalid-ts","offset":"off-r2"}`, `{"total":1,"offset":"off-page-r2","limit":2}`)},
			},
			initial:             cursor{LastOffset: "stale-offset"},
			retries:             1,
			wantOffset:          "off-page-r2",
			wantRecovery:        false,
			wantPublishedEvents: 1,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 3)
				assert.Equal(t, "stale-offset", reqs[0].Get("offset"))
				assert.Equal(t, "stale-offset", reqs[1].Get("offset"))
				assert.Empty(t, reqs[2].Get("offset"))
				assert.NotEmpty(t, reqs[2].Get("from"))
				assert.NotEmpty(t, reqs[2].Get("to"))
			},
		},
		{
			name: "stops on non recoverable 400",
			steps: []mockResponseStep{
				{status: http.StatusBadRequest, body: `{"detail":"invalid request payload","status":400}`},
			},
			wantOffset:          "",
			wantRecovery:        false,
			wantPublishedEvents: 0,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 1)
				assert.Empty(t, reqs[0].Get("offset"))
				assert.NotEmpty(t, reqs[0].Get("from"))
				assert.NotEmpty(t, reqs[0].Get("to"))
			},
		},
		{
			name: "stops when next page offset is missing",
			steps: []mockResponseStep{
				{body: ndjson(`{"event":"a"}`, `{"event":"b"}`)},
			},
			wantOffset:          "",
			wantRecovery:        false,
			wantPublishedEvents: 2,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 1)
				assert.Empty(t, reqs[0].Get("offset"))
				assert.NotEmpty(t, reqs[0].Get("from"))
				assert.NotEmpty(t, reqs[0].Get("to"))
			},
		},
		{
			name: "ends cycle on empty response",
			steps: []mockResponseStep{
				{body: ""},
			},
			wantOffset:          "",
			wantRecovery:        false,
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
			wantRecovery:        false,
			wantPublishedEvents: 0,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 1)
			},
		},
		{
			name: "advances cursor despite partial publish failures",
			steps: []mockResponseStep{
				{body: ndjson(`{"message":"ok-1","offset":"o1"}`, `{"message":"drop-me","offset":"o2"}`, `{"total":2,"offset":"off-page-pf","limit":2}`)},
				{body: ndjson(`{"message":"ok-2","offset":"o3"}`, `{"total":1,"offset":"off-page-pf-2","limit":2}`)},
			},
			failMessages:        map[string]bool{"drop-me": true},
			wantOffset:          "off-page-pf-2",
			wantRecovery:        false,
			wantPublishedEvents: 2,
			wantPublishError:    true,
			verifyReqs: func(t *testing.T, reqs []url.Values) {
				require.Len(t, reqs, 2)
				assert.Equal(t, "off-page-pf", reqs[1].Get("offset"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runInputScenario(t, srv.URL, serverState, tc)
		})
	}
}

// intPtr is a small helper for optional int pointer fields in scenarios.
func intPtr(v int) *int {
	return &v
}

// runInputScenario executes a single input scenario and validates request
// sequence, cursor state, and publish outcomes.
func runInputScenario(t *testing.T, serverURL string, serverState *mockInputServer, tc inputScenario) {
	t.Helper()

	serverState.setScenario(tc.steps)
	cfg := baseTestConfig(serverURL)
	cfg.InvalidTimestampRetries = tc.retries
	if tc.maxAttempts != nil {
		cfg.Resource.Retry.MaxAttempts = tc.maxAttempts
	}

	client, err := NewClient(cfg, logp.NewNopLogger(), monitoring.NewRegistry())
	require.NoError(t, err)
	defer client.Close()

	pub := &mockPublisher{failMessages: tc.failMessages}
	poller := &siemPoller{
		cfg:    cfg,
		client: client,
		log:    logp.NewNopLogger(),
		pub:    pub,
		cursor: tc.initial,
		env:    v2.Context{},
	}

	err = poller.poll(context.Background())
	require.NoError(t, err)

	reqs := serverState.snapshotRequests()
	tc.verifyReqs(t, reqs)
	assert.Equal(t, tc.wantOffset, poller.cursor.LastOffset)
	assert.Equal(t, tc.wantRecovery, poller.cursor.RecoveryMode)
	assert.Len(t, pub.events, tc.wantPublishedEvents)
	assert.Equal(t, tc.wantPublishError, pub.failCount > 0)
}

// ndjson builds a newline-delimited JSON payload expected by the SIEM API.
func ndjson(lines ...string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
