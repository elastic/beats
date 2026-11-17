// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/paths"
)

func TestCheckRedirect(t *testing.T) {
	headers := http.Header{
		"Authorization":       {"Bearer secret"},
		"Proxy-Authorization": {"Basic secret"},
		"Cookie":              {"session=secret"},
		"X-Custom":            {"value"},
		"X-Banned":            {"value"},
	}
	tests := []struct {
		name      string
		max       int
		forward   bool
		sensitive []string // nil means defaultSensitiveHeaders
		from      string
		to        string
		wantErr   bool
		want      []string // headers expected on the redirected request
	}{
		{name: "max_reached", max: 1, forward: true, from: "https://a.example/", to: "https://a.example/next", wantErr: true},
		{name: "no_forward", max: 10, from: "https://a.example/", to: "https://a.example/next"},
		{name: "same_origin", max: 10, forward: true, from: "https://a.example/", to: "https://a.example/next", want: []string{"Authorization", "Proxy-Authorization", "Cookie", "X-Custom"}},
		{name: "cross_origin", max: 10, forward: true, from: "https://a.example/", to: "https://b.example/next", want: []string{"X-Custom"}},
		{name: "other_port", max: 10, forward: true, from: "https://a.example/", to: "https://a.example:8443/next", want: []string{"X-Custom"}},
		{name: "https_to_http", max: 10, forward: true, from: "https://a.example/", to: "http://a.example/next", want: []string{"X-Custom"}},
		{name: "cross_origin_no_sensitive", max: 10, forward: true, sensitive: []string{}, from: "https://a.example/", to: "https://b.example/next", want: []string{"Authorization", "Proxy-Authorization", "Cookie", "X-Custom"}},
		{name: "cross_origin_custom_sensitive", max: 10, forward: true, sensitive: []string{"X-Custom"}, from: "https://a.example/", to: "https://b.example/next", want: []string{"Authorization", "Proxy-Authorization", "Cookie"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prev := httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.from, nil)
			prev.Header = headers.Clone()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.to, nil)
			req.Header = http.Header{}

			err := checkRedirect(test.max, test.forward, sensitiveHeaders(test.sensitive), []string{"X-Banned"})(req, []*http.Request{prev})
			if test.wantErr {
				if err == nil {
					t.Error("checkRedirect() = nil; want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("checkRedirect() = %v; want nil", err)
			}
			if len(req.Header) != len(test.want) {
				t.Errorf("redirected headers = %v; want only %q", req.Header, test.want)
			}
			for _, k := range test.want {
				if req.Header.Get(k) != headers.Get(k) {
					t.Errorf("redirected %s = %q; want %q", k, req.Header.Get(k), headers.Get(k))
				}
			}
		})
	}
}

// TestRequestConfigClientRedirect follows a cross-origin redirect with a
// client built from a RequestConfig that forwards headers, and checks what
// the second server receives.
func TestRequestConfigClientRedirect(t *testing.T) {
	got := make(chan http.Header, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.Header.Clone()
	}))
	defer target.Close()
	// Use a different host name for the same address so the redirect
	// is cross-origin.
	targetURL := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetURL, http.StatusFound)
	}))
	defer origin.Close()

	for _, test := range []struct {
		name              string
		sensitive         []string
		wantAuthorization string
		wantCustom        string
	}{
		{name: "default_sensitive_headers", wantAuthorization: "", wantCustom: "value"},
		{name: "no_sensitive_headers", sensitive: []string{}, wantAuthorization: "Bearer secret", wantCustom: "value"},
		{name: "custom_sensitive_headers", sensitive: []string{"X-Custom"}, wantAuthorization: "Bearer secret", wantCustom: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := DefaultRequestConfig(5 * time.Second)
			cfg.RedirectForwardHeaders = true
			cfg.RedirectSensitiveHeaders = test.sensitive
			cli, err := cfg.Client(logptest.NewTestingLogger(t, ""))
			if err != nil {
				t.Fatalf("Client() error: %v", err)
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, origin.URL, nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			req.Header.Set("Authorization", "Bearer secret")
			req.Header.Set("X-Custom", "value")
			resp, err := cli.Do(req)
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			resp.Body.Close()

			h := <-got
			if v := h.Get("Authorization"); v != test.wantAuthorization {
				t.Errorf("cross-origin redirect sent Authorization %q; want %q", v, test.wantAuthorization)
			}
			if v := h.Get("X-Custom"); v != test.wantCustom {
				t.Errorf("cross-origin redirect sent X-Custom %q; want %q", v, test.wantCustom)
			}
		})
	}
}

// TestRequestConfigClientMaxRedirects checks that the client stops at the
// configured number of redirects.
func TestRequestConfigClientMaxRedirects(t *testing.T) {
	var hits atomic.Int64
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, srv.URL+"/again", http.StatusFound)
	}))
	defer srv.Close()

	cfg := DefaultRequestConfig(5 * time.Second)
	cfg.RedirectMaxRedirects = 2
	cli, err := cfg.Client(logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatalf("Client() error: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := cli.Do(req)
	if err == nil {
		resp.Body.Close()
		t.Fatal("Do followed an endless redirect loop without error")
	}
	if !strings.Contains(err.Error(), "stopped after 2 redirects") {
		t.Errorf("Do error = %v; want it to report stopping after 2 redirects", err)
	}
	// As with Go's default policy, the limit counts the original
	// request, so two requests are sent.
	if n := hits.Load(); n != 2 {
		t.Errorf("server saw %d requests; want 2", n)
	}
}

// TestRequestConfigClientKeepAlive checks that keep-alives stay enabled by
// default and that keep_alive.disable turns them off.
func TestRequestConfigClientKeepAlive(t *testing.T) {
	closed := make(chan bool, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		closed <- r.Close
	}))
	defer srv.Close()

	disable := true
	for _, test := range []struct {
		name      string
		keepAlive KeepAlive
		wantClose bool
	}{
		{name: "default", wantClose: false},
		{name: "disabled", keepAlive: KeepAlive{Disable: &disable}, wantClose: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := DefaultRequestConfig(5 * time.Second)
			cfg.KeepAlive = test.keepAlive
			cli, err := cfg.Client(logptest.NewTestingLogger(t, ""))
			if err != nil {
				t.Fatalf("Client() error: %v", err)
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			resp, err := cli.Do(req)
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			resp.Body.Close()
			if got := <-closed; got != test.wantClose {
				t.Errorf("request asked to close the connection %t; want %t", got, test.wantClose)
			}
		})
	}
}

func TestRequestConfigRetryWarning(t *testing.T) {
	for _, test := range []struct {
		name  string
		retry *config.C
		want  int
	}{
		{name: "unset", want: 0},
		{name: "set", retry: config.MustNewConfigFrom(map[string]any{"max_attempts": 2}), want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := DefaultRequestConfig(time.Second)
			cfg.Retry = test.retry
			log, logs := logptest.NewTestingLoggerWithObserver(t, "")
			if _, err := cfg.Client(log); err != nil {
				t.Fatalf("Client() error: %v", err)
			}
			if n := logs.FilterMessageSnippet("request.retry options are not used").Len(); n != test.want {
				t.Errorf("got %d warnings; want %d", n, test.want)
			}
		})
	}
}

func TestRequestConfigUnpack(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{name: "negative_max_redirects", yaml: "redirect.max_redirects: -1", wantErr: "redirect.max_redirects must not be negative"},
		{name: "negative_max_idle_connections", yaml: "keep_alive.max_idle_connections: -1", wantErr: "max_idle_connections must not be negative"},
		{name: "negative_max_idle_connections_per_host", yaml: "keep_alive.max_idle_connections_per_host: -1", wantErr: "max_idle_connections_per_host must not be negative"},
		{name: "negative_idle_connection_timeout", yaml: "keep_alive.idle_connection_timeout: -1s", wantErr: "idle_connection_timeout must not be negative"},
		{name: "negative_limits_with_keep_alive_disabled", yaml: "keep_alive: {disable: true, max_idle_connections: -1, idle_connection_timeout: -1s}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := config.NewConfigWithYAML([]byte(test.yaml), "")
			if err != nil {
				t.Fatalf("parsing YAML: %v", err)
			}
			rc := DefaultRequestConfig(time.Second)
			err = cfg.Unpack(&rc)
			switch {
			case test.wantErr == "" && err != nil:
				t.Errorf("Unpack() = %v; want nil", err)
			case test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)):
				t.Errorf("Unpack() = %v; want error containing %q", err, test.wantErr)
			}
		})
	}
}

func TestSensitiveHeadersConfig(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want []string
	}{
		{name: "unset_uses_defaults", yaml: "timeout: 1s", want: defaultSensitiveHeaders},
		{name: "null_uses_defaults", yaml: "redirect.sensitive_headers: ~", want: defaultSensitiveHeaders},
		{name: "empty_disables", yaml: "redirect.sensitive_headers: []", want: []string{}},
		{name: "custom_replaces_defaults", yaml: "redirect.sensitive_headers: [X-Api-Key]", want: []string{"X-Api-Key"}},
		{name: "longer_custom_replaces_defaults", yaml: "redirect.sensitive_headers: [A, B, C, D]", want: []string{"A", "B", "C", "D"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := config.NewConfigWithYAML([]byte(test.yaml), "")
			if err != nil {
				t.Fatalf("parsing YAML: %v", err)
			}
			rc := DefaultRequestConfig(time.Second)
			if err := cfg.Unpack(&rc); err != nil {
				t.Fatalf("Unpack: %v", err)
			}
			if got := sensitiveHeaders(rc.RedirectSensitiveHeaders); !slices.Equal(got, test.want) {
				t.Errorf("sensitive headers = %q; want %q", got, test.want)
			}
		})
	}
}

func TestTracerConfigValidate(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]any{"tracer.enabled": true})
	var c struct {
		Tracer *TracerConfig `config:"tracer"`
	}
	err := cfg.Unpack(&c)
	if err == nil || !strings.Contains(err.Error(), "request tracer must have a filename") {
		t.Errorf("Unpack() = %v; want an error about the missing filename", err)
	}
}

func TestSkipTracing(t *testing.T) {
	var got []string // "traced" or "direct" for each request, in order
	respond := func(route string) http.RoundTripper {
		return roundTripperFunc(func(*http.Request) (*http.Response, error) {
			got = append(got, route)
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		})
	}
	traced, direct := respond("traced"), respond("direct")

	tests := []struct {
		name     string
		prefixes []string
		url      string
		want     string
	}{
		{name: "token_request", prefixes: []string{"https://login.example/"}, url: "https://login.example/tenant/oauth2/v2.0/token", want: "direct"},
		{name: "api_request", prefixes: []string{"https://login.example/"}, url: "https://graph.example/v1.0/users", want: "traced"},
		{name: "case_differs", prefixes: []string{"HTTPS://Login.Example/"}, url: "https://login.example/tenant/oauth2/v2.0/token", want: "direct"},
		{name: "other_path", prefixes: []string{"https://example/login/"}, url: "https://example/v1.0/users", want: "traced"},
		{name: "no_prefixes", url: "https://login.example/tenant/oauth2/v2.0/token", want: "traced"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got = nil
			rt, err := SkipTracing(traced, direct, test.prefixes)
			if err != nil {
				t.Fatalf("SkipTracing: %v", err)
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, test.url, nil)
			if err != nil {
				t.Fatalf("NewRequest(%s): %v", test.url, err)
			}
			resp, err := rt.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip(%s): %v", test.url, err)
			}
			resp.Body.Close()
			if len(got) != 1 || got[0] != test.want {
				t.Errorf("request to %s went %v; want [%s]", test.url, got, test.want)
			}
		})
	}

	if _, err := SkipTracing(traced, direct, []string{"https://bad host/"}); err == nil {
		t.Error("SkipTracing accepted an invalid prefix")
	}
}

func TestKeepAlive(t *testing.T) {
	if (KeepAlive{}).set() {
		t.Error("zero KeepAlive reports set; want unset so keep-alives keep their default")
	}
	disable := true
	ka := KeepAlive{Disable: &disable, MaxIdleConns: 3}
	if !ka.set() {
		t.Error("configured KeepAlive reports unset")
	}
	s := ka.settings()
	if !s.Disable || s.MaxIdleConns != 3 {
		t.Errorf("settings() = %+v; want Disable true and MaxIdleConns 3", s)
	}
	if (KeepAlive{MaxIdleConns: 3}).settings().Disable {
		t.Error("KeepAlive without disable disables keep-alives; want them enabled")
	}
}

func TestTraceRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	newPaths := func(t *testing.T) (*paths.Path, string) {
		logs := filepath.Join(t.TempDir(), "logs")
		return &paths.Path{Logs: logs}, filepath.Join(logs, "okta")
	}
	// The file name the Fleet integration sends.
	const filename = "../../logs/okta/http-request-trace-*.ndjson"

	t.Run("enabled", func(t *testing.T) {
		p, dir := newPaths(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		cfg := &TracerConfig{}
		cfg.Filename = filename
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		rt, err := TraceRequests(ctx, http.DefaultTransport, cfg, p, "okta", "input-1", logptest.NewTestingLogger(t, ""))
		if err != nil {
			t.Fatalf("TraceRequests: %v", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		req.Header.Set("Authorization", "SSWS secret-token")
		resp, err := (&http.Client{Transport: rt}).Do(req)
		if err != nil {
			t.Fatalf("Do: %v", err)
		}
		resp.Body.Close()

		b, err := os.ReadFile(filepath.Join(dir, "http-request-trace-input-1.ndjson"))
		if err != nil {
			t.Fatalf("reading trace: %v", err)
		}
		if !strings.Contains(string(b), srv.URL) {
			t.Errorf("trace does not mention the request URL %s:\n%s", srv.URL, b)
		}
		if strings.Contains(string(b), "secret-token") {
			t.Errorf("trace contains the Authorization value:\n%s", b)
		}
	})

	t.Run("disabled_removes_files", func(t *testing.T) {
		p, dir := newPaths(t)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		old := filepath.Join(dir, "http-request-trace-input-1.ndjson")
		if err := os.WriteFile(old, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		disabled := false
		cfg := &TracerConfig{Enabled: &disabled}
		cfg.Filename = filename
		rt, err := TraceRequests(t.Context(), http.DefaultTransport, cfg, p, "okta", "input-1", logptest.NewTestingLogger(t, ""))
		if err != nil {
			t.Fatalf("TraceRequests: %v", err)
		}
		if rt != http.DefaultTransport {
			t.Error("disabled tracer wrapped the transport")
		}
		if _, err := os.Stat(old); !os.IsNotExist(err) {
			t.Errorf("old trace file still exists (stat error %v)", err)
		}
	})

	t.Run("outside_logs", func(t *testing.T) {
		p, _ := newPaths(t)
		cfg := &TracerConfig{}
		cfg.Filename = "/tmp/trace.ndjson"
		if _, err := TraceRequests(t.Context(), http.DefaultTransport, cfg, p, "okta", "input-1", logptest.NewTestingLogger(t, "")); err == nil {
			t.Error("TraceRequests accepted a file outside the logs directory")
		}
	})

	t.Run("not_configured", func(t *testing.T) {
		p, _ := newPaths(t)
		rt, err := TraceRequests(t.Context(), http.DefaultTransport, nil, p, "okta", "input-1", logptest.NewTestingLogger(t, ""))
		if err != nil || rt != http.DefaultTransport {
			t.Errorf("TraceRequests(nil config) = %v, %v; want the transport unchanged", rt, err)
		}
	})
}
