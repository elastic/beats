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

	"github.com/elastic/beats/v7/libbeat/logp"
)

type rateLimiter struct {
	log *logp.Logger

	limit      *valueTpl
	reset      *valueTpl
	remaining  *valueTpl
	earlyLimit *float64
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

		if err != nil {
			return nil, fmt.Errorf("failed to read http.response.body: %w", err)
		}

		if r == nil || resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return nil, fmt.Errorf("http request was unsuccessful with a status code %d", resp.StatusCode)
		}

		if err := r.applyRateLimit(ctx, resp); err != nil {
			return nil, err
		}
	}
}

// applyRateLimit applies appropriate rate limit if specified in the HTTP Header of the response
func (r *rateLimiter) applyRateLimit(ctx context.Context, resp *http.Response) error {
	epoch, err := r.getRateLimit(resp)
	if err != nil {
		return err
	}

	t := time.Unix(epoch, 0)
	w := time.Until(t)
	if epoch == 0 || w <= 0 {
		r.log.Debugf("Rate Limit: No need to apply rate limit.")
		return nil
	}
	r.log.Debugf("Rate Limit: Wait until %v for the rate limit to reset.", t)
	ticker := time.NewTicker(w)
	defer ticker.Stop()

	select {
	case <-ctx.Done():
		r.log.Info("Context done.")
		return nil
	case <-ticker.C:
		r.log.Debug("Rate Limit: time is up.")
		return nil
	}
}

// getRateLimit gets the rate limit value if specified in the response,
// and returns an int64 value in seconds since unix epoch for rate limit reset time.
// When there is a remaining rate limit quota, or when the rate limit reset time has expired, it
// returns 0 for the epoch value.
func (r *rateLimiter) getRateLimit(resp *http.Response) (int64, error) {
	if r == nil {
		return 0, nil
	}

	if r.remaining == nil {
		return 0, nil
	}

	tr := transformable{}
	ctx := emptyTransformContext()
	ctx.updateLastResponse(response{header: resp.Header.Clone()})

	remaining, _ := r.remaining.Execute(ctx, tr, nil, r.log)
	if remaining == "" {
		return 0, errors.New("remaining value is empty")
	}
	m, err := strconv.ParseInt(remaining, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate-limit remaining value: %w", err)
	}

	// by default, httpjson will continue requests until Limit is 0
	// can optionally stop requests "early"
	var activeLimit int64 = 0
	if r.earlyLimit != nil {
		earlyLimit := *r.earlyLimit
		if earlyLimit > 0 && earlyLimit < 1 {
			limit, _ := r.limit.Execute(ctx, tr, nil, r.log)
			if limit != "" {
				l, err := strconv.ParseInt(limit, 10, 64)
				if err == nil {
					activeLimit = l - int64(earlyLimit*float64(l))
				}
			}
		} else if earlyLimit >= 1 {
			activeLimit = int64(earlyLimit)
		}
	}

	r.log.Debugf("Rate Limit: Using active Early Limit: %f", activeLimit)
	if m > activeLimit {
		return 0, nil
	}

	if r.reset == nil {
		r.log.Warn("reset rate limit is not set")
		return 0, nil
	}

	reset, _ := r.reset.Execute(ctx, tr, nil, r.log)
	if reset == "" {
		return 0, errors.New("reset value is empty")
	}

	epoch, err := strconv.ParseInt(reset, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate-limit reset value: %w", err)
	}

	if timeNow().Unix() > epoch {
		return 0, nil
	}

	return epoch, nil
}
