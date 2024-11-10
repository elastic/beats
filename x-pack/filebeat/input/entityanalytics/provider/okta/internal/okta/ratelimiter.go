// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/elastic-agent-libs/logp"
)

type RateLimiter struct {
	lim *rate.Limiter
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		lim: rate.NewLimiter(1, 1), // Allow a single fetch operation to obtain limits from the API
	}
}

func (r RateLimiter) Wait(ctx context.Context) (err error) {
	return r.lim.Wait(ctx)
}

func (r RateLimiter) Limit() rate.Limit {
	return r.lim.Limit()
}

func (r RateLimiter) Burst() int {
	return r.lim.Burst()
}

// Update implements the Okta rate limit policy translation.
//
// See https://developer.okta.com/docs/reference/rl-best-practices/ for details.
func (r RateLimiter) Update(h http.Header, window time.Duration, log *logp.Logger) error {
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
		waitUntil := resetTime.UTC()
		// next gives us a sane next window estimate, but the
		// estimate will be overwritten when we make the next
		// permissible API request.
		next := rate.Limit(lim / window.Seconds())
		r.lim.SetLimitAt(waitUntil, next)
		r.lim.SetBurstAt(waitUntil, burst)
		log.Debugw("rate limit adjust", "reset_time", waitUntil, "next_rate", next, "next_burst", burst)
		return nil
	}
	r.lim.SetLimit(rateLimit)
	r.lim.SetBurst(burst)
	log.Debugw("rate limit adjust", "set_rate", rateLimit, "set_burst", burst)
	return nil
}
