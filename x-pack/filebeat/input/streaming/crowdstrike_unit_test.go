// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
)

func TestFollowSession_FirehoseHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "400_plain_text", statusCode: 400, body: "400 Bad Request"},
		{name: "401_unauthorized", statusCode: 401, body: `{"errors":[{"code":401,"message":"access denied"}]}`},
		{name: "500_internal", statusCode: 500, body: "Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()

			discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, discoverResponse(t, srv.URL+"/firehose", srv.URL+"/refresh"))
			}))
			defer discoverSrv.Close()

			s := newTestStream(t, discoverSrv.URL, srv.Client())
			state := map[string]any{}
			state, err := s.followSession(context.Background(), discoverSrv.Client(), state)
			if err == nil {
				t.Fatal("expected error from followSession, got nil")
			}
			if !strings.Contains(err.Error(), "unsuccessful firehose request") {
				t.Errorf("expected 'unsuccessful firehose request' error, got: %v", err)
			}
			if state == nil {
				t.Error("expected non-nil state on non-hard error")
			}
		})
	}
}

func TestFollowSession_EmptyDiscoverBody(t *testing.T) {
	discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A 200 OK with an empty body, as observed from the CrowdStrike
		// discover endpoint; Decode returns io.EOF.
		w.Header().Set("Content-Type", "application/json")
	}))
	defer discoverSrv.Close()

	s := newTestStream(t, discoverSrv.URL, discoverSrv.Client())
	state, err := s.followSession(context.Background(), discoverSrv.Client(), map[string]any{})
	if err == nil {
		t.Fatal("expected error from followSession, got nil")
	}
	if want := "discover stream returned an empty body"; !strings.Contains(err.Error(), want) {
		t.Errorf("followSession() error = %v; want substring %q", err, want)
	}
	// The empty body is a transient condition so the retry loop keeps trying
	// rather than terminating the input.
	if !errors.Is(err, transientError{}) {
		t.Errorf("followSession() error = %v; want transientError", err)
	}
	if state == nil {
		t.Error("expected non-nil state on non-hard error")
	}
}

func TestFollowSession_DiscoverGETFailureIsTransient(t *testing.T) {
	// Point at a server that is immediately closed so the discover GET fails
	// at the connection level.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	client := srv.Client()
	discoverURL := srv.URL
	srv.Close()

	s := newTestStream(t, discoverURL, client)
	state, err := s.followSession(context.Background(), client, map[string]any{})
	if err == nil {
		t.Fatal("expected error from followSession, got nil")
	}
	// A connection-level GET failure is transient, not input-terminating.
	if !errors.Is(err, transientError{}) {
		t.Errorf("followSession() error = %v; want transientError", err)
	}
	if errors.Is(err, hardError{}) {
		t.Errorf("followSession() error = %v; want non-hard error", err)
	}
	if state == nil {
		t.Error("expected non-nil state on non-hard error")
	}
}

func TestFollowSession_NonObjectMessage(t *testing.T) {
	validEvent := `{"metadata":{"eventType":"TestEvent","offset":1},"event":{"TestField":"value"}}`

	tests := []struct {
		name          string
		body          string
		wantPublished int
	}{
		{
			name:          "bare_number_skipped",
			body:          "400\n",
			wantPublished: 0,
		},
		{
			name:          "bare_string_skipped",
			body:          `"error"` + "\n",
			wantPublished: 0,
		},
		{
			name:          "array_skipped",
			body:          `[1,2,3]` + "\n",
			wantPublished: 0,
		},
		{
			name:          "non_object_then_valid_event",
			body:          "400\n" + validEvent + "\n",
			wantPublished: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firehoseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tt.body)
			}))
			defer firehoseSrv.Close()

			discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, discoverResponse(t, firehoseSrv.URL+"/firehose", firehoseSrv.URL+"/refresh"))
			}))
			defer discoverSrv.Close()

			pub := new(countingPublisher)
			s := newTestStreamWithPublisher(t, discoverSrv.URL, firehoseSrv.Client(), pub)
			state := map[string]any{}
			_, err := s.followSession(context.Background(), discoverSrv.Client(), state)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pub.published() != tt.wantPublished {
				t.Errorf("expected %d published events, got %d", tt.wantPublished, pub.published())
			}
		})
	}
}

func TestUserAgentTransport(t *testing.T) {
	const want = "Elastic-crowdstrike/4.0.0"

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	cli := srv.Client()
	cli.Transport = userAgentTransport{ua: want, base: cli.Transport}

	// http.Client.Do sets "Go-http-client/1.1" before RoundTrip;
	// the transport must overwrite it.
	req, err := http.NewRequestWithContext(t.Context(), "GET", srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got != want {
		t.Errorf("User-Agent = %q; want %q", got, want)
	}
}

func TestFollowSession_UserAgent(t *testing.T) {
	const want = "Elastic-crowdstrike/4.0.0"

	type requestRecord struct {
		path      string
		userAgent string
	}
	var (
		mu       sync.Mutex
		requests []requestRecord
	)
	record := func(r *http.Request) {
		mu.Lock()
		requests = append(requests, requestRecord{path: r.URL.Path, userAgent: r.Header.Get("User-Agent")})
		mu.Unlock()
	}

	firehoseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record(r)
		switch {
		case strings.HasPrefix(r.URL.Path, "/firehose"):
			// Send one event then EOF to end the session cleanly.
			fmt.Fprintln(w, `{"metadata":{"eventType":"Test","offset":1},"event":{"field":"value"}}`)
		case strings.HasPrefix(r.URL.Path, "/refresh"):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer firehoseSrv.Close()

	discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record(r)
		w.Header().Set("Content-Type", "application/json")
		// refreshActiveSessionInterval is short but refreshSessionWait
		// clamps to a 15s floor, so no refresh fires during the test.
		resp := map[string]any{
			"resources": []map[string]any{
				{
					"dataFeedURL": firehoseSrv.URL + "/firehose",
					"sessionToken": map[string]any{
						"token":      "test-token",
						"expiration": "2099-01-01T00:00:00Z",
					},
					"refreshActiveSessionURL":      firehoseSrv.URL + "/refresh",
					"refreshActiveSessionInterval": 1,
				},
			},
			"meta": map[string]any{},
		}
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer discoverSrv.Close()

	// Build clients wrapped with the userAgentTransport, matching the
	// production wiring in NewFalconHoseFollower.
	authClient := discoverSrv.Client()
	authClient.Transport = userAgentTransport{ua: want, base: authClient.Transport}
	plainClient := firehoseSrv.Client()
	plainClient.Transport = userAgentTransport{ua: want, base: plainClient.Transport}

	pub := new(countingPublisher)
	s := newTestStreamWithPublisher(t, discoverSrv.URL, plainClient, pub)

	state := map[string]any{}
	_, err := s.followSession(context.Background(), authClient, state)
	if err != nil {
		t.Fatalf("followSession() unexpected error: %v", err)
	}

	// Check that the discover and firehose requests carried the custom
	// User-Agent. We don't assert on refresh because the goroutine may
	// or may not fire within the test window.
	mu.Lock()
	defer mu.Unlock()
	var sawDiscover, sawFirehose bool
	for _, r := range requests {
		if r.userAgent != want {
			t.Errorf("request to %s had User-Agent = %q; want %q", r.path, r.userAgent, want)
		}
		switch {
		case r.path == "/" || r.path == "":
			sawDiscover = true
		case strings.HasPrefix(r.path, "/firehose"):
			sawFirehose = true
		}
	}
	if !sawDiscover {
		t.Error("no discover request recorded")
	}
	if !sawFirehose {
		t.Error("no firehose request recorded")
	}
}

func TestFollowSessionProcessesAllDiscoveredResources(t *testing.T) {
	// Hold the first feed open after one event so a sequential follower
	// would starve the second resource. Concurrent following should
	// still publish both events within a bounded time.
	release := make(chan struct{})

	feed1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("ResponseWriter does not implement http.Flusher")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"metadata":{"eventType":"Test","offset":1},"event":{"feed":"one"}}`)
		flusher.Flush()
		<-release
	}))
	t.Cleanup(func() {
		close(release)
		feed1.Close()
	})

	feed2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"metadata":{"eventType":"Test","offset":2},"event":{"feed":"two"}}`)
	}))
	defer feed2.Close()

	refreshSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer refreshSrv.Close()

	discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []map[string]any{
				{
					"dataFeedURL": feed1.URL + "/firehose",
					"sessionToken": map[string]any{
						"token":      "test-token",
						"expiration": "2099-01-01T00:00:00Z",
					},
					"refreshActiveSessionURL":      refreshSrv.URL + "/refresh",
					"refreshActiveSessionInterval": 1800,
				},
				{
					"dataFeedURL": feed2.URL + "/firehose",
					"sessionToken": map[string]any{
						"token":      "test-token",
						"expiration": "2099-01-01T00:00:00Z",
					},
					"refreshActiveSessionURL":      refreshSrv.URL + "/refresh",
					"refreshActiveSessionInterval": 1800,
				},
			},
			"meta": map[string]any{},
		}
		b, err := json.Marshal(resp)
		if err != nil {
			t.Errorf("failed to marshal discover response: %v", err)
			return
		}
		_, _ = w.Write(b)
	}))
	defer discoverSrv.Close()

	pub := new(recordingPublisher)
	s := newTestStreamWithPublisher(t, discoverSrv.URL, http.DefaultClient, pub)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := s.followSession(ctx, discoverSrv.Client(), map[string]any{})
		done <- err
	}()

	deadline := time.After(3 * time.Second)
	for pub.published() < 2 {
		select {
		case <-deadline:
			t.Fatalf("expected events from both feeds within 3s, got %d", pub.published())
		case err := <-done:
			if pub.published() < 2 {
				t.Fatalf("followSession returned before both feeds published: err=%v published=%d", err, pub.published())
			}
		case <-time.After(10 * time.Millisecond):
		}
	}

	got := map[string]bool{}
	for _, e := range pub.snapshot() {
		ev, _ := e.Fields["event"].(map[string]any)
		if ev == nil {
			continue
		}
		if name, ok := ev["feed"].(string); ok {
			got[name] = true
		}
	}
	if !got["one"] || !got["two"] {
		t.Errorf("expected events from feeds one and two, got %v", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("followSession did not return after context cancel")
	}
}

func discoverResponse(t *testing.T, feedURL, refreshURL string) string {
	t.Helper()
	resp := map[string]any{
		"resources": []map[string]any{
			{
				"dataFeedURL": feedURL,
				"sessionToken": map[string]any{
					"token":      "test-token",
					"expiration": "2099-01-01T00:00:00Z",
				},
				"refreshActiveSessionURL":      refreshURL,
				"refreshActiveSessionInterval": 1800,
			},
		},
		"meta": map[string]any{},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal discover response: %v", err)
	}
	return string(b)
}

func newTestStream(t *testing.T, discoverURL string, firehoseClient *http.Client) *falconHoseStream {
	t.Helper()
	return newTestStreamWithPublisher(t, discoverURL, firehoseClient, new(countingPublisher))
}

func newTestStreamWithPublisher(t *testing.T, discoverURL string, firehoseClient *http.Client, pub cursor.Publisher) *falconHoseStream {
	t.Helper()
	log := logptest.NewTestingLogger(t, t.Name())
	reg := monitoring.NewRegistry()
	m := newInputMetrics(reg, log)

	ctx := context.Background()
	prg, ast, err := newProgram(ctx, `
		state.response.decode_json().as(body, {
			"events": [body],
			?"cursor": has(body.metadata) ?
				optional.of(state.?cursor.orValue({}).with({
					?state.feed: body.?metadata.optMap(m, {"offset": m.offset}),
				}))
			:
				state.?cursor,
		})
	`, root, nil, "", log)
	if err != nil {
		t.Fatalf("failed to compile CEL program: %v", err)
	}

	return &falconHoseStream{
		cfg:         config{},
		discoverURL: discoverURL,
		plainClient: firehoseClient,
		status:      noopReporter{},
		processor: processor{
			ns:      "test",
			pub:     pub,
			log:     log,
			metrics: m,
			prg:     prg,
			ast:     ast,
		},
		time: time.Now,
	}
}

type countingPublisher struct {
	n atomic.Int64
}

func (p *countingPublisher) Publish(beat.Event, any) error {
	p.n.Add(1)
	return nil
}

func (p *countingPublisher) published() int {
	return int(p.n.Load())
}

type recordingPublisher struct {
	mu     sync.Mutex
	events []beat.Event
}

func (p *recordingPublisher) Publish(e beat.Event, _ any) error {
	p.mu.Lock()
	p.events = append(p.events, e)
	p.mu.Unlock()
	return nil
}

func (p *recordingPublisher) published() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.events)
}

func (p *recordingPublisher) snapshot() []beat.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]beat.Event, len(p.events))
	copy(out, p.events)
	return out
}
