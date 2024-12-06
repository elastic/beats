// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Test getRateLimit function with a remaining quota, expect to receive 0, nil.
func TestGetRateLimitReturnsFalse0IfRemainingQuota(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "118")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	assert.NoError(t, tplLimit.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Limit"]]`))
	assert.NoError(t, tplReset.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Reset"]]`))
	assert.NoError(t, tplRemaining.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Remaining"]]`))
	rateLimit := &rateLimiter{
		limit:     tplLimit,
		reset:     tplReset,
		remaining: tplRemaining,
		log:       logp.NewLogger(""),
	}
	resp := &http.Response{Header: header}
	applied, resumeAt, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.False(t, applied)
	assert.EqualValues(t, 0, resumeAt)
}

func TestGetRateLimitReturnsTrue0IfResumeAtInPast(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	assert.NoError(t, tplLimit.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Limit"]]`))
	assert.NoError(t, tplReset.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Reset"]]`))
	assert.NoError(t, tplRemaining.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Remaining"]]`))
	rateLimit := &rateLimiter{
		limit:     tplLimit,
		reset:     tplReset,
		remaining: tplRemaining,
		log:       logp.NewLogger(""),
	}
	resp := &http.Response{Header: header}
	applied, resumeAt, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.True(t, applied)
	assert.EqualValues(t, 0, resumeAt)
}

func TestGetRateLimitReturnsResetValue(t *testing.T) {
	reset := int64(1604582732 + 100)
	timeNow = func() time.Time { return time.Unix(1604582732, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(reset, 10))
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	assert.NoError(t, tplLimit.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Limit"]]`))
	assert.NoError(t, tplReset.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Reset"]]`))
	assert.NoError(t, tplRemaining.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Remaining"]]`))
	rateLimit := &rateLimiter{
		limit:     tplLimit,
		reset:     tplReset,
		remaining: tplRemaining,
		log:       logp.NewLogger(""),
	}
	resp := &http.Response{Header: header}
	applied, resumeAt, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.True(t, applied)
	assert.EqualValues(t, reset, resumeAt)
}

// Test getRateLimit function with a remaining quota, using default early limit
// expect to receive false, 0, nil.
func TestGetRateLimitReturns0IfEarlyLimit0(t *testing.T) {
	resetAt := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "1")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(resetAt, 10))
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	earlyLimit := func(i float64) *float64 { return &i }(0)
	assert.NoError(t, tplLimit.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Limit"]]`))
	assert.NoError(t, tplReset.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Reset"]]`))
	assert.NoError(t, tplRemaining.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Remaining"]]`))
	rateLimit := &rateLimiter{
		limit:      tplLimit,
		reset:      tplReset,
		remaining:  tplRemaining,
		log:        logp.NewLogger("TestGetRateLimitReturns0IfEarlyLimit0"),
		earlyLimit: earlyLimit,
	}
	resp := &http.Response{Header: header}
	applied, resumeAt, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.False(t, applied)
	assert.EqualValues(t, 0, resumeAt)
}

// Test getRateLimit function with a remaining limit, but early limit
// expect to receive true, Reset Time
func TestGetRateLimitReturnsResetValueIfEarlyLimit1(t *testing.T) {
	resetAt := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "1")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(resetAt, 10))
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	earlyLimit := func(i float64) *float64 { return &i }(1)
	assert.NoError(t, tplLimit.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Limit"]]`))
	assert.NoError(t, tplReset.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Reset"]]`))
	assert.NoError(t, tplRemaining.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Remaining"]]`))
	rateLimit := &rateLimiter{
		limit:      tplLimit,
		reset:      tplReset,
		remaining:  tplRemaining,
		log:        logp.NewLogger("TestGetRateLimitReturnsResetValueIfEarlyLimit1"),
		earlyLimit: earlyLimit,
	}
	resp := &http.Response{Header: header}
	applied, resumeAt, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.True(t, applied)
	assert.EqualValues(t, resetAt, resumeAt)
}

// Test getRateLimit function with a remaining quota, using 90% early limit
// expect to receive false, 0, nil.
func TestGetRateLimitReturns0IfEarlyLimitPercent(t *testing.T) {
	resetAt := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "13")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(resetAt, 10))
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	earlyLimit := func(i float64) *float64 { return &i }(0.9)
	assert.NoError(t, tplLimit.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Limit"]]`))
	assert.NoError(t, tplReset.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Reset"]]`))
	assert.NoError(t, tplRemaining.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Remaining"]]`))
	rateLimit := &rateLimiter{
		limit:      tplLimit,
		reset:      tplReset,
		remaining:  tplRemaining,
		log:        logp.NewLogger("TestGetRateLimitReturns0IfEarlyLimitPercent"),
		earlyLimit: earlyLimit,
	}
	resp := &http.Response{Header: header}
	applied, resumeAt, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.False(t, applied)
	assert.EqualValues(t, 0, resumeAt)
}

// Test getRateLimit function with a remaining limit, but early limit of 90%
// expect to receive true, Reset Time
func TestGetRateLimitReturnsResetValueIfEarlyLimitPercent(t *testing.T) {
	resetAt := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "12")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(resetAt, 10))
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	earlyLimit := func(i float64) *float64 { return &i }(0.9)
	assert.NoError(t, tplLimit.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Limit"]]`))
	assert.NoError(t, tplReset.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Reset"]]`))
	assert.NoError(t, tplRemaining.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Remaining"]]`))
	rateLimit := &rateLimiter{
		limit:      tplLimit,
		reset:      tplReset,
		remaining:  tplRemaining,
		log:        logp.NewLogger("TestGetRateLimitReturnsResetValueIfEarlyLimitPercent"),
		earlyLimit: earlyLimit,
	}
	resp := &http.Response{Header: header}
	applied, resumeAt, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.True(t, applied)
	assert.EqualValues(t, resetAt, resumeAt)
}

// Test getRateLimit function when "Limit" header is missing, when using a Percentage early-limit
// expect to receive false, 0, nil. (default rate-limiting)
func TestGetRateLimitWhenMissingLimit(t *testing.T) {
	reset := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Remaining", "1")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(reset, 10))
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	earlyLimit := func(i float64) *float64 { return &i }(0.9)
	assert.NoError(t, tplReset.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Reset"]]`))
	assert.NoError(t, tplRemaining.Unpack(`[[.last_response.header.Get "X-Rate-Limit-Remaining"]]`))
	rateLimit := &rateLimiter{
		limit:      nil,
		reset:      tplReset,
		remaining:  tplRemaining,
		log:        logp.NewLogger("TestGetRateLimitWhenMissingLimit"),
		earlyLimit: earlyLimit,
	}
	resp := &http.Response{Header: header}
	applied, resumeAt, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.False(t, applied)
	assert.EqualValues(t, 0, resumeAt)
}
