// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestCheckRedirectSensitiveHeaders(t *testing.T) {
	tests := []struct {
		name             string
		prevURL          string
		reqURL           string
		sensitiveHeaders []string
		wantStripped     bool
	}{
		{
			name:             "same_origin_preserves_headers",
			prevURL:          "https://api.example.com/v1/data",
			reqURL:           "https://api.example.com/v1/other",
			sensitiveHeaders: []string{"Authorization", "Proxy-Authorization", "Cookie"},
		},
		{
			name:             "cross-origin_strips_sensitive_headers",
			prevURL:          "https://api.example.com/v1/data",
			reqURL:           "https://evil.example.net/capture",
			sensitiveHeaders: []string{"Authorization", "Proxy-Authorization", "Cookie"},
			wantStripped:     true,
		},
		{
			name:             "scheme_downgrade_strips_sensitive_headers",
			prevURL:          "https://api.example.com/v1/data",
			reqURL:           "http://api.example.com/v1/data",
			sensitiveHeaders: []string{"Authorization", "Proxy-Authorization", "Cookie"},
			wantStripped:     true,
		},
		{
			name:             "empty_sensitive_headers_preserves_cross-origin",
			prevURL:          "https://api.example.com/v1/data",
			reqURL:           "https://other.example.net/resource",
			sensitiveHeaders: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log := logptest.NewTestingLogger(t, "")
			cfg := &requestConfig{
				RedirectForwardHeaders:   true,
				RedirectSensitiveHeaders: test.sensitiveHeaders,
				RedirectMaxRedirects:     10,
			}
			prev := &http.Request{
				URL: mustParseURL(t, test.prevURL),
				Header: http.Header{
					"Authorization":       {"SSWS secret"},
					"Proxy-Authorization": {"Basic proxy-creds"},
					"Cookie":              {"session=abc123"},
					"X-Custom":            {"keep-me"},
				},
			}
			req := &http.Request{URL: mustParseURL(t, test.reqURL), Header: http.Header{}}

			err := checkRedirect(cfg, log)(req, []*http.Request{prev})
			if err != nil {
				t.Fatalf("checkRedirect returned error: %v", err)
			}

			for _, k := range []string{"Authorization", "Proxy-Authorization", "Cookie"} {
				_, present := req.Header[k]
				if present == test.wantStripped {
					t.Errorf("header %s present = %t; want %t", k, present, !test.wantStripped)
				}
			}
			if req.Header.Get("X-Custom") != "keep-me" {
				t.Errorf("X-Custom = %q; want %q", req.Header.Get("X-Custom"), "keep-me")
			}
		})
	}
}

func TestCheckRedirectLogOutput(t *testing.T) {
	log, logs := logptest.NewTestingLoggerWithObserver(t, "")
	cfg := &requestConfig{RedirectForwardHeaders: true, RedirectMaxRedirects: 10}
	prev := &http.Request{
		URL:    mustParseURL(t, "https://api.example.com/"),
		Header: http.Header{"Authorization": {"SSWS super-secret-token"}},
	}
	req := &http.Request{URL: mustParseURL(t, "https://api.example.com/next"), Header: http.Header{}}

	err := checkRedirect(cfg, log)(req, []*http.Request{prev})
	if err != nil {
		t.Fatalf("checkRedirect returned error: %v", err)
	}

	var logged bool
	for _, e := range logs.All() {
		if strings.Contains(e.Message, "super-secret-token") {
			t.Errorf("log entry leaks header value: %q", e.Message)
		}
		if strings.Contains(e.Message, "[Authorization]") {
			logged = true
		}
	}
	if !logged {
		t.Error("no log entry lists the forwarded header names")
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

func TestRedirectCrossHost(t *testing.T) {
	tests := []struct {
		name             string
		downloadHostname string // Hostname used for the redirect target; empty to use the API's.
		sensitiveHeaders []string
		wantAuth         bool
	}{
		{
			name:             "different_hostname_stripped_by_default",
			downloadHostname: "localhost",
			wantAuth:         false,
		},
		{
			name:             "different_hostname_empty_list_forwards",
			downloadHostname: "localhost",
			sensitiveHeaders: []string{},
			wantAuth:         true,
		},
		{
			// Hosts are compared including the port.
			name:     "same_hostname_different_port_stripped_by_default",
			wantAuth: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log := logptest.NewTestingLogger(t, "")

			var gotAuth string
			download := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
			}))
			defer download.Close()

			target := download.URL + "/file"
			if test.downloadHostname != "" {
				target = strings.Replace(target, "127.0.0.1", test.downloadHostname, 1)
			}
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target, http.StatusFound)
			}))
			defer api.Close()

			cfg := defaultConfig()
			cfg.Request.RedirectForwardHeaders = true
			cfg.Request.RedirectSensitiveHeaders = test.sensitiveHeaders
			client, err := newClient(t.Context(), cfg, log)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, api.URL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Authorization", "secret")
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()

			if (gotAuth != "") != test.wantAuth {
				t.Errorf("Authorization at download server = %q; want present: %t", gotAuth, test.wantAuth)
			}
		})
	}
}
