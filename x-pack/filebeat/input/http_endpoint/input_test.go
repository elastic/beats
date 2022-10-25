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
		wantErr: errors.New("inconsistent TLS usage on 127.0.0.1:9001: mixed TLS and unencrypted"),
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
		wantErr: errors.New(`inconsistent TLS configuration on 127.0.0.1:9001: ` +
			`configuration options do not agree: ` +
			`old={"ca_sha256":[],"certificate":"","certificate_authorities":[],"cipher_suites":[],"client_authentication":0,"curve_types":[],"key":"","key_passphrase":"","supported_protocols":[],"verification_mode":1} ` +
			`new={"ca_sha256":[],"certificate":"","certificate_authorities":[],"cipher_suites":[],"client_authentication":0,"curve_types":[],"key":"","key_passphrase":"","supported_protocols":[],"verification_mode":3}`),
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
			var wg sync.WaitGroup
			for _, cfg := range test.cfgs {
				cfg := cfg
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := servers.serve(ctx, cfg, &pub)
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
				} else if test.wantErr.Error() != err.Error() {
					t.Errorf("unexpected error calling serve: got=%#q, want=%#q", err, test.wantErr)
				}
			default:
				if test.wantErr != nil {
					t.Errorf("expected error calling serve")
				}
			}
			for i, e := range test.events {
				resp, err := http.Post(e.url, "application/json", strings.NewReader(e.event))
				if err != nil {
					t.Fatalf("failed to post event #%d: %v", i, err)
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("unexpected response status code: %s (%d)\nresp: %s",
						resp.Status, resp.StatusCode, dump(resp.Body))
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
		})
	}
}

func newCtx(log, id string) (_ v2.Context, cancel func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger(log),
		ID:          id,
		Cancelation: ctx,
	}, cancel
}

func dump(r io.ReadCloser) string {
	defer r.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
