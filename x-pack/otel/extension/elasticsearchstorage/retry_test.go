// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		name   string
		status int
		err    error
		want   bool
	}{
		{"success", http.StatusOK, nil, false},
		{"network error status 0", 0, errors.New("dial tcp: connection refused"), true},
		{"429", http.StatusTooManyRequests, errors.New("429"), true},
		{"502", http.StatusBadGateway, errors.New("502"), true},
		{"503", http.StatusServiceUnavailable, errors.New("503"), true},
		{"504", http.StatusGatewayTimeout, errors.New("504"), true},
		{"400 permanent", http.StatusBadRequest, errors.New("400"), false},
		{"401 permanent", http.StatusUnauthorized, errors.New("401"), false},
		{"404 permanent", http.StatusNotFound, errors.New("404"), false},
		{"409 permanent", http.StatusConflict, errors.New("409"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isRetryable(tc.status, tc.err))
		})
	}
}

func TestBackoffDelay(t *testing.T) {
	base := 100 * time.Millisecond
	max := 5 * time.Second

	assert.Equal(t, base, backoffDelay(0, base, max), "attempt 0 == base")
	assert.Equal(t, 200*time.Millisecond, backoffDelay(1, base, max))
	assert.Equal(t, 400*time.Millisecond, backoffDelay(2, base, max))
	assert.Equal(t, max, backoffDelay(20, base, max), "large attempt clamps to max")
	assert.LessOrEqual(t, int64(backoffDelay(100, base, max)), int64(max), "must never exceed max or overflow")
}

func TestRetryConfig_ClampsBaseDelayToMaxDelay(t *testing.T) {
	// A programmatically-built config can carry base > max (Validate is not
	// run for it); retryConfig must clamp so backoffDelay never sees it.
	ext := &elasticStorage{cfg: &Config{
		Retry: RetryConfig{MaxAttempts: 3, BaseDelay: 10 * time.Second, MaxDelay: 5 * time.Second},
	}}
	p := ext.retryConfig()
	assert.Equal(t, p.maxDelay, p.baseDelay, "base delay must be clamped to max delay")
}

// newRetryTestExtension stands up an extension whose connection points at srv,
// with fast, deterministic retry settings so the retry loop can be exercised
// without a real cluster and without slow backoff.
func newRetryTestExtension(t *testing.T, srv *httptest.Server) *elasticStorage {
	t.Helper()
	cfg := &Config{
		ElasticsearchConfig: map[string]any{
			"hosts":    []string{srv.URL},
			"username": "elastic",
			"password": "changeme",
		},
		Retry: RetryConfig{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
	}
	ext := &elasticStorage{cfg: cfg, logger: logptest.NewTestingLogger(t, t.Name())}
	require.NoError(t, ext.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })
	return ext
}

func retryTestClient(t *testing.T, ext *elasticStorage) *esStorageClient {
	t.Helper()
	id := component.MustNewIDWithName("retry_test", "c")
	c, err := ext.GetClient(context.Background(), component.KindReceiver, id, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	return c.(*esStorageClient)
}

// TestRequest_RetriesTransientThenSucceeds drives Set against a server that
// returns 503 twice on the document write before succeeding, and asserts the
// write ultimately succeeds via the retry loop.
func TestRequest_RetriesTransientThenSucceeds(t *testing.T) {
	var docAttempts atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			_, _ = io.WriteString(w, `{"version":{"number":"8.10.0","build_flavor":"default"},"name":"fake"}`)
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/_doc/"):
			if docAttempts.Add(1) <= 2 {
				http.Error(w, `{"error":"unavailable"}`, http.StatusServiceUnavailable)
				return
			}
			_, _ = io.WriteString(w, `{"result":"created"}`)
		default: // index create (PUT /<index>) and anything else
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer srv.Close()

	ext := newRetryTestExtension(t, srv)
	c := retryTestClient(t, ext)

	require.NoError(t, c.Set(context.Background(), "k", []byte(`{"a":1}`)),
		"Set must succeed after transient 503s are retried")
	assert.Equal(t, int64(3), docAttempts.Load(), "expected 2 failures + 1 success")
}

// TestRequest_IndexCreateRetriedThenSucceeds drives Set against a server that
// returns 503 twice on the index create (PUT /<index>) before acknowledging,
// and asserts the create goes through the same retry loop as document writes.
func TestRequest_IndexCreateRetriedThenSucceeds(t *testing.T) {
	var createAttempts atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			_, _ = io.WriteString(w, `{"version":{"number":"8.10.0","build_flavor":"default"},"name":"fake"}`)
		case r.Method == http.MethodPut && !strings.Contains(r.URL.Path, "/_doc/"):
			if createAttempts.Add(1) <= 2 {
				http.Error(w, `{"error":"unavailable"}`, http.StatusServiceUnavailable)
				return
			}
			_, _ = io.WriteString(w, `{"acknowledged":true}`)
		default: // document write and anything else
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer srv.Close()

	ext := newRetryTestExtension(t, srv)
	c := retryTestClient(t, ext)

	require.NoError(t, c.Set(context.Background(), "k", []byte(`{"a":1}`)),
		"Set must succeed after transient 503s on index creation are retried")
	assert.Equal(t, int64(3), createAttempts.Load(), "expected 2 failed creates + 1 success")
}

// TestRequest_PermanentErrorNotRetried asserts a 400 on the document write is
// returned immediately without retrying.
func TestRequest_PermanentErrorNotRetried(t *testing.T) {
	var docAttempts atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			_, _ = io.WriteString(w, `{"version":{"number":"8.10.0","build_flavor":"default"},"name":"fake"}`)
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/_doc/"):
			docAttempts.Add(1)
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		default:
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer srv.Close()

	ext := newRetryTestExtension(t, srv)
	c := retryTestClient(t, ext)

	err := c.Set(context.Background(), "k", []byte(`{"a":1}`))
	require.Error(t, err, "a permanent 400 must surface as an error")
	assert.Equal(t, int64(1), docAttempts.Load(), "permanent errors must not be retried")
}

// TestRequest_RetryStopsOnContextCancel asserts an in-flight backoff is
// interrupted when the operation context is cancelled.
func TestRequest_RetryStopsOnContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			_, _ = io.WriteString(w, `{"version":{"number":"8.10.0","build_flavor":"default"},"name":"fake"}`)
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/_doc/"):
			// The document write always fails transiently, forcing the retry
			// loop into a long backoff that the cancel must interrupt.
			http.Error(w, `{"error":"unavailable"}`, http.StatusServiceUnavailable)
		default: // index create succeeds so Set reaches the retry loop
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer srv.Close()

	// Long backoff so the cancel wins the race deterministically.
	cfg := &Config{
		ElasticsearchConfig: map[string]any{
			"hosts":    []string{srv.URL},
			"username": "elastic",
			"password": "changeme",
		},
		Retry: RetryConfig{MaxAttempts: 10, BaseDelay: time.Second, MaxDelay: time.Second},
	}
	ext := &elasticStorage{cfg: cfg, logger: logptest.NewTestingLogger(t, t.Name())}
	require.NoError(t, ext.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	id := component.MustNewIDWithName("retry_test", "cancel")
	client, err := ext.GetClient(context.Background(), component.KindReceiver, id, "")
	require.NoError(t, err)
	c := client.(*esStorageClient)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err = c.Set(ctx, "k", []byte(`{"a":1}`))
	require.Error(t, err)
	assert.Less(t, time.Since(start), time.Second, "cancel must interrupt the backoff well before a full 1s delay elapses")
}
