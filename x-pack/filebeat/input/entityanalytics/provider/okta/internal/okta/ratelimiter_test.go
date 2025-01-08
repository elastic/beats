// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestRateLimiter(t *testing.T) {
	logp.TestingSetup()

	t.Run("separation by endpoint", func(t *testing.T) {
		const window = time.Minute
		var fixedLimit *int = nil
		r := NewRateLimiter(window, fixedLimit)
		e1 := r.endpoint("/foo")
		e2 := r.endpoint("/bar")

		e1.limiter.SetBurst(1000)

		if e2.limiter.Burst() == 1000 {
			t.Errorf("changes to one endpoint's limits affected another")
		}
	})

	t.Run("Update stops requests when none are remaining", func(t *testing.T) {
		t.Skip("Flaky test: https://github.com/elastic/beats/issues/42059")
		const window = time.Minute
		var fixedLimit *int = nil
		r := NewRateLimiter(window, fixedLimit)
		const endpoint = "/foo"
		url, err := url.Parse(endpoint)
		if err != nil {
			t.Errorf("unexpected error from url.Parse(): %v", err)
		}
		ctx := context.Background()
		log := logp.L()
		e := r.endpoint(endpoint)

		if !e.limiter.Allow() {
			t.Errorf("doesn't allow an initial request")
		}

		// update to none remaining, reset soon
		now := time.Now().Unix()
		resetSoon := now + 30
		headers := http.Header{
			"X-Rate-Limit-Limit":     []string{"60"},
			"X-Rate-Limit-Remaining": []string{"0"},
			"X-Rate-Limit-Reset":     []string{strconv.FormatInt(resetSoon, 10)},
		}
		err = r.Update(endpoint, headers, logp.L())
		if err != nil {
			t.Errorf("unexpected error from Update(): %v", err)
		}
		e = r.endpoint(endpoint)

		if e.limiter.Allow() {
			t.Errorf("allowed a request when none are remaining")
		}
		if e.limiter.AllowN(time.Unix(resetSoon-1, 999999999), 1) {
			t.Errorf("allowed a request before reset, when none are remaining")
		}

		// update to none remaining, reset now
		headers = http.Header{
			"X-Rate-Limit-Limit":     []string{"60"},
			"X-Rate-Limit-Remaining": []string{"0"},
			"X-Rate-Limit-Reset":     []string{strconv.FormatInt(now, 10)},
		}
		err = r.Update(endpoint, headers, logp.L())
		if err != nil {
			t.Errorf("unexpected error from Update(): %v", err)
		}
		e = r.endpoint(endpoint)

		start := time.Now()
		r.Wait(ctx, endpoint, url, log)
		wait := time.Since(start)

		if wait > 1100*time.Millisecond {
			t.Errorf("doesn't allow requests to resume after reset. had to wait %s", wait)
		}
		if e.limiter.Limit() != 1.0 {
			t.Errorf("unexpected rate following reset (not 60 requests / 60 seconds): %f", e.limiter.Limit())
		}
		if e.limiter.Burst() != 1 {
			t.Errorf("unexpected burst following reset (not 1): %d", e.limiter.Burst())
		}

		e.limiter.SetBurst(100) // increase bucket size to check token accumulation
		tokens := e.limiter.TokensAt(time.Unix(0, time.Now().Add(30*time.Second).UnixNano()))
		target := 30.0
		buffer := 0.1

		if tokens < target-buffer || target+buffer < tokens {
			t.Errorf("tokens don't accumulate at the expected rate over 30s: %f", tokens)
		}
	})

	t.Run("Very long waits are considered errors", func(t *testing.T) {
		const window = time.Minute
		var fixedLimit *int = nil
		r := NewRateLimiter(window, fixedLimit)

		const endpoint = "/foo"

		url, err := url.Parse(endpoint)
		if err != nil {
			t.Errorf("unexpected error from url.Parse(): %v", err)
		}
		reset := time.Now().Add(31 * time.Minute).Unix()
		headers := http.Header{
			"X-Rate-Limit-Limit":     []string{"60"},
			"X-Rate-Limit-Remaining": []string{"1"},
			"X-Rate-Limit-Reset":     []string{strconv.FormatInt(reset, 10)},
		}
		log := logp.L()
		ctx := context.Background()

		r.Wait(ctx, endpoint, url, log)  // consume the initial request
		r.Update(endpoint, headers, log) // update to a slow rate

		err = r.Wait(ctx, endpoint, url, log)

		const expectedErr = "rate: Wait(n=1) would exceed context deadline"
		if err == nil {
			t.Errorf("expected error message %q, but got no error", expectedErr)
		} else if err.Error() != expectedErr {
			t.Errorf("expected error message %q, but got %q", expectedErr, err.Error())
		}
	})

	t.Run("A fixed limit overrides response information", func(t *testing.T) {
		const window = time.Minute
		var fixedLimit int = 120
		r := NewRateLimiter(window, &fixedLimit)
		const endpoint = "/foo"
		e := r.endpoint(endpoint)

		if e.limiter.Limit() != 120/60 {
			t.Errorf("unexpected rate (for fixed 120 reqs / 60 secs): %f", e.limiter.Limit())
		}

		// update to 15 requests remaining, reset in 30s
		headers := http.Header{
			"X-Rate-Limit-Limit":     []string{"60"},
			"X-Rate-Limit-Remaining": []string{"15"},
			"X-Rate-Limit-Reset":     []string{strconv.FormatInt(time.Now().Unix()+30, 10)},
		}
		err := r.Update(endpoint, headers, logp.L())
		if err != nil {
			t.Errorf("unexpected error from Update(): %v", err)
		}
		e = r.endpoint(endpoint)

		if e.limiter.Limit() != 120/60 {
			t.Errorf("unexpected rate following Update() (for fixed 120 reqs / 60 secs): %f", e.limiter.Limit())
		}
	})

	t.Run("A concurrent rate limit should not set a new rate of zero", func(t *testing.T) {
		const window = time.Minute
		r := NewRateLimiter(window, nil)
		const endpoint = "/foo"
		url, err := url.Parse(endpoint)
		if err != nil {
			t.Errorf("unexpected error from url.Parse(): %v", err)
		}
		ctx := context.Background()
		log := logp.L()

		// update to 30 requests remaining, reset in 30s
		headers := http.Header{
			"X-Rate-Limit-Limit":     []string{"60"},
			"X-Rate-Limit-Remaining": []string{"30"},
			"X-Rate-Limit-Reset":     []string{strconv.FormatInt(time.Now().Unix()+30, 10)},
		}
		err = r.Update(endpoint, headers, logp.L())
		if err != nil {
			t.Errorf("unexpected error from Update(): %v", err)
		}

		// update to concurrent rate limit, reset now
		headers = http.Header{
			"X-Rate-Limit-Limit":     []string{"0"},
			"X-Rate-Limit-Remaining": []string{"0"},
			"X-Rate-Limit-Reset":     []string{strconv.FormatInt(time.Now().Unix(), 10)},
		}
		err = r.Update(endpoint, headers, logp.L())
		if err != nil {
			t.Errorf("unexpected error from Update(): %v", err)
		}

		// Wait to make the new rate become active
		err = r.Wait(ctx, endpoint, url, log)
		if err != nil {
			t.Errorf("unexpected error from Wait(): %v", err)
		}

		e := r.endpoint(endpoint)

		newLimit := e.limiter.Limit()
		expectedNewLimit := rate.Limit(1)
		if newLimit != expectedNewLimit {
			t.Errorf("expected rate %f, but got %f, after exceeding the concurrent rate limit", expectedNewLimit, newLimit)
		}
	})
}
