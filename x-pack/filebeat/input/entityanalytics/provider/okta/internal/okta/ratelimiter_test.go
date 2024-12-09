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

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestRateLimiter(t *testing.T) {
	logp.TestingSetup()

	t.Run("separation by endpoint", func(t *testing.T) {
		r := NewRateLimiter()
		limiter1 := r.limiter("/foo")
		limiter2 := r.limiter("/bar")

		limiter1.SetBurst(1000)

		if limiter2.Burst() == 1000 {
			t.Errorf("changes to one endpoint's limits affected another")
		}
	})

	t.Run("Update stops requests when none are remaining", func(t *testing.T) {
		r := NewRateLimiter()

		const endpoint = "/foo"
		limiter := r.limiter(endpoint)

		if !limiter.Allow() {
			t.Errorf("doesn't allow an initial request")
		}

		now := time.Now().Unix()
		reset := now + 30

		headers := http.Header{
			"X-Rate-Limit-Limit":     []string{"60"},
			"X-Rate-Limit-Remaining": []string{"0"},
			"X-Rate-Limit-Reset":     []string{strconv.FormatInt(reset, 10)},
		}
		window := time.Minute

		err := r.Update(endpoint, headers, window, logp.L())
		if err != nil {
			t.Errorf("unexpected error from Update(): %v", err)
		}
		limiter = r.limiter(endpoint)

		if limiter.Allow() {
			t.Errorf("allowed a request when none are remaining")
		}

		if limiter.AllowN(time.Unix(reset-1, 999999999), 1) {
			t.Errorf("allowed a request before reset, when none are remaining")
		}

		if !limiter.AllowN(time.Unix(reset+1, 0), 1) {
			t.Errorf("doesn't allow requests to resume after reset")
		}

		if limiter.Limit() != 1.0 {
			t.Errorf("unexpected rate following reset (not 60 requests / 60 seconds): %f", limiter.Limit())
		}

		if limiter.Burst() != 1 {
			t.Errorf("unexpected burst following reset (not 1): %d", limiter.Burst())
		}

		limiter.SetBurstAt(time.Unix(reset, 0), 100) // increase bucket size to check token accumulation
		tokens := limiter.TokensAt(time.Unix(reset+30, 0))
		if tokens < 29.5 || tokens > 30.0 {
			t.Errorf("tokens don't accumulate at the expected rate. tokens 30s after reset: %f", tokens)
		}

	})

	t.Run("Very long waits are considered errors", func(t *testing.T) {
		r := NewRateLimiter()

		const endpoint = "/foo"

		url, _ := url.Parse(endpoint)
		reset := time.Now().Add(31 * time.Minute).Unix()
		headers := http.Header{
			"X-Rate-Limit-Limit":     []string{"60"},
			"X-Rate-Limit-Remaining": []string{"1"},
			"X-Rate-Limit-Reset":     []string{strconv.FormatInt(reset, 10)},
		}
		window := time.Minute
		log := logp.L()
		ctx := context.Background()

		r.Wait(ctx, endpoint, url, log)          // consume the initial request
		r.Update(endpoint, headers, window, log) // update to a slow rate

		err := r.Wait(ctx, endpoint, url, log)

		const expectedErr = "rate: Wait(n=1) would exceed context deadline"
		if err == nil {
			t.Errorf("expected error message %q, but got no error", expectedErr)
		} else if err.Error() != expectedErr {
			t.Errorf("expected error message %q, but got %q", expectedErr, err.Error())
		}
	})
}
