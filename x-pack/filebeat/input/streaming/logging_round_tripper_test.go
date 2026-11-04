// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestLoggingRoundTripperLogOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=response-secret")
	}))
	defer srv.Close()

	log, logs := logptest.NewTestingLoggerWithObserver(t, "")
	client := &http.Client{Transport: &loggingRoundTripper{http.DefaultTransport, log}}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Basic request-secret")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	var sb strings.Builder
	for _, e := range logs.All() {
		fmt.Fprintln(&sb, e.Message)
	}
	messages := sb.String()

	for _, secret := range []string{"request-secret", "response-secret"} {
		if strings.Contains(messages, secret) {
			t.Errorf("log output contains header value %q: %s", secret, messages)
		}
	}
	for _, name := range []string{"Authorization", "Set-Cookie"} {
		if !strings.Contains(messages, name) {
			t.Errorf("log output does not contain header name %q: %s", name, messages)
		}
	}
}
