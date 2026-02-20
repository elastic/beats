// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
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
			time.Sleep(time.Second)

			select {
			case err := <-fails:
				if test.wantErr == nil {
					t.Errorf("unexpected error calling serve: %#q", err)
				} else if !errors.Is(err, test.wantErr) {
					t.Errorf("unexpected error calling serve: got=%#q, want=%#q", err, test.wantErr)
				}
			default:
				if test.wantErr != nil {
					t.Errorf("expected error calling serve")
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
	time.Sleep(500 * time.Millisecond) // Wait for server to start.

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
