// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

type rateLimiter struct {
	limit      *valueTpl
	reset      *valueTpl
	remaining  *valueTpl
	earlyLimit *float64

	log *logp.Logger
}

func newRateLimiterFromConfig(config *rateLimitConfig, log *logp.Logger) *rateLimiter {
	if config == nil {
		return nil
	}

	return &rateLimiter{
		log:        log,
		limit:      config.Limit,
		reset:      config.Reset,
		remaining:  config.Remaining,
		earlyLimit: config.EarlyLimit,
	}
}

func (r *rateLimiter) execute(ctx context.Context, f func() (*http.Response, error)) (*http.Response, error) {
	for {
		resp, err := f()
		if err != nil {
			return nil, err
		}

		if r == nil {
			return resp, nil
		}

		applied, err := r.applyRateLimit(ctx, resp)
		if err != nil {
			return nil, fmt.Errorf("error applying rate limit: %w", err)
		}

		if resp.StatusCode == http.StatusOK || !applied {
			return resp, nil
		}
	}
}

// applyRateLimit applies appropriate rate limit if specified in the HTTP Header of the response.
// It returns a bool indicating whether a limit was reached.
func (r *rateLimiter) applyRateLimit(ctx context.Context, resp *http.Response) (bool, error) {
	limitReached, resumeAt, err := r.getRateLimit(resp)
	if err != nil {
		return limitReached, err
	}

	t := time.Unix(resumeAt, 0)
	w := time.Until(t)
	if resumeAt == 0 || w <= 0 {
		r.log.Debugf("Rate Limit: No need to apply rate limit.")
		return limitReached, nil
	}
	r.log.Debugf("Rate Limit: Wait until %v for the rate limit to reset.", t)
	timer := time.NewTimer(w)
	select {
	case <-ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
		r.log.Info("Context done.")
		return limitReached, nil
	case <-timer.C:
		r.log.Debug("Rate Limit: time is up.")
		return limitReached, nil
	}
}

// getRateLimit gets the rate limit value if specified in the response,
// and returns a bool indicating whether a limit was reached, and
// an int64 value in seconds since unix epoch for rate limit reset time.
// When there is a remaining rate limit quota, or when the rate limit reset time has expired, it
// returns 0 for the epoch value.
func (r *rateLimiter) getRateLimit(resp *http.Response) (bool, int64, error) {
	if r == nil {
		return false, 0, nil
	}

	if r.remaining == nil {
		return false, 0, nil
	}

	tr := transformable{}
	ctx := emptyTransformContext()
	ctx.updateLastResponse(response{header: resp.Header.Clone()})

	remaining, _ := r.remaining.Execute(ctx, tr, "rate-limit_remaining", nil, r.log)
	if remaining == "" {
		return false, 0, errors.New("remaining value is empty")
	}
	m, err := strconv.ParseInt(remaining, 10, 64)
	if err != nil {
		return false, 0, fmt.Errorf("failed to parse rate-limit remaining value: %w", err)
	}

	// by default, httpjson will continue requests until Limit is 0
	// can optionally stop requests "early"
	var minRemaining int64 = 0
	if r.earlyLimit != nil {
		earlyLimit := *r.earlyLimit
		if earlyLimit > 0 && earlyLimit < 1 {
			limit, _ := r.limit.Execute(ctx, tr, "early_limit", nil, r.log)
			if limit != "" {
				l, err := strconv.ParseInt(limit, 10, 64)
				if err == nil {
					minRemaining = l - int64(earlyLimit*float64(l))
				}
			}
		} else if earlyLimit >= 1 {
			minRemaining = int64(earlyLimit)
		}
	}

	r.log.Debugf("Rate Limit: Using active Early Limit: %f", minRemaining)
	if m > minRemaining {
		return false, 0, nil
	}

	if r.reset == nil {
		r.log.Warn("reset rate limit is not set")
		return false, 0, nil
	}

	reset, _ := r.reset.Execute(ctx, tr, "rate-limit_reset", nil, r.log)
	if reset == "" {
		return false, 0, errors.New("reset value is empty")
	}

	resumeAt, err := strconv.ParseInt(reset, 10, 64)
	if err != nil {
		return false, 0, fmt.Errorf("failed to parse rate-limit reset value: %w", err)
	}

	if timeNow().Unix() > resumeAt {
		return true, 0, nil
	}

	return true, resumeAt, nil
}
