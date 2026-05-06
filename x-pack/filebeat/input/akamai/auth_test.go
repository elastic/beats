// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRetryNonceFreshness proves that every request sent through EdgeGridTransport
// carries a unique nonce and timestamp in its Authorization header, even across
// retries to the same endpoint.
func TestRetryNonceFreshness(t *testing.T) {
	var mu sync.Mutex
	var authHeaders []string
	requestCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		requestCount++
		idx := requestCount
		mu.Unlock()

		if idx <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"detail":"temporary failure"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total":0,"offset":"","limit":10}` + "\n"))
	}))
	defer srv.Close()

	signer := NewEdgeGridSigner("test-client-token", "test-client-secret-key", "test-access-token")
	transport := &EdgeGridTransport{
		Transport: http.DefaultTransport,
		Signer:    signer,
	}

	client := &http.Client{Transport: transport}

	for i := 0; i < 3; i++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/test", nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
	}

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, authHeaders, 3)

	nonceRe := regexp.MustCompile(`nonce=([^;]+)`)
	tsRe := regexp.MustCompile(`timestamp=([^;]+)`)

	nonces := make(map[string]bool)
	timestamps := make(map[string]bool)
	for _, h := range authHeaders {
		nm := nonceRe.FindStringSubmatch(h)
		require.Len(t, nm, 2, "nonce not found in Authorization header: %s", h)
		nonces[nm[1]] = true

		tm := tsRe.FindStringSubmatch(h)
		require.Len(t, tm, 2, "timestamp not found in Authorization header: %s", h)
		timestamps[tm[1]] = true
	}

	assert.Len(t, nonces, 3, "each request must use a unique nonce")
}
