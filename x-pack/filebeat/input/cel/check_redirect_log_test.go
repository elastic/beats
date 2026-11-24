// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"net/http"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

func TestCheckRedirectLogOutput(t *testing.T) {
	log, logs := logptest.NewTestingLoggerWithObserver(t, "")
	cfg := &ResourceConfig{RedirectForwardHeaders: true, RedirectMaxRedirects: 10}
	prev := &http.Request{
		URL:    mustParseURL(t, "https://api.example.com/"),
		Header: http.Header{"Authorization": {"Bearer super-secret-token"}},
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
