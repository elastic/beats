// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

var serverPoolTests = []struct {
	name       string
	method     string
	cfgs       []*httpEndpoint
	events     []target
	want       []mapstr.M
	wantStatus int
	wantErr    error
}{
	{
		name: "single",
		cfgs: []*httpEndpoint{{
			addr: "127.0.0.1:9001",
			config: config{
				ResponseCode:  http.StatusOK,
				ResponseBody:  `{"message": "success"}`,
				ListenAddress: "127.0.0.1",
				ListenPort:    "9001",
				URL:           "/",
				Prefix:        "json",
				ContentType:   "application/json",
			},
		}},
		events: []target{
			{url: "http://127.0.0.1:9001/", event: `{"a":1}`},
			{url: "http://127.0.0.1:9001/", event: `{"b":2}`},
			{url: "http://127.0.0.1:9001/", event: `{"c":3}`},
		},
		wantStatus: http.StatusOK,
		want: []mapstr.M{
			{"json": mapstr.M{"a": int64(1)}},
			{"json": mapstr.M{"b": int64(2)}},
			{"json": mapstr.M{"c": int64(3)}},
		},
	},
	{
		name:   "put",
		method: http.MethodPut,
		cfgs: []*httpEndpoint{{
			addr: "127.0.0.1:9001",
			config: config{
				Method:        http.MethodPut,
				ResponseCode:  http.StatusOK,
				ResponseBody:  `{"message": "success"}`,
				ListenAddress: "127.0.0.1",
				ListenPort:    "9001",
				URL:           "/",
				Prefix:        "json",
				ContentType:   "application/json",
			},
		}},
		events: []target{
			{url: "http://127.0.0.1:9001/", event: `{"a":1}`},
			{url: "http://127.0.0.1:9001/", event: `{"b":2}`},
			{url: "http://127.0.0.1:9001/", event: `{"c":3}`},
		},
		wantStatus: http.StatusOK,
		want: []mapstr.M{
			{"json": mapstr.M{"a": int64(1)}},
			{"json": mapstr.M{"b": int64(2)}},
			{"json": mapstr.M{"c": int64(3)}},
		},
	},
	{
		name:   "patch",
		method: http.MethodPatch,
		cfgs: []*httpEndpoint{{
			addr: "127.0.0.1:9001",
			config: config{
				Method:        http.MethodPatch,
				ResponseCode:  http.StatusOK,
				ResponseBody:  `{"message": "success"}`,
				ListenAddress: "127.0.0.1",
				ListenPort:    "9001",
				URL:           "/",
				Prefix:        "json",
				ContentType:   "application/json",
			},
		}},
		events: []target{
			{url: "http://127.0.0.1:9001/", event: `{"a":1}`},
			{url: "http://127.0.0.1:9001/", event: `{"b":2}`},
			{url: "http://127.0.0.1:9001/", event: `{"c":3}`},
		},
		wantStatus: http.StatusOK,
		want: []mapstr.M{
			{"json": mapstr.M{"a": int64(1)}},
			{"json": mapstr.M{"b": int64(2)}},
			{"json": mapstr.M{"c": int64(3)}},
		},
	},
	{
		name:   "options_with_headers",
		method: http.MethodOptions,
		cfgs: []*httpEndpoint{{
			addr: "127.0.0.1:9001",
			config: config{
				ResponseCode:   http.StatusOK,
				ResponseBody:   `{"message": "success"}`,
				OptionsStatus:  http.StatusOK,
				OptionsHeaders: http.Header{"option-header": {"options-header-value"}},
				ListenAddress:  "127.0.0.1",
				ListenPort:     "9001",
				URL:            "/",
				Prefix:         "json",
				ContentType:    "application/json",
			},
		}},
		events: []target{
			{
				url: "http://127.0.0.1:9001/", wantHeader: http.Header{
					"Content-Length": {"0"},
					"Option-Header":  {"options-header-value"},
				},
			},
		},
		wantStatus: http.StatusOK,
	},
	{
		name:   "options_empty_headers",
		method: http.MethodOptions,
		cfgs: []*httpEndpoint{{
			addr: "127.0.0.1:9001",
			config: config{
				ResponseCode:   http.StatusOK,
				ResponseBody:   `{"message": "success"}`,
				OptionsStatus:  http.StatusOK,
				OptionsHeaders: http.Header{},
				ListenAddress:  "127.0.0.1",
				ListenPort:     "9001",
				URL:            "/",
				Prefix:         "json",
				ContentType:    "application/json",
			},
		}},
		events: []target{
			{
				url: "http://127.0.0.1:9001/", wantHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
		wantStatus: http.StatusOK,
	},
	{
		name:   "options_no_headers",
		method: http.MethodOptions,
		cfgs: []*httpEndpoint{{
			addr: "127.0.0.1:9001",
			config: config{
				ResponseCode:   http.StatusOK,
				ResponseBody:   `{"message": "success"}`,
				OptionsStatus:  http.StatusOK,
				OptionsHeaders: nil,
				ListenAddress:  "127.0.0.1",
				ListenPort:     "9001",
				URL:            "/",
				Prefix:         "json",
				ContentType:    "application/json",
			},
		}},
		events: []target{
			{url: "http://127.0.0.1:9001/", wantBody: `{"message":"OPTIONS requests are only allowed with options_headers set"}` + "\n"},
		},
		wantStatus: http.StatusBadRequest,
	},
	{
		name: "distinct_ports",
		cfgs: []*httpEndpoint{
			{
				addr: "127.0.0.1:9001",
				config: config{
					ResponseCode:  http.StatusOK,
					ResponseBody:  `{"message": "success"}`,
					ListenAddress: "127.0.0.1",
					ListenPort:    "9001",
					URL:           "/a/",
					Prefix:        "json",
					ContentType:   "application/json",
				},
			},
			{
				addr: "127.0.0.1:9002",
				config: config{
					ResponseCode:  http.StatusOK,
					ResponseBody:  `{"message": "success"}`,
					ListenAddress: "127.0.0.1",
					ListenPort:    "9002",
					URL:           "/b/",
					Prefix:        "json",
					ContentType:   "application/json",
				},
			},
		},
		events: []target{
			{url: "http://127.0.0.1:9001/a/", event: `{"a":1}`},
			{url: "http://127.0.0.1:9002/b/", event: `{"b":2}`},
			{url: "http://127.0.0.1:9001/a/", event: `{"c":3}`},
		},
		wantStatus: http.StatusOK,
		want: []mapstr.M{
			{"json": mapstr.M{"a": int64(1)}},
			{"json": mapstr.M{"b": int64(2)}},
			{"json": mapstr.M{"c": int64(3)}},
		},
	},
	{
		name: "shared_ports",
		cfgs: []*httpEndpoint{
			{
				addr: "127.0.0.1:9001",
				config: config{
					ResponseCode:  http.StatusOK,
					ResponseBody:  `{"message": "success"}`,
					ListenAddress: "127.0.0.1",
					ListenPort:    "9001",
					URL:           "/a/",
					Prefix:        "json",
					ContentType:   "application/json",
				},
			},
			{
				addr: "127.0.0.1:9001",
				config: config{
					ResponseCode:  http.StatusOK,
					ResponseBody:  `{"message": "success"}`,
					ListenAddress: "127.0.0.1",
					ListenPort:    "9001",
					URL:           "/b/",
					Prefix:        "json",
					ContentType:   "application/json",
				},
			},
		},
		events: []target{
			{url: "http://127.0.0.1:9001/a/", event: `{"a":1}`},
			{url: "http://127.0.0.1:9001/b/", event: `{"b":2}`},
			{url: "http://127.0.0.1:9001/a/", event: `{"c":3}`},
		},
		wantStatus: http.StatusOK,
		want: []mapstr.M{
			{"json": mapstr.M{"a": int64(1)}},
			{"json": mapstr.M{"b": int64(2)}},
			{"json": mapstr.M{"c": int64(3)}},
		},
	},
	{
		name: "inconsistent_tls_mixed_traffic",
		cfgs: []*httpEndpoint{
			{
				addr: "127.0.0.1:9001",
				config: config{
					ResponseCode:  http.StatusOK,
					ResponseBody:  `{"message": "success"}`,
					ListenAddress: "127.0.0.1",
					ListenPort:    "9001",
					URL:           "/a/",
					Prefix:        "json",
					ContentType:   "application/json",
				},
			},
			{
				addr: "127.0.0.1:9001",
				config: config{
					TLS:           &tlscommon.ServerConfig{},
					ResponseCode:  http.StatusOK,
					ResponseBody:  `{"message": "success"}`,
					ListenAddress: "127.0.0.1",
					ListenPort:    "9001",
					URL:           "/b/",
					Prefix:        "json",
					ContentType:   "application/json",
				},
			},
		},
		wantErr: invalidTLSStateErr{addr: "127.0.0.1:9001", reason: "mixed TLS and unencrypted"},
	},
	{
		name: "inconsistent_tls_config",
		cfgs: []*httpEndpoint{
			{
				addr: "127.0.0.1:9001",
				config: config{
					TLS: &tlscommon.ServerConfig{
						VerificationMode: tlscommon.VerifyStrict,
					},
					ResponseCode:  http.StatusOK,
					ResponseBody:  `{"message": "success"}`,
					ListenAddress: "127.0.0.1",
					ListenPort:    "9001",
					URL:           "/a/",
					Prefix:        "json",
					ContentType:   "application/json",
				},
			},
			{
				addr: "127.0.0.1:9001",
				config: config{
					TLS: &tlscommon.ServerConfig{
						VerificationMode: tlscommon.VerifyNone,
					},
					ResponseCode:  http.StatusOK,
					ResponseBody:  `{"message": "success"}`,
					ListenAddress: "127.0.0.1",
					ListenPort:    "9001",
					URL:           "/b/",
					Prefix:        "json",
					ContentType:   "application/json",
				},
			},
		},
		wantErr: invalidTLSStateErr{addr: "127.0.0.1:9001", reason: "configuration options do not agree"},
	},
	{
		// Test that sequential requests properly release in-flight bytes after ACK.
		// With sequential requests and immediate ACK, in-flight bytes return to 0
		// between requests, so all requests succeed. This verifies byte tracking
		// correctly adds bytes during read and releases them after ACK.
		// (See TestConcurrentExceedMaxInFlight for concurrent rejection testing.)
		name:   "sequential_in_flight_tracking",
		method: http.MethodPost,
		cfgs: []*httpEndpoint{{
			addr: "127.0.0.1:9001",
			config: config{
				Method:            http.MethodPost,
				ResponseCode:      http.StatusOK,
				ResponseBody:      `{"message": "success"}`,
				ListenAddress:     "127.0.0.1",
				ListenPort:        "9001",
				URL:               "/",
				Prefix:            "json",
				MaxInFlight:       100,
				HighWaterInFlight: 50,
				LowWaterInFlight:  25,
				RetryAfter:        10,
				ContentType:       "application/json",
			},
		}},
		events: []target{
			// Sequential requests succeed because in-flight returns to 0 between requests
			{url: "http://127.0.0.1:9001/?wait_for_completion_timeout=1s", event: `{"a":1}`, wantBody: `{"message": "success"}`},
			{url: "http://127.0.0.1:9001/?wait_for_completion_timeout=1s", event: `{"b":2}`, wantBody: `{"message": "success"}`},
			{url: "http://127.0.0.1:9001/?wait_for_completion_timeout=1s", event: `{"c":3}`, wantBody: `{"message": "success"}`},
		},
		wantStatus: http.StatusOK,
		want: []mapstr.M{
			{"json": mapstr.M{"a": int64(1)}},
			{"json": mapstr.M{"b": int64(2)}},
			{"json": mapstr.M{"c": int64(3)}},
		},
	},
	{
		name:   "not_exceed_max_in_flight",
		method: http.MethodPost,
		cfgs: []*httpEndpoint{{
			addr: "127.0.0.1:9001",
			config: config{
				Method:        http.MethodPost,
				ResponseCode:  http.StatusOK,
				ResponseBody:  `{"message": "success"}`,
				ListenAddress: "127.0.0.1",
				ListenPort:    "9001",
				URL:           "/",
				Prefix:        "json",
				MaxInFlight:   20,
				RetryAfter:    10,
				ContentType:   "application/json",
			},
		}},
		events: []target{
			{url: "http://127.0.0.1:9001/?wait_for_completion_timeout=1s", event: `{"a":1}`, wantBody: `{"message": "success"}`, wantHeader: http.Header{"Retry-After": nil}},
			{url: "http://127.0.0.1:9001/?wait_for_completion_timeout=1s", event: `{"b":2}`, wantBody: `{"message": "success"}`, wantHeader: http.Header{"Retry-After": nil}},
			{url: "http://127.0.0.1:9001/?wait_for_completion_timeout=1s", event: `{"c":3}`, wantBody: `{"message": "success"}`, wantHeader: http.Header{"Retry-After": nil}},
		},
		wantStatus: http.StatusOK,
		want: []mapstr.M{
			{"json": mapstr.M{"a": int64(1)}},
			{"json": mapstr.M{"b": int64(2)}},
			{"json": mapstr.M{"c": int64(3)}},
		},
	},
}

type target struct {
	url        string
	event      string
	wantBody   string
	wantHeader http.Header
}

// isWantedHeader returns whether got includes the wanted header and that
// the values match. A nil value for a header in the receiver matches absence
// of that header in the got parameter.
func (t target) isWantedHeader(got http.Header) bool {
	for h, v := range t.wantHeader {
		if v == nil {
			if _, ok := got[h]; ok {
				return false
			}
			continue
		}
		if !slices.Equal(got[h], v) {
			return false
		}
	}
	return true
}

func TestServerPool(t *testing.T) {
	for _, test := range serverPoolTests {
		t.Run(test.name, func(t *testing.T) {
			servers := pool{servers: make(map[string]*server)}

			var (
				pub   publisher
				fails = make(chan error, 1)
			)
			ctx, cancel := newCtx("server_pool_test", test.name)
			metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())
			var wg sync.WaitGroup
			for _, cfg := range test.cfgs {
				cfg := cfg
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := servers.serve(ctx, cfg, pub.Publish, metrics)
					if err != http.ErrServerClosed {
						select {
						case fails <- err:
						default:
						}
					}
				}()
			}
			if test.wantErr == nil {
				addrCount := make(map[string]int)
				for _, cfg := range test.cfgs {
					addrCount[cfg.addr]++
				}
				for addr := range addrCount {
					waitForServer(t, addr, 5*time.Second)
				}
				for addr, n := range addrCount {
					servers.waitForHandlers(t, addr, n, 5*time.Second)
				}
			}

			if test.wantErr != nil {
				select {
				case err := <-fails:
					if !errors.Is(err, test.wantErr) {
						t.Errorf("unexpected error calling serve: got=%#q, want=%#q", err, test.wantErr)
					}
				case <-time.After(5 * time.Second):
					t.Errorf("expected error calling serve")
				}
			} else {
				select {
				case err := <-fails:
					t.Errorf("unexpected error calling serve: %#q", err)
				default:
				}
			}
			for i, e := range test.events {
				resp, err := doRequest(test.method, e.url, "application/json", strings.NewReader(e.event))
				if err != nil {
					t.Fatalf("failed to post event #%d: %v", i, err)
				}
				body := dump(resp.Body)
				if resp.StatusCode != test.wantStatus {
					t.Errorf("unexpected response status code: %s (%d), want: %d\nresp: %s",
						resp.Status, resp.StatusCode, test.wantStatus, body)
				}
				if len(e.wantBody) != 0 && string(body) != e.wantBody {
					t.Errorf("unexpected response body:\ngot: %s\nwant:%s", body, e.wantBody)
				}
				if !e.isWantedHeader(resp.Header) {
					t.Errorf("unexpected header:\n--- want\n+++ got\n%s", cmp.Diff(e.wantHeader, resp.Header))
				}
			}
			cancel()
			wg.Wait()
			var got []mapstr.M
			for _, e := range pub.events {
				got = append(got, e.Fields)
			}
			if !cmp.Equal(test.want, got) {
				t.Errorf("unexpected result:\n--- want\n+++ got\n%s", cmp.Diff(test.want, got))
			}

			// Try to re-register the same addresses.
			ctx, cancel = newCtx("server_pool_test", test.name)
			for _, cfg := range test.cfgs {
				cfg := cfg
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := servers.serve(ctx, cfg, pub.Publish, metrics)
					if err != nil && err != http.ErrServerClosed && test.wantErr == nil {
						t.Errorf("failed to re-register %v: %v", cfg.addr, err)
					}
				}()
			}
			cancel()
			wg.Wait()
		})
	}
}

// TestConcurrentExceedMaxInFlight tests that concurrent requests are properly
// rejected when in-flight bytes exceed the high water mark. This requires
// holding bytes until ACK, which means we need a publisher that delays ACK.
func TestConcurrentExceedMaxInFlight(t *testing.T) {
	servers := pool{servers: make(map[string]*server)}

	// delayedACKPublisher delays ACK to simulate slow Elasticsearch processing.
	// This causes bytes to be held in-flight while waiting for ACK.
	var (
		mu         sync.Mutex
		events     []mapstr.M
		ackDelayed = make(chan struct{})
	)
	delayedPublish := func(e beat.Event) {
		mu.Lock()
		events = append(events, e.Fields)
		acker := e.Private.(*batchACKTracker)
		mu.Unlock()

		// First event waits for signal before ACKing.
		// This holds bytes in-flight during the wait.
		<-ackDelayed
		acker.ACK()
	}

	cfg := &httpEndpoint{
		addr: "127.0.0.1:9010",
		config: config{
			Method:            http.MethodPost,
			ResponseCode:      http.StatusOK,
			ResponseBody:      `{"message": "success"}`,
			ListenAddress:     "127.0.0.1",
			ListenPort:        "9010",
			URL:               "/",
			Prefix:            "json",
			MaxInFlight:       100,
			HighWaterInFlight: 10, // Low threshold so small JSON exceeds it.
			LowWaterInFlight:  5,
			ContentType:       "application/json",
		},
	}

	ctx, cancel := newCtx("concurrent_exceed_test", "test")
	defer cancel()

	metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := servers.serve(ctx, cfg, delayedPublish, metrics)
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected serve error: %v", err)
		}
	}()
	waitForServer(t, "127.0.0.1:9010", 5*time.Second)

	var reqWg sync.WaitGroup

	// Send first request with wait_for_completion_timeout.
	// This will hold bytes in-flight until ACK (which we delay).
	var firstStatus int
	reqWg.Add(1)
	go func() {
		defer reqWg.Done()
		resp, err := doRequest(http.MethodPost,
			"http://127.0.0.1:9010/?wait_for_completion_timeout=5s",
			"application/json",
			strings.NewReader(`{"first":"request with enough bytes to exceed high water"}`))
		if err != nil {
			t.Errorf("first request failed: %v", err)
			return
		}
		firstStatus = resp.StatusCode
		resp.Body.Close()
	}()

	// Wait a bit for first request to be processing (reading body, waiting for ACK).
	time.Sleep(200 * time.Millisecond)

	// Send second request while first is holding bytes.
	var secondStatus int
	reqWg.Add(1)
	go func() {
		defer reqWg.Done()
		resp, err := doRequest(http.MethodPost,
			"http://127.0.0.1:9010/?wait_for_completion_timeout=5s",
			"application/json",
			strings.NewReader(`{"second":"request"}`))
		if err != nil {
			t.Errorf("second request failed: %v", err)
			return
		}
		secondStatus = resp.StatusCode
		resp.Body.Close()
	}()

	// Wait for second request to complete (should be rejected quickly).
	time.Sleep(200 * time.Millisecond)

	// Now release the first request's ACK.
	close(ackDelayed)

	// Wait for both requests to complete.
	reqWg.Wait()

	// First request should succeed (it got in before high water).
	if firstStatus != http.StatusOK {
		t.Errorf("first request: got status %d, want %d", firstStatus, http.StatusOK)
	}

	// Second request should be rejected with 503 (high water exceeded).
	if secondStatus != http.StatusServiceUnavailable {
		t.Errorf("second request: got status %d, want %d", secondStatus, http.StatusServiceUnavailable)
	}

	cancel()
	wg.Wait()
}

func TestNewHTTPEndpoint(t *testing.T) {
	cfg := config{
		ListenAddress: "0:0:0:0:0:0:0:1",
		ListenPort:    "9200",
		ResponseBody:  "{}",
		Method:        http.MethodPost,
	}
	h, err := newHTTPEndpoint(cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	require.Equal(t, "[0:0:0:0:0:0:0:1]:9200", h.addr)
}

func doRequest(method, url, contentType string, body io.Reader) (*http.Response, error) {
	if method == "" {
		method = http.MethodPost
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return http.DefaultClient.Do(req)
}

// Is is included to simplify testing, but is not exposed to avoid unwanted error
// matching outside tests.
func (e invalidTLSStateErr) Is(err error) bool {
	if err, ok := err.(invalidTLSStateErr); ok { //nolint:errorlint // "An Is method should only shallowly compare err and the target and not call Unwrap on either."
		// However for our purposes here, we will abuse
		// the Is convention and also consider the addr
		// and reason fields.
		return e.addr == err.addr && e.reason == err.reason
	}
	return false
}

func newCtx(log, id string) (_ v2.Context, cancel func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger(log),
		ID:          id,
		Cancelation: ctx,
	}, cancel
}

func dump(r io.ReadCloser) []byte {
	defer r.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.Bytes()
}

func TestMux(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	t.Run("exact_match", func(t *testing.T) {
		m := &mux{exact: make(map[string]http.Handler)}
		m.add("/foo", ok)
		if m.match("/foo") == nil {
			t.Error("expected handler for /foo")
		}
		if m.match("/foo/bar") != nil {
			t.Error("unexpected handler for /foo/bar")
		}
	})
	t.Run("prefix_match", func(t *testing.T) {
		m := &mux{exact: make(map[string]http.Handler)}
		m.add("/a/", ok)
		if m.match("/a/") == nil {
			t.Error("expected handler for /a/")
		}
		if m.match("/a/b") == nil {
			t.Error("expected handler for /a/b")
		}
		if m.match("/b/") != nil {
			t.Error("unexpected handler for /b/")
		}
	})
	t.Run("longest_prefix_wins", func(t *testing.T) {
		m := &mux{exact: make(map[string]http.Handler)}
		short := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		long := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		})
		m.add("/a/", short)
		m.add("/a/b/", long)
		rec := httptest.NewRecorder()
		m.ServeHTTP(rec, httptest.NewRequest("GET", "/a/b/c", nil))
		if rec.Code != http.StatusAccepted {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusAccepted)
		}
	})
	t.Run("exact_beats_prefix", func(t *testing.T) {
		m := &mux{exact: make(map[string]http.Handler)}
		prefix := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		exact := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		})
		m.add("/a/", prefix)
		m.add("/a/b", exact)
		rec := httptest.NewRecorder()
		m.ServeHTTP(rec, httptest.NewRequest("GET", "/a/b", nil))
		if rec.Code != http.StatusAccepted {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusAccepted)
		}
	})
	t.Run("remove_exact", func(t *testing.T) {
		m := &mux{exact: make(map[string]http.Handler)}
		m.add("/foo", ok)
		m.add("/bar", ok)
		empty := m.remove("/foo")
		if empty {
			t.Error("mux should not be empty")
		}
		if m.match("/foo") != nil {
			t.Error("handler should be removed")
		}
		empty = m.remove("/bar")
		if !empty {
			t.Error("mux should be empty")
		}
	})
	t.Run("remove_prefix", func(t *testing.T) {
		m := &mux{exact: make(map[string]http.Handler)}
		m.add("/a/", ok)
		m.add("/b/", ok)
		empty := m.remove("/a/")
		if empty {
			t.Error("mux should not be empty")
		}
		if m.match("/a/x") != nil {
			t.Error("handler should be removed")
		}
		empty = m.remove("/b/")
		if !empty {
			t.Error("mux should be empty")
		}
	})
	t.Run("not_found", func(t *testing.T) {
		m := &mux{exact: make(map[string]http.Handler)}
		m.add("/foo", ok)
		rec := httptest.NewRecorder()
		m.ServeHTTP(rec, httptest.NewRequest("GET", "/bar", nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
	t.Run("path_clean_conformance", func(t *testing.T) {
		patterns := []string{"/a/b", "/a/", "/x/y/z/"}

		sm := http.NewServeMux()
		m := &mux{exact: make(map[string]http.Handler)}
		for _, p := range patterns {
			sm.Handle(p, ok)
			m.add(p, ok)
		}

		paths := []string{
			"/a/b",
			"/a//b",
			"/a/b/",
			"/a/",
			"/a/b/../",
			"/a/./b",
			"/x///y/z/",
			"/x/y/z/../z/",
			"/x/y/../y/z/",
			"/clean",
		}
		for _, p := range paths {
			req := httptest.NewRequest("GET", p, nil)

			smRec := httptest.NewRecorder()
			sm.ServeHTTP(smRec, req)

			mRec := httptest.NewRecorder()
			m.ServeHTTP(mRec, req)

			if mRec.Code != smRec.Code {
				t.Errorf("path %q: status mux=%d, http.ServeMux=%d", p, mRec.Code, smRec.Code)
			}
			if mRec.Header().Get("Location") != smRec.Header().Get("Location") {
				t.Errorf("path %q: Location mux=%q, http.ServeMux=%q",
					p, mRec.Header().Get("Location"), smRec.Header().Get("Location"))
			}
		}
	})
}

func TestJoinerDeregisterKeepsServer(t *testing.T) {
	servers := pool{servers: make(map[string]*server)}
	var pub publisher
	metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	ctxA, cancelA := newCtx("test", "input-a")
	ctxB, cancelB := newCtx("test", "input-b")

	cfgA := &httpEndpoint{
		addr: "127.0.0.1:9021",
		config: config{
			ResponseCode:  http.StatusOK,
			ResponseBody:  `{"message": "success"}`,
			ListenAddress: "127.0.0.1",
			ListenPort:    "9021",
			URL:           "/a/",
			Prefix:        "json",
			ContentType:   "application/json",
		},
	}
	cfgB := &httpEndpoint{
		addr: "127.0.0.1:9021",
		config: config{
			ResponseCode:  http.StatusOK,
			ResponseBody:  `{"message": "success"}`,
			ListenAddress: "127.0.0.1",
			ListenPort:    "9021",
			URL:           "/b/",
			Prefix:        "json",
			ContentType:   "application/json",
		},
	}

	var wg sync.WaitGroup
	errA := make(chan error, 1)
	errB := make(chan error, 1)

	wg.Add(2)
	go func() {
		defer wg.Done()
		errA <- servers.serve(ctxA, cfgA, pub.Publish, metrics)
	}()
	go func() {
		defer wg.Done()
		errB <- servers.serve(ctxB, cfgB, pub.Publish, metrics)
	}()
	waitForServer(t, "127.0.0.1:9021", 5*time.Second)
	servers.waitForHandlers(t, "127.0.0.1:9021", 2, 5*time.Second)

	// Stop B (joiner). A's server should stay alive.
	cancelB()
	select {
	case err := <-errB:
		if err != nil {
			t.Errorf("joiner returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("joiner did not return in time")
	}

	// A's endpoint should still work.
	resp, err := doRequest("", "http://127.0.0.1:9021/a/", "application/json", strings.NewReader(`{"x":1}`))
	if err != nil {
		t.Fatalf("request to remaining endpoint failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// B's endpoint should be gone (404).
	resp, err = doRequest("", "http://127.0.0.1:9021/b/", "application/json", strings.NewReader(`{"x":2}`))
	if err != nil {
		t.Fatalf("request to removed endpoint failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want %d for removed endpoint", resp.StatusCode, http.StatusNotFound)
	}

	// Stop A (last input). Server should shut down.
	cancelA()
	select {
	case err := <-errA:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("creator returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("creator did not return in time")
	}
	wg.Wait()
}

func TestCreatorDeregisterKeepsServer(t *testing.T) {
	servers := pool{servers: make(map[string]*server)}
	var pub publisher
	metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	ctxA, cancelA := newCtx("test", "input-a")
	ctxB, cancelB := newCtx("test", "input-b")

	cfgA := &httpEndpoint{
		addr: "127.0.0.1:9022",
		config: config{
			ResponseCode:  http.StatusOK,
			ResponseBody:  `{"message": "success"}`,
			ListenAddress: "127.0.0.1",
			ListenPort:    "9022",
			URL:           "/a/",
			Prefix:        "json",
			ContentType:   "application/json",
		},
	}
	cfgB := &httpEndpoint{
		addr: "127.0.0.1:9022",
		config: config{
			ResponseCode:  http.StatusOK,
			ResponseBody:  `{"message": "success"}`,
			ListenAddress: "127.0.0.1",
			ListenPort:    "9022",
			URL:           "/b/",
			Prefix:        "json",
			ContentType:   "application/json",
		},
	}

	var wg sync.WaitGroup
	errA := make(chan error, 1)
	errB := make(chan error, 1)

	wg.Add(2)
	go func() {
		defer wg.Done()
		errA <- servers.serve(ctxA, cfgA, pub.Publish, metrics)
	}()
	go func() {
		defer wg.Done()
		errB <- servers.serve(ctxB, cfgB, pub.Publish, metrics)
	}()
	waitForServer(t, "127.0.0.1:9022", 5*time.Second)
	servers.waitForHandlers(t, "127.0.0.1:9022", 2, 5*time.Second)

	// Stop A (creator). B's server should stay alive.
	cancelA()
	select {
	case err := <-errA:
		if err != nil {
			t.Errorf("creator returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("creator did not return in time")
	}

	// B's endpoint should still work.
	resp, err := doRequest("", "http://127.0.0.1:9022/b/", "application/json", strings.NewReader(`{"x":1}`))
	if err != nil {
		t.Fatalf("request to remaining endpoint failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// A's endpoint should be gone (404).
	resp, err = doRequest("", "http://127.0.0.1:9022/a/", "application/json", strings.NewReader(`{"x":2}`))
	if err != nil {
		t.Fatalf("request to removed endpoint failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want %d for removed endpoint", resp.StatusCode, http.StatusNotFound)
	}

	// Stop B (last input). Server should shut down.
	cancelB()
	select {
	case err := <-errB:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("last input returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("last input did not return in time")
	}
	wg.Wait()
}

func TestPatternReregistration(t *testing.T) {
	servers := pool{servers: make(map[string]*server)}
	var pub publisher
	metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	cfg := &httpEndpoint{
		addr: "127.0.0.1:9023",
		config: config{
			ResponseCode:  http.StatusOK,
			ResponseBody:  `{"message": "success"}`,
			ListenAddress: "127.0.0.1",
			ListenPort:    "9023",
			URL:           "/a/",
			Prefix:        "json",
			ContentType:   "application/json",
		},
	}

	// First registration.
	ctx1, cancel1 := newCtx("test", "input-1")
	var wg sync.WaitGroup
	err1 := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err1 <- servers.serve(ctx1, cfg, pub.Publish, metrics)
	}()
	waitForServer(t, "127.0.0.1:9023", 5*time.Second)

	resp, err := doRequest("", "http://127.0.0.1:9023/a/", "application/json", strings.NewReader(`{"x":1}`))
	if err != nil {
		t.Fatalf("first registration request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Deregister (also shuts down server since it's the only input).
	cancel1()
	select {
	case err := <-err1:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("first registration returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first registration did not return in time")
	}
	wg.Wait()

	// Re-register same pattern on a new server.
	ctx2, cancel2 := newCtx("test", "input-2")
	err2 := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err2 <- servers.serve(ctx2, cfg, pub.Publish, metrics)
	}()
	waitForServer(t, "127.0.0.1:9023", 5*time.Second)

	resp, err = doRequest("", "http://127.0.0.1:9023/a/", "application/json", strings.NewReader(`{"x":2}`))
	if err != nil {
		t.Fatalf("re-registration request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d after re-registration", resp.StatusCode, http.StatusOK)
	}

	cancel2()
	select {
	case err := <-err2:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("re-registration returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("re-registration did not return in time")
	}
	wg.Wait()
}

func TestSimultaneousShutdown(t *testing.T) {
	servers := pool{servers: make(map[string]*server)}
	var pub publisher
	metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	ctxA, cancelA := newCtx("test", "input-a")
	ctxB, cancelB := newCtx("test", "input-b")

	cfgA := &httpEndpoint{
		addr: "127.0.0.1:9024",
		config: config{
			ResponseCode:  http.StatusOK,
			ResponseBody:  `{"message": "success"}`,
			ListenAddress: "127.0.0.1",
			ListenPort:    "9024",
			URL:           "/a/",
			Prefix:        "json",
			ContentType:   "application/json",
		},
	}
	cfgB := &httpEndpoint{
		addr: "127.0.0.1:9024",
		config: config{
			ResponseCode:  http.StatusOK,
			ResponseBody:  `{"message": "success"}`,
			ListenAddress: "127.0.0.1",
			ListenPort:    "9024",
			URL:           "/b/",
			Prefix:        "json",
			ContentType:   "application/json",
		},
	}

	var wg sync.WaitGroup
	errA := make(chan error, 1)
	errB := make(chan error, 1)

	wg.Add(2)
	go func() {
		defer wg.Done()
		errA <- servers.serve(ctxA, cfgA, pub.Publish, metrics)
	}()
	go func() {
		defer wg.Done()
		errB <- servers.serve(ctxB, cfgB, pub.Publish, metrics)
	}()
	waitForServer(t, "127.0.0.1:9024", 5*time.Second)
	servers.waitForHandlers(t, "127.0.0.1:9024", 2, 5*time.Second)

	// Cancel both at once.
	cancelA()
	cancelB()

	for _, ch := range []chan error{errA, errB} {
		select {
		case err := <-ch:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("input did not return in time")
		}
	}
	wg.Wait()
}

// waitForServer polls addr until a TCP connection succeeds or the
// timeout expires.
func waitForServer(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		c, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			c.Close()
			return
		}
		select {
		case <-deadline:
			t.Fatalf("server %s not ready after %s", addr, timeout)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// waitForHandlers polls until at least n handlers are registered for
// addr. Use after waitForServer in multi-handler tests to ensure all
// joiner registrations have completed before sending requests.
func (p *pool) waitForHandlers(t *testing.T, addr string, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		p.mu.Lock()
		s, ok := p.servers[addr]
		count := 0
		if ok {
			count = len(s.idOf)
		}
		p.mu.Unlock()
		if count >= n {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("server %s: wanted %d handlers, got %d after %s", addr, n, count, timeout)
		case <-time.After(10 * time.Millisecond):
		}
	}
}
