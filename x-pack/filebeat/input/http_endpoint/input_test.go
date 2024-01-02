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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

var serverPoolTests = []struct {
	name    string
	method  string
	cfgs    []*httpEndpoint
	events  []target
	want    []mapstr.M
	wantErr error
}{
	{
		name: "single",
		cfgs: []*httpEndpoint{{
			addr: "127.0.0.1:9001",
			config: config{
				ResponseCode:  200,
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
				ResponseCode:  200,
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
				ResponseCode:  200,
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
		want: []mapstr.M{
			{"json": mapstr.M{"a": int64(1)}},
			{"json": mapstr.M{"b": int64(2)}},
			{"json": mapstr.M{"c": int64(3)}},
		},
	},
	{
		name: "distinct_ports",
		cfgs: []*httpEndpoint{
			{
				addr: "127.0.0.1:9001",
				config: config{
					ResponseCode:  200,
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
					ResponseCode:  200,
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
					ResponseCode:  200,
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
					ResponseCode:  200,
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
					ResponseCode:  200,
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
					ResponseCode:  200,
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
					ResponseCode:  200,
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
					ResponseCode:  200,
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
}

type target struct {
	url   string
	event string
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
			metrics := newInputMetrics("")
			defer metrics.Close()
			var wg sync.WaitGroup
			for _, cfg := range test.cfgs {
				cfg := cfg
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := servers.serve(ctx, cfg, &pub, metrics)
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
				if resp.StatusCode != http.StatusOK {
					t.Errorf("unexpected response status code: %s (%d)\nresp: %s",
						resp.Status, resp.StatusCode, body)
				}
			}
			cancel()
			wg.Wait()
			var got []mapstr.M
			for _, e := range pub.events {
				got = append(got, e.Fields)
			}
			if !cmp.Equal(got, test.want) {
				t.Errorf("unexpected result:\n--- got\n--- want\n%s", cmp.Diff(got, test.want))
			}

			// Try to re-register the same addresses.
			ctx, cancel = newCtx("server_pool_test", test.name)
			for _, cfg := range test.cfgs {
				cfg := cfg
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := servers.serve(ctx, cfg, &pub, metrics)
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
