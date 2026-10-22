// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestCheckRedirectSensitiveHeaders(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")

	tests := []struct {
		name             string
		prevURL          string
		reqURL           string
		sensitiveHeaders []string
		wantAuth         bool
		wantProxyAuth    bool
		wantCookie       bool
		wantCustom       bool
	}{
		{
			name:             "same_origin_preserves_all_headers",
			prevURL:          "https://api.example.com/v1/data",
			reqURL:           "https://api.example.com/v1/other",
			sensitiveHeaders: []string{"Authorization", "Proxy-Authorization", "Cookie"},
			wantAuth:         true,
			wantProxyAuth:    true,
			wantCookie:       true,
			wantCustom:       true,
		},
		{
			name:             "cross-origin_strips_sensitive_headers",
			prevURL:          "https://api.example.com/v1/data",
			reqURL:           "https://evil.example.net/capture",
			sensitiveHeaders: []string{"Authorization", "Proxy-Authorization", "Cookie"},
			wantAuth:         false,
			wantProxyAuth:    false,
			wantCookie:       false,
			wantCustom:       true,
		},
		{
			name:             "scheme_downgrade_strips_sensitive_headers",
			prevURL:          "https://api.example.com/v1/data",
			reqURL:           "http://api.example.com/v1/data",
			sensitiveHeaders: []string{"Authorization", "Proxy-Authorization", "Cookie"},
			wantAuth:         false,
			wantProxyAuth:    false,
			wantCookie:       false,
			wantCustom:       true,
		},
		{
			name:             "empty_sensitive_headers_preserves_all_cross-origin",
			prevURL:          "https://api.example.com/v1/data",
			reqURL:           "https://other.example.net/resource",
			sensitiveHeaders: []string{},
			wantAuth:         true,
			wantProxyAuth:    true,
			wantCookie:       true,
			wantCustom:       true,
		},
		{
			name:             "scheme_upgrade_same_host_preserves_headers",
			prevURL:          "http://api.example.com/v1/data",
			reqURL:           "https://api.example.com/v1/secure",
			sensitiveHeaders: []string{"Authorization", "Proxy-Authorization", "Cookie"},
			wantAuth:         true,
			wantProxyAuth:    true,
			wantCookie:       true,
			wantCustom:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &ResourceConfig{
				RedirectFollow:           true,
				RedirectForwardHeaders:   true,
				RedirectSensitiveHeaders: test.sensitiveHeaders,
				RedirectMaxRedirects:     10,
			}

			prev := &http.Request{
				URL: mustParseURL(t, test.prevURL),
				Header: http.Header{
					"Authorization":       {"Bearer secret"},
					"Proxy-Authorization": {"Basic proxy-creds"},
					"Cookie":              {"session=abc123"},
					"X-Custom":            {"keep-me"},
				},
			}

			req := &http.Request{
				URL:    mustParseURL(t, test.reqURL),
				Header: http.Header{},
			}

			fn := checkRedirect(cfg, log)
			err := fn(req, []*http.Request{prev})
			if err != nil {
				t.Fatalf("checkRedirect returned error: %v", err)
			}

			check(t, req.Header, "Authorization", test.wantAuth)
			check(t, req.Header, "Proxy-Authorization", test.wantProxyAuth)
			check(t, req.Header, "Cookie", test.wantCookie)
			check(t, req.Header, "X-Custom", test.wantCustom)
		})
	}
}

func TestRedirectCrossHost(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")

	tests := []struct {
		name             string
		downloadHostname string // Hostname used for the redirect target; empty to use the API's.
		forwardHeaders   bool
		sensitiveHeaders []string
		wantAuth         bool
	}{
		{
			name:             "different_hostname_headers_not_forwarded",
			downloadHostname: "localhost",
			sensitiveHeaders: []string{"Authorization"},
			wantAuth:         false,
		},
		{
			name:             "different_hostname_forwarded_sensitive_stripped",
			downloadHostname: "localhost",
			forwardHeaders:   true,
			sensitiveHeaders: []string{"Authorization"},
			wantAuth:         false,
		},
		{
			name:             "different_hostname_forwarded_no_sensitive_headers",
			downloadHostname: "localhost",
			forwardHeaders:   true,
			wantAuth:         true,
		},
		{
			// The standard library compares hostnames without the port,
			// so it does not strip headers when only the port differs.
			name:             "same_hostname_different_port_headers_not_forwarded",
			sensitiveHeaders: []string{"Authorization"},
			wantAuth:         true,
		},
		{
			// checkRedirect compares URL hosts including the port.
			name:             "same_hostname_different_port_forwarded_sensitive_stripped",
			forwardHeaders:   true,
			sensitiveHeaders: []string{"Authorization"},
			wantAuth:         false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotAuth string
			download := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				_, _ = w.Write([]byte("hello"))
			}))
			defer download.Close()

			target := download.URL + "/file"
			if test.downloadHostname != "" {
				target = strings.Replace(target, "127.0.0.1", test.downloadHostname, 1)
			}
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", target)
				w.Header().Set("Content-Length", "2")
				w.WriteHeader(http.StatusFound)
				_, _ = w.Write([]byte("{}"))
			}))
			defer api.Close()

			cfg := &ResourceConfig{
				RedirectFollow:           true,
				RedirectForwardHeaders:   test.forwardHeaders,
				RedirectSensitiveHeaders: test.sensitiveHeaders,
				RedirectMaxRedirects:     10,
			}
			client := &http.Client{CheckRedirect: checkRedirect(cfg, log)}

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, api.URL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Authorization", "Bearer secret")
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			if string(body) != "hello" {
				t.Errorf("unexpected body: got:%q want:%q", body, "hello")
			}
			if gotAuth != "" != test.wantAuth {
				t.Errorf("unexpected Authorization at download server: got:%q want present:%t", gotAuth, test.wantAuth)
			}
		})
	}
}

func check(t *testing.T, h http.Header, key string, wantPresent bool) {
	t.Helper()
	_, got := h[key]
	if got != wantPresent {
		if wantPresent {
			t.Errorf("expected header %s to be present, but it was stripped", key)
		} else {
			t.Errorf("expected header %s to be stripped, but it is present with value %q", key, h.Get(key))
		}
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", raw, err)
	}
	return u
}
