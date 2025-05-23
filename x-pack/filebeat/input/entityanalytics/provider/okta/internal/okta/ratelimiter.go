// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/elastic-agent-libs/logp"
)

// RateLimiter holds rate limiting information for an API.
//
// Each API endpoint has its own rate limit, which can be dynamically updated
// using response headers. If a fixed limit is set, it takes precedence over any
// information from response headers.
type RateLimiter struct {
	window     time.Duration
	fixedLimit *int
	byEndpoint map[string]endpointRateLimiter
}

// endpointRateLimiter represents rate limiting information for a single API endpoint.
type endpointRateLimiter struct {
	limiter *rate.Limiter
	ready   chan struct{}
}

// maxWait defines the maximum wait duration allowed for rate limiting.
// Longer waits are considered errors.
const maxWait = 30 * time.Minute

// NewRateLimiter constructs a new RateLimiter.
//
// Parameters:
//   - `window`: The time between API limit resets. Used for setting an initial
//     target rate.
//   - `fixedLimit`: A fixed number of requests to allow in each `window`,
//     overriding the guidance in API responses.
//
// Returns:
//   - A pointer to a new RateLimiter instance.
func NewRateLimiter(window time.Duration, fixedLimit *int) *RateLimiter {
	endpoints := make(map[string]endpointRateLimiter)
	r := RateLimiter{
		window:     window,
		fixedLimit: fixedLimit,
		byEndpoint: endpoints,
	}
	r.fixedLimit = fixedLimit
	return &r
}

var immediatelyReady = make(chan struct{})

func init() { close(immediatelyReady) }

func (r RateLimiter) endpoint(path string) endpointRateLimiter {
	if existing, ok := r.byEndpoint[path]; ok {
		return existing
	}
	limit := rate.Limit(1)
	if r.fixedLimit != nil {
		limit = rate.Limit(float64(*r.fixedLimit) / r.window.Seconds())
	}
	limiter := rate.NewLimiter(limit, 1) // Allow a single fetch operation to obtain limits from the API
	newEndpointRateLimiter := endpointRateLimiter{
		limiter: limiter,
		ready:   immediatelyReady,
	}
	r.byEndpoint[path] = newEndpointRateLimiter
	return newEndpointRateLimiter
}

func (r RateLimiter) Wait(ctx context.Context, endpoint string, url *url.URL, log *logp.Logger) (err error) {
	e := r.endpoint(endpoint)
	log.Debugw("rate limit", "limit", e.limiter.Limit(), "burst", e.limiter.Burst(), "url", url.String())
	ctxWithDeadline, cancel := context.WithDeadline(ctx, time.Now().Add(maxWait))
	defer cancel()
	select {
	case <-e.ready:
	case <-ctxWithDeadline.Done():
		return ctxWithDeadline.Err()
	}
	return e.limiter.Wait(ctxWithDeadline)
}

// Update implements the Okta rate limit policy translation.
//
// See https://developer.okta.com/docs/reference/rl-best-practices/ for details.
func (r RateLimiter) Update(endpoint string, h http.Header, log *logp.Logger) error {
	if r.fixedLimit != nil {
		return nil
	}
	e := r.endpoint(endpoint)
	limit := h.Get("X-Rate-Limit-Limit")
	remaining := h.Get("X-Rate-Limit-Remaining")
	reset := h.Get("X-Rate-Limit-Reset")
	log.Debugw("rate limit header", "X-Rate-Limit-Limit", limit, "X-Rate-Limit-Remaining", remaining, "X-Rate-Limit-Reset", reset)
	if limit == "" || remaining == "" || reset == "" {
		return nil
	}

	lim, err := strconv.ParseFloat(limit, 64)
	if err != nil {
		return err
	}
	rem, err := strconv.ParseFloat(remaining, 64)
	if err != nil {
		return err
	}
	rst, err := strconv.ParseInt(reset, 10, 64)
	if err != nil {
		return err
	}
	resetTime := time.Unix(rst, 0)
	per := time.Until(resetTime).Seconds()

	// Be conservative here; the docs don't exactly specify burst rates.
	// Make sure we can make at least one new request, even if we fail
	// to get a non-zero rate.Limit. We could set to zero for the case
	// that limit=rate.Inf, but that detail is not important.
	burst := 1

	rateLimit := rate.Limit(rem / per)

	// Process reset if we need to wait until reset to avoid a request against a zero quota.
	if rateLimit <= 0 {
		// Reset limiter to block requests until reset
		limiter := rate.NewLimiter(0, 0)
		ready := make(chan struct{})
		newEndpointRateLimiter := endpointRateLimiter{
			limiter: limiter,
			ready:   ready,
		}
		r.byEndpoint[endpoint] = newEndpointRateLimiter

		// next gives us a sane next window estimate, but the
		// estimate will be overwritten when we make the next
		// permissible API request.
		var next rate.Limit
		if lim == 0 {
			log.Debugw("exceeded the concurrent rate limit")
			next = rate.Limit(1)
		} else {
			next = rate.Limit(lim / r.window.Seconds())
		}

		resetTimeUTC := resetTime.UTC()
		log.Debugw("rate limit block until reset", "reset_time", resetTimeUTC)
		waitFor := time.Until(resetTimeUTC)

		time.AfterFunc(waitFor, func() {
			limiter.SetLimit(next)
			limiter.SetBurst(burst)
			close(ready)
			log.Debugw("rate limit reset", "reset_time", resetTimeUTC, "reset_rate", next, "reset_burst", burst)
		})

		return nil
	}
	e.limiter.SetLimit(rateLimit)
	e.limiter.SetBurst(burst)
	log.Debugw("rate limit adjust", "set_rate", rateLimit, "set_burst", burst)
	return nil
}
