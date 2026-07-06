// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

var (
	// These flags are only used by TestCrowdstrikeFalconHose, which is a
	// manual integration test requiring real CrowdStrike credentials.
	timeout    = flag.Duration("crowdstrike_timeout", time.Minute, "time to allow Crowdstrike FalconHose test to run")
	offset     = flag.Int("crowdstrike_offset", -1, "offset into stream (negative to ignore)")
	cursorText = flag.String("cursor", "", "cursor JSON to inject into test")
)

// TestCrowdstrikeFalconHose is a manual integration test against a real
// CrowdStrike Falcon stream endpoint. It is skipped unless all required
// CROWDSTRIKE_* environment variables are set.
func TestCrowdstrikeFalconHose(t *testing.T) {
	logp.TestingSetup()
	logger := logp.L()

	feedURL, ok := os.LookupEnv("CROWDSTRIKE_URL")
	if !ok {
		t.Skip("crowdstrike tests require ${CROWDSTRIKE_URL} to be set")
	}
	tokenURL, ok := os.LookupEnv("CROWDSTRIKE_TOKEN_URL")
	if !ok {
		t.Skip("crowdstrike tests require ${CROWDSTRIKE_TOKEN_URL} to be set")
	}
	clientID, ok := os.LookupEnv("CROWDSTRIKE_CLIENT_ID")
	if !ok {
		t.Skip("crowdstrike tests require ${CROWDSTRIKE_CLIENT_ID} to be set")
	}
	clientSecret, ok := os.LookupEnv("CROWDSTRIKE_CLIENT_SECRET")
	if !ok {
		t.Skip("crowdstrike tests require ${CROWDSTRIKE_CLIENT_SECRET} to be set")
	}
	appID, ok := os.LookupEnv("CROWDSTRIKE_APPID")
	if !ok {
		t.Skip("crowdstrike tests require ${CROWDSTRIKE_APPID} to be set")
	}

	var state map[string]any
	if *cursorText != "" {
		var crsr any
		err := json.Unmarshal([]byte(*cursorText), &crsr)
		if err != nil {
			t.Fatalf("failed to parse cursor text: %v", err)
		}
		state = map[string]any{"cursor": crsr}
	}

	u, err := url.Parse(feedURL)
	if err != nil {
		t.Fatalf("unexpected error parsing feed url: %v", err)
	}
	cfg := config{
		Type: "crowdstrike",
		URL:  &urlConfig{u},
		Program: `
				state.response.decode_json().as(body,{
					"events": [body],
					"cursor": state.cursor.with({
						?state.feed: body.?metadata.optMap(m, {"offset": m.offset}),
					}),
				})`,
		Auth: authConfig{
			OAuth2: oAuth2Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				TokenURL:     tokenURL,
			},
		},
		CrowdstrikeAppID: appID,
		State:            state,
	}

	err = cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error validating config: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(*timeout, func() {
		cancel()
	})
	var cursor map[string]any
	if *offset >= 0 {
		cursor = map[string]any{"offset": *offset}
	}
	env := v2.Context{
		ID:              "crowdstrike_testing",
		MetricsRegistry: monitoring.NewRegistry(),
	}
	s, err := NewFalconHoseFollower(ctx, env, cfg, cursor, &testPublisher{logger}, nil, logger, time.Now)
	if err != nil {
		t.Fatalf("unexpected error constructing follower: %v", err)
	}
	err = s.FollowStream(ctx)
	if err != nil {
		t.Errorf("unexpected error following stream: %v", err)
	}
}

func TestFollowSessionRefreshDoesNotSpinForShortIntervals(t *testing.T) {
	// TODO: When the project baseline moves to Go 1.25+, rewrite this test with
	// testing/synctest. A fake clock would remove the manual timer/channel
	// wiring, making the async timing assertions simpler and more readable.
	t.Parallel()

	var (
		timer             = make(chan time.Time)
		refreshCalls      atomic.Int32
		refreshCallSignal = make(chan struct{}, 1)
		afterCalls        = make(chan time.Duration, 2)
	)

	after := func(d time.Duration) <-chan time.Time {
		// Capture the requested delay so we can assert scheduling intent.
		afterCalls <- d
		return timer
	}
	refresh := func() error {
		// Signal each refresh callback execution to the test goroutine.
		refreshCalls.Add(1)
		refreshCallSignal <- struct{}{}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		// Drive the loop with a controlled timer channel instead of sleeping.
		runRefreshLoopWithAfter(ctx, 15*time.Second, after, refresh)
		close(done)
	}()

	select {
	case d := <-afterCalls:
		if d != 15*time.Second {
			t.Fatalf("unexpected refresh wait duration: got %v, want %v", d, 15*time.Second)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first refresh timer")
	}
	if got := refreshCalls.Load(); got != 0 {
		t.Fatalf("unexpected refresh calls before first timer fire: got %d, want 0", got)
	}

	// Trigger the synthetic timer and verify exactly one refresh executes.
	timer <- time.Now()
	select {
	case <-refreshCallSignal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for refresh callback")
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("unexpected refresh calls after timer fire: got %d, want 1", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for refresh loop shutdown")
	}
}

func TestRefreshSessionWait(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		refreshAfter time.Duration
		want         time.Duration
	}{
		{
			name:         "long interval uses 90 percent rule",
			refreshAfter: 10 * time.Minute,
			want:         9 * time.Minute,
		},
		{
			name:         "short interval uses 90 percent rule",
			refreshAfter: 30 * time.Second,
			want:         27 * time.Second,
		},
		{
			name:         "very short interval uses minimum clamp",
			refreshAfter: 10 * time.Second,
			want:         15 * time.Second,
		},
		{
			name:         "zero interval uses minimum clamp",
			refreshAfter: 0,
			want:         15 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := refreshSessionWait(tt.refreshAfter)
			if got != tt.want {
				t.Fatalf("unexpected wait duration: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCrowdstrikeOAuthHTTPClientRespectsConfiguredTimeout(t *testing.T) {
	t.Parallel()

	const requestTimeout = 50 * time.Millisecond
	const tokenDelay = 200 * time.Millisecond
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(tokenDelay)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"token","token_type":"bearer","expires_in":3600}`)
	}))
	defer tokenSrv.Close()

	discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("discover endpoint should not be reached when token fetch times out")
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer discoverSrv.Close()

	u, err := url.Parse(discoverSrv.URL)
	if err != nil {
		t.Fatalf("failed to parse discover URL: %v", err)
	}
	cfg := config{
		Type: "crowdstrike",
		URL:  &urlConfig{u},
		Auth: authConfig{
			OAuth2: oAuth2Config{
				ClientID:     "id",
				ClientSecret: "secret",
				TokenURL:     tokenSrv.URL,
			},
		},
		CrowdstrikeAppID: "test",
		Retry:            &retry{MaxAttempts: 1, WaitMin: time.Millisecond, WaitMax: time.Millisecond},
		Transport:        httpcommon.HTTPTransportSettings{Timeout: requestTimeout},
		Program: `
			state.response.decode_json().as(body,{
				"events": [body],
			})`,
	}

	log := logp.NewNopLogger()
	env := v2.Context{
		ID:              "crowdstrike_timeout_test",
		MetricsRegistry: monitoring.NewRegistry(),
	}
	s, err := NewFalconHoseFollower(context.Background(), env, cfg, nil, &testPublisher{log}, nil, log, time.Now)
	if err != nil {
		t.Fatalf("failed to construct follower: %v", err)
	}

	start := time.Now()
	err = s.FollowStream(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected FollowStream to fail due to token request timeout, but it succeeded")
	}
	if elapsed > tokenDelay {
		t.Fatalf("expected FollowStream to fail before server delay %v, but it took %v: %v", tokenDelay, elapsed, err)
	}
}

func TestFollowStreamRetryCapHonorsMaxAttempts(t *testing.T) {
	log := logptest.NewTestingLogger(t, t.Name())

	discoverURL, tokenURL, hits := startEmptyDiscover(t)
	const maxAttempts = 15 // Deliberately > the unconfigured cap of 10.
	cfg := emptyDiscoverConfig(t, discoverURL, tokenURL, &retry{MaxAttempts: maxAttempts, WaitMin: time.Millisecond, WaitMax: time.Millisecond})
	s := newEmptyDiscoverFollower(t, cfg, nil, log)

	err := s.FollowStream(context.Background())
	if err == nil {
		t.Fatal("FollowStream() error = nil; want max-attempts error")
	}
	// Regression: a configured MaxAttempts greater than 10 must be honoured,
	// not silently capped at the unconfigured default of 10.
	want := fmt.Sprintf("max retry attempts (%d) exceeded", maxAttempts)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("FollowStream() error = %v; want substring %q", err, want)
	}
	if got := hits.Load(); got != int64(maxAttempts) {
		t.Errorf("discover attempts = %d; want %d", got, maxAttempts)
	}
}

func TestFollowStreamInfiniteRetriesDoesNotCap(t *testing.T) {
	log := logptest.NewTestingLogger(t, t.Name())

	discoverURL, tokenURL, hits := startEmptyDiscover(t)
	// MaxAttempts is intentionally tiny: InfiniteRetries must override it and
	// the unconfigured cap of 10 entirely.
	cfg := emptyDiscoverConfig(t, discoverURL, tokenURL, &retry{InfiniteRetries: true, MaxAttempts: 1, WaitMin: time.Millisecond, WaitMax: time.Millisecond})
	s := newEmptyDiscoverFollower(t, cfg, nil, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.FollowStream(ctx) }()

	// Wait until well past the unconfigured cap to prove the input keeps
	// retrying, then cancel.
	deadline := time.After(10 * time.Second)
	for hits.Load() <= 12 {
		select {
		case <-deadline:
			t.Fatalf("discover attempts = %d before timeout; want > 12", hits.Load())
		case err := <-done:
			t.Fatalf("FollowStream returned early after %d attempts: %v", hits.Load(), err)
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("FollowStream() error = %v; want nil after context cancel", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for FollowStream to return after cancel")
	}
}

func TestFollowStreamDefersDegraded(t *testing.T) {
	log := logptest.NewTestingLogger(t, t.Name())

	discoverURL, tokenURL, _ := startEmptyDiscover(t)
	// With MaxAttempts=5 and the degrade threshold of 3, the input reports
	// DEGRADED only on attempts 3 and 4, then terminates on attempt 5 without
	// reporting DEGRADED on the earlier transient failures.
	cfg := emptyDiscoverConfig(t, discoverURL, tokenURL, &retry{MaxAttempts: 5, WaitMin: time.Millisecond, WaitMax: time.Millisecond})
	rec := &degradedRecorder{}
	s := newEmptyDiscoverFollower(t, cfg, rec, log)

	if err := s.FollowStream(context.Background()); err == nil {
		t.Fatal("FollowStream() error = nil; want max-attempts error")
	}
	if got := rec.count(); got != 2 {
		t.Errorf("DEGRADED reports = %d; want 2 (attempts 3 and 4 only)", got)
	}
}

// startEmptyDiscover starts a token endpoint and a discover endpoint that
// returns 200 with an empty body (the transient EOF condition observed from
// CrowdStrike), returning the discover URL, token URL, and a counter of
// discover requests.
func startEmptyDiscover(t *testing.T) (discoverURL, tokenURL string, hits *atomic.Int64) {
	t.Helper()
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"token","token_type":"bearer","expires_in":3600}`)
	}))
	t.Cleanup(tokenSrv.Close)

	hits = new(atomic.Int64)
	discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
	}))
	t.Cleanup(discoverSrv.Close)
	return discoverSrv.URL, tokenSrv.URL, hits
}

func emptyDiscoverConfig(t *testing.T, discoverURL, tokenURL string, r *retry) config {
	t.Helper()
	u, err := url.Parse(discoverURL)
	if err != nil {
		t.Fatalf("failed to parse discover URL: %v", err)
	}
	return config{
		Type: "crowdstrike",
		URL:  &urlConfig{u},
		Auth: authConfig{
			OAuth2: oAuth2Config{
				ClientID:     "id",
				ClientSecret: "secret",
				TokenURL:     tokenURL,
			},
		},
		CrowdstrikeAppID: "test",
		Retry:            r,
		Program: `
			state.response.decode_json().as(body,{
				"events": [body],
			})`,
	}
}

func newEmptyDiscoverFollower(t *testing.T, cfg config, stat status.StatusReporter, log *logp.Logger) StreamFollower {
	t.Helper()
	env := v2.Context{ID: "crowdstrike_test", MetricsRegistry: monitoring.NewRegistry()}
	s, err := NewFalconHoseFollower(context.Background(), env, cfg, nil, &testPublisher{log}, stat, log, time.Now)
	if err != nil {
		t.Fatalf("failed to construct follower: %v", err)
	}
	return s
}

// degradedRecorder counts the number of times DEGRADED is reported.
type degradedRecorder struct {
	n atomic.Int64
}

func (r *degradedRecorder) UpdateStatus(s status.Status, _ string) {
	if s == status.Degraded {
		r.n.Add(1)
	}
}

func (r *degradedRecorder) count() int64 {
	return r.n.Load()
}

func TestFollowStreamCancelsRefreshOnReconnect(t *testing.T) {
	const totalSessions = 10

	var discoverCalls atomic.Int32

	goroutineSamples := make(chan int, totalSessions)

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"tok","token_type":"bearer","expires_in":3600}`)
	}))
	defer tokenSrv.Close()

	refreshSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer refreshSrv.Close()

	// Each feed is one event and then an EOF, triggering a reconnect.
	feedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fmt.Sprintf(`{"metadata":{"offset":%d,"eventType":"test"}}`, discoverCalls.Load()))
	}))
	defer feedSrv.Close()

	discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := discoverCalls.Add(1)
		if n > 1 && n <= totalSessions {
			// Give the previous session's canceled refresh goroutine
			// time to observe ctx.Done() and exit before sampling.
			time.Sleep(5 * time.Millisecond)
			goroutineSamples <- runtime.NumGoroutine()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
			"resources": [{
				"dataFeedURL": %q,
				"sessionToken": {"token": "tok", "expiration": "2099-01-01T00:00:00Z"},
				"refreshActiveSessionURL": %q,
				"refreshActiveSessionInterval": 30
			}],
			"meta": {}
		}`, feedSrv.URL, refreshSrv.URL)
	}))
	defer discoverSrv.Close()

	u, err := url.Parse(discoverSrv.URL)
	if err != nil {
		t.Fatalf("failed to parse discover URL: %v", err)
	}
	cfg := config{
		Type: "crowdstrike",
		URL:  &urlConfig{u},
		Auth: authConfig{
			OAuth2: oAuth2Config{
				ClientID:     "id",
				ClientSecret: "secret",
				TokenURL:     tokenSrv.URL,
			},
		},
		CrowdstrikeAppID: "test",
		Program:          `state.response.decode_json().as(body, {"events": [body]})`,
	}

	log := logptest.NewTestingLogger(t, "recon")
	env := v2.Context{
		ID:              "reconnect_test",
		MetricsRegistry: monitoring.NewRegistry(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := NewFalconHoseFollower(ctx, env, cfg, nil, &testPublisher{log}, nil, log, time.Now)
	if err != nil {
		t.Fatalf("failed to construct follower: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- s.FollowStream(ctx)
	}()

	// Collect goroutine samples from sessions 2 through totalSessions.
	// Session 1 has no predecessor, so skip it.
	samples := make([]int, 0, totalSessions-1)
	for i := range totalSessions - 1 {
		select {
		case n := <-goroutineSamples:
			samples = append(samples, n)
		case err := <-done:
			t.Fatalf("FollowStream exited early after %d sessions: %v", i+1, err)
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out waiting for session %d", i+2)
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for FollowStream to return")
	}

	if len(samples) != totalSessions-1 {
		t.Fatalf("collected %d goroutine samples; want %d", len(samples), totalSessions-1)
	}

	// With the fix, goroutine count stays roughly constant because each
	// session's refresh goroutine exits when followSession returns. Without
	// the fix, goroutines accumulate at one per session, giving growth ≈
	// totalSessions-1. The threshold is set at half that to tolerate
	// scheduling jitter while still catching the linear leak.
	growth := samples[len(samples)-1] - samples[0]
	if growth >= totalSessions/2 {
		t.Errorf("goroutine growth = %d; want < %d (samples: %v)",
			growth, totalSessions/2, samples)
	}
}

func TestSameOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		base string
		tgt  string
		want bool
	}{
		{
			name: "same_host",
			base: "https://api.crowdstrike.com/sensors",
			tgt:  "https://api.crowdstrike.com/other",
			want: true,
		},
		{
			name: "same_registrable_domain_different_subdomains",
			base: "https://api.crowdstrike.com/sensors",
			tgt:  "https://firehose.crowdstrike.com/feed",
			want: true,
		},
		{
			name: "different_registrable_domain",
			base: "https://api.crowdstrike.com/sensors",
			tgt:  "https://evil.example.com/capture",
			want: false,
		},
		{
			name: "scheme_downgrade_https_to_http",
			base: "https://api.crowdstrike.com/sensors",
			tgt:  "http://api.crowdstrike.com/sensors",
			want: false,
		},
		{
			name: "scheme_upgrade_http_to_https",
			base: "http://api.crowdstrike.com/sensors",
			tgt:  "https://api.crowdstrike.com/sensors",
			want: true,
		},
		{
			name: "same_IP_address",
			base: "https://192.168.1.1:8080/api",
			tgt:  "https://192.168.1.1:9090/feed",
			want: true,
		},
		{
			name: "different_IP_addresses",
			base: "https://192.168.1.1:8080/api",
			tgt:  "https://10.0.0.1:8080/api",
			want: false,
		},
		{
			name: "regional_subdomains_same_domain",
			base: "https://api.us-2.crowdstrike.com/sensors",
			tgt:  "https://firehose.us-2.crowdstrike.com/feed",
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			base, err := url.Parse(test.base)
			if err != nil {
				t.Fatalf("failed to parse base URL: %v", err)
			}
			tgt, err := url.Parse(test.tgt)
			if err != nil {
				t.Fatalf("failed to parse target URL: %v", err)
			}
			if got := sameOrigin(base, tgt); got != test.want {
				t.Errorf("sameOrigin(%q, %q) = %v, want %v", test.base, test.tgt, got, test.want)
			}
		})
	}
}

func TestAllowedOrigin(t *testing.T) {
	t.Parallel()

	parse := func(t *testing.T, raw string) *url.URL {
		t.Helper()
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("failed to parse URL %q: %v", raw, err)
		}
		return u
	}

	tests := []struct {
		name    string
		base    string
		allowed []string
		tgt     string
		want    bool
	}{
		{
			name: "same_origin_no_allowlist",
			base: "https://api.crowdstrike.com",
			tgt:  "https://firehose.crowdstrike.com/feed",
			want: true,
		},
		{
			name: "different_origin_no_allowlist",
			base: "https://api.crowdstrike.com",
			tgt:  "https://evil.example.com/capture",
			want: false,
		},
		{
			name:    "different_origin_allowed_by_allowlist",
			base:    "https://api.crowdstrike.com",
			allowed: []string{"https://evil.example.com"},
			tgt:     "https://evil.example.com/capture",
			want:    true,
		},
		{
			name:    "allowlist_scheme_mismatch",
			base:    "https://api.crowdstrike.com",
			allowed: []string{"https://streaming.newdomain.com"},
			tgt:     "http://streaming.newdomain.com/feed",
			want:    false,
		},
		{
			name:    "allowlist_host_mismatch",
			base:    "https://api.crowdstrike.com",
			allowed: []string{"https://streaming.newdomain.com"},
			tgt:     "https://other.newdomain.com/feed",
			want:    false,
		},
		{
			name:    "allowlist_matches_with_explicit_default_port",
			base:    "https://api.crowdstrike.com",
			allowed: []string{"https://streaming.newdomain.com"},
			tgt:     "https://streaming.newdomain.com:443/feed",
			want:    true,
		},
		{
			name:    "allowlist_rejects_non_default_port",
			base:    "https://api.crowdstrike.com",
			allowed: []string{"https://streaming.newdomain.com"},
			tgt:     "https://streaming.newdomain.com:8443/feed",
			want:    false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			base := parse(t, test.base)
			tgt := parse(t, test.tgt)
			allowed := make([]*url.URL, len(test.allowed))
			for i, a := range test.allowed {
				allowed[i] = parse(t, a)
			}
			if got := allowedOrigin(base, allowed, tgt); got != test.want {
				t.Errorf("allowedOrigin(%q, %v, %q) = %v, want %v", test.base, test.allowed, test.tgt, got, test.want)
			}
		})
	}
}

func TestFollowSessionRejectsCrossOriginResourceURLs(t *testing.T) {
	t.Parallel()

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"tok","token_type":"bearer","expires_in":3600}`)
	}))
	t.Cleanup(tokenSrv.Close)

	tests := []struct {
		name       string
		feedHost   string
		refreshURL string
		wantErr    string
	}{
		{
			name:     "cross_origin_feed_URL",
			feedHost: "https://evil.example.com",
			wantErr:  "feed url origin",
		},
		{
			name:       "cross_origin_refresh_URL",
			refreshURL: "https://evil.example.com/refresh",
			wantErr:    "refresh url origin",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			feedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"metadata":{"offset":1,"eventType":"test"}}`)
			}))
			defer feedSrv.Close()

			refreshSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer refreshSrv.Close()

			feedURL := feedSrv.URL
			if test.feedHost != "" {
				feedURL = test.feedHost + "/feed"
			}
			refreshURL := refreshSrv.URL
			if test.refreshURL != "" {
				refreshURL = test.refreshURL
			}

			discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{
					"resources": [{
						"dataFeedURL": %q,
						"sessionToken": {"token": "tok", "expiration": "2099-01-01T00:00:00Z"},
						"refreshActiveSessionURL": %q,
						"refreshActiveSessionInterval": 30
					}],
					"meta": {}
				}`, feedURL, refreshURL)
			}))
			defer discoverSrv.Close()

			u, err := url.Parse(discoverSrv.URL)
			if err != nil {
				t.Fatalf("failed to parse discover URL: %v", err)
			}
			cfg := config{
				Type: "crowdstrike",
				URL:  &urlConfig{u},
				Auth: authConfig{
					OAuth2: oAuth2Config{
						ClientID:     "id",
						ClientSecret: "secret",
						TokenURL:     tokenSrv.URL,
					},
				},
				CrowdstrikeAppID: "test",
				Retry:            &retry{MaxAttempts: 1, WaitMin: time.Millisecond, WaitMax: time.Millisecond},
				Program:          `state.response.decode_json().as(body, {"events": [body]})`,
			}

			log := logptest.NewTestingLogger(t, "origin")
			env := v2.Context{
				ID:              "origin_test",
				MetricsRegistry: monitoring.NewRegistry(),
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			s, err := NewFalconHoseFollower(ctx, env, cfg, nil, &testPublisher{log}, nil, log, time.Now)
			if err != nil {
				t.Fatalf("failed to construct follower: %v", err)
			}

			err = s.FollowStream(ctx)
			if err == nil {
				t.Fatal("expected FollowStream to fail with origin mismatch, but it succeeded")
			}
			if !errors.Is(err, hardError{}) {
				t.Errorf("expected hardError, got: %v", err)
			}
			if got := err.Error(); !strings.Contains(got, test.wantErr) {
				t.Errorf("error %q does not contain %q", got, test.wantErr)
			}
		})
	}
}

type testPublisher struct {
	log *logp.Logger
}

var _ cursor.Publisher = testPublisher{}

func (p testPublisher) Publish(e beat.Event, cursor any) error {
	p.log.Infow("publish", "event", e.Fields, "cursor", cursor)
	return nil
}
