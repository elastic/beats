// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

var _ http.RoundTripper = (*rateLimitTransport)(nil)

// rateLimitTransport wraps an http.RoundTripper to intercept 429 responses
// and retry after the duration indicated by the Retry-After header. This sits
// below the oauth2 transport so that rate-limited token-refresh requests are
// retried before the oauth2 library sees the failure and generates additional
// unauthorized requests.
type rateLimitTransport struct {
	base     http.RoundTripper
	maxRetry int
	wait     time.Duration // default wait when Retry-After is absent
	log      *logp.Logger
	now      func() time.Time
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Buffer the body so POST requests (token endpoint) can be replayed.
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("rate limit transport: reading request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	for attempt := 0; ; attempt++ {
		resp, err := t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests || attempt >= t.maxRetry {
			return resp, nil
		}

		wait := parseRetryAfter(resp.Header.Get("Retry-After"), t.wait, t.timeNow())
		resp.Body.Close()

		t.log.Warnw("rate limited, backing off",
			"attempt", attempt+1,
			"max_retries", t.maxRetry,
			"retry_after", wait,
		)

		timer := time.NewTimer(wait)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		case <-timer.C:
		}

		if body != nil {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
	}
}

func (t *rateLimitTransport) timeNow() time.Time {
	if t.now != nil {
		return t.now()
	}
	return time.Now()
}

// parseRetryAfter parses a Retry-After header as either an integer number
// of seconds or an HTTP-date (RFC 7231 §7.1.3). If the value is empty or
// unparseable, fallback is returned. ref is the reference time for computing
// the delay from an HTTP-date.
//
// CrowdStrike's documented response headers include X-Ratelimit-Limit and
// X-Ratelimit-Remaining but not Retry-After. The 429 that triggers this
// code path is a security rate limit (15 unauthorized requests/minute/IP),
// not the normal 6000 req/min API limit, and may return no retry guidance
// at all. The fallback duration (60s, matching the rate-limit window) is
// expected to do the real work in practice; the header parsing is defensive.
func parseRetryAfter(val string, fallback time.Duration, ref time.Time) time.Duration {
	val = strings.TrimSpace(val)
	if val == "" {
		return fallback
	}
	if secs, err := strconv.ParseInt(val, 10, 64); err == nil {
		if secs > 0 {
			return time.Duration(secs) * time.Second
		}
		return fallback
	}
	if t, err := http.ParseTime(val); err == nil {
		if d := t.Sub(ref); d > 0 {
			return d
		}
		return fallback
	}
	return fallback
}
