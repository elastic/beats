// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	ref := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	fallback := 60 * time.Second

	tests := []struct {
		name string
		val  string
		want time.Duration
	}{
		{name: "empty", val: "", want: fallback},
		{name: "seconds", val: "30", want: 30 * time.Second},
		{name: "large_seconds", val: "120", want: 120 * time.Second},
		{name: "zero", val: "0", want: fallback},
		{name: "negative", val: "-5", want: fallback},
		{name: "http_date_future", val: "Thu, 13 Mar 2026 12:01:00 GMT", want: 60 * time.Second},
		{name: "http_date_past", val: "Thu, 13 Mar 2026 11:59:00 GMT", want: fallback},
		{name: "invalid", val: "garbage", want: fallback},
		{name: "whitespace", val: "  30  ", want: 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseRetryAfter(tt.val, fallback, ref)
			if got != tt.want {
				t.Errorf("parseRetryAfter(%q, %v, _) = %v, want %v", tt.val, fallback, got, tt.want)
			}
		})
	}
}

func TestRateLimitTransport(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")

	t.Run("passthrough_200", func(t *testing.T) {
		t.Parallel()
		ft := &fakeTransport{statuses: []int{200}}
		rt := &rateLimitTransport{base: ft, maxRetry: 3, wait: time.Millisecond, log: log}

		req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
		got, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer got.Body.Close()
		if got.StatusCode != 200 {
			t.Errorf("got status %d, want 200", got.StatusCode)
		}
		if ft.call != 1 {
			t.Errorf("got %d calls, want 1", ft.call)
		}
	})

	t.Run("retry_then_success", func(t *testing.T) {
		t.Parallel()
		ft := &fakeTransport{statuses: []int{429, 429, 200}}
		rt := &rateLimitTransport{base: ft, maxRetry: 3, wait: time.Millisecond, log: log}

		req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
		got, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer got.Body.Close()
		if got.StatusCode != 200 {
			t.Errorf("got status %d, want 200", got.StatusCode)
		}
		if ft.call != 3 {
			t.Errorf("got %d calls, want 3", ft.call)
		}
	})

	t.Run("max_retries_exceeded", func(t *testing.T) {
		t.Parallel()
		ft := &fakeTransport{statuses: []int{429, 429, 429, 429}}
		rt := &rateLimitTransport{base: ft, maxRetry: 3, wait: time.Millisecond, log: log}

		req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
		got, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer got.Body.Close()
		if got.StatusCode != http.StatusTooManyRequests {
			t.Errorf("got status %d, want 429", got.StatusCode)
		}
		if ft.call != 4 {
			t.Errorf("got %d calls, want 4", ft.call)
		}
	})

	t.Run("context_cancelled_during_wait", func(t *testing.T) {
		t.Parallel()
		ft := &fakeTransport{statuses: []int{429, 429}}
		rt := &rateLimitTransport{base: ft, maxRetry: 3, wait: time.Hour, log: log}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
		_, err := rt.RoundTrip(req)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("got error %v, want context.Canceled", err)
		}
	})
}

func TestRateLimitTransportBodyReplay(t *testing.T) {
	t.Parallel()

	log := logptest.NewTestingLogger(t, "")
	ft := &fakeTransport{statuses: []int{429, 200}}
	rt := &rateLimitTransport{base: ft, maxRetry: 3, wait: time.Millisecond, log: log}

	body := "grant_type=client_credentials&client_id=test"
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "http://example.com/token", io.NopCloser(strings.NewReader(body)))
	got, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer got.Body.Close()
	if got.StatusCode != 200 {
		t.Errorf("got status %d, want 200", got.StatusCode)
	}
	if ft.call != 2 {
		t.Errorf("got %d calls, want 2", ft.call)
	}
	for i, b := range ft.bodies {
		if !bytes.Equal(b, []byte(body)) {
			t.Errorf("call %d: body = %q, want %q", i, b, body)
		}
	}
}

// fakeTransport returns canned responses built from statuses.
// Responses are constructed inside RoundTrip so that bodyclose
// can trace each *http.Response to the caller.
type fakeTransport struct {
	statuses []int
	call     int
	bodies   [][]byte // captured request bodies, indexed by call
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := f.call
	f.call++
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		req.Body.Close()
		for len(f.bodies) <= idx {
			f.bodies = append(f.bodies, nil)
		}
		f.bodies[idx] = b
	}
	status := 500
	if idx < len(f.statuses) {
		status = f.statuses[idx]
	}
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func TestFollowSessionRateLimit(t *testing.T) {
	t.Parallel()

	log := logptest.NewTestingLogger(t, "")
	const retryAfter = "45"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errors":[{"message":"rate limit exceeded"}]}`))
	}))
	t.Cleanup(srv.Close)

	s := &falconHoseStream{
		processor: processor{
			ns:      "test",
			log:     log,
			metrics: newInputMetrics(monitoring.NewRegistry(), log),
		},
		status:      noopReporter{},
		discoverURL: srv.URL,
	}

	state := map[string]any{}
	_, err := s.followSession(context.Background(), srv.Client(), state)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var rle *rateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected *rateLimitError, got %T: %v", err, err)
	}
	if rle.wait != 45*time.Second {
		t.Errorf("rateLimitError.wait = %v, want %v", rle.wait, 45*time.Second)
	}
}
