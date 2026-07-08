// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package elasticsearchstorage

import (
	"context"
	"net/http"
	"time"
)

// retryParams is the resolved (default-applied) retry configuration.
type retryParams struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
}

// request runs an ES request under clientMu with bounded retry on transient
// failures. The returned body is a fresh copy, safe to use after the
// connection's response buffer is reused by a later request. Callers MUST NOT
// hold clientMu.
func (c *esStorageClient) request(ctx context.Context, method, path string, params map[string]string, body interface{}) (int, []byte, error) {
	rc := c.ext.retryConfig()

	// extDone lets a slow retry backoff be interrupted when the extension's
	// runtime context is cancelled (e.g. Shutdown). A nil channel blocks
	// forever, which is the correct "ignore" behaviour when unset.
	var extDone <-chan struct{}
	if c.ext.ctx != nil {
		extDone = c.ext.ctx.Done()
	}

	var (
		status int
		out    []byte
		err    error
	)
	for attempt := 0; ; attempt++ {
		status, out, err = c.doOnce(method, path, params, body)
		if !isRetryable(status, err) || attempt+1 >= rc.maxAttempts {
			return status, out, err
		}

		timer := time.NewTimer(backoffDelay(attempt, rc.baseDelay, rc.maxDelay))
		select {
		case <-ctx.Done():
			timer.Stop()
			return status, out, ctx.Err()
		case <-extDone:
			timer.Stop()
			return status, out, context.Canceled
		case <-timer.C:
		}
	}
}

// doOnce performs a single request under clientMu, copying the response body
// out before releasing the lock (the connection reuses its response buffer).
func (c *esStorageClient) doOnce(method, path string, params map[string]string, body interface{}) (int, []byte, error) {
	c.ext.clientMu.Lock()
	defer c.ext.clientMu.Unlock()

	status, b, err := c.ext.client.Request(method, path, "", params, body)
	var out []byte
	if len(b) > 0 {
		out = make([]byte, len(b))
		copy(out, b)
	}
	return status, out, err
}

// isRetryable reports whether a failed request should be retried. Only
// transient classes qualify: network errors (no HTTP response, status 0) and
// 429/502/503/504. Permanent responses (400/401/403/404/409/...) are not
// retried. A nil error is a success and never retried.
func isRetryable(status int, err error) bool {
	if err == nil {
		return false
	}
	switch status {
	case 0:
		return true
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// backoffDelay returns the capped exponential backoff for a zero-based
// attempt: base * 2^attempt, clamped to max (and guarded against overflow).
func backoffDelay(attempt int, base, max time.Duration) time.Duration {
	if attempt < 0 || base <= 0 {
		return base
	}
	d := base
	for i := 0; i < attempt; i++ {
		d *= 2
		if d <= 0 || d >= max {
			return max
		}
	}
	if d > max {
		return max
	}
	return d
}
