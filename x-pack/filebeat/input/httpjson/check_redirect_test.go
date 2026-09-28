// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"net/url"
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
			cfg := &requestConfig{
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
