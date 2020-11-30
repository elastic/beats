// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// Test getRateLimit function with a remaining quota, expect to receive 0, nil.
func TestGetRateLimitReturns0IfRemainingQuota(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "118")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	assert.NoError(t, tplLimit.Unpack(`{{.header.Get "X-Rate-Limit-Limit"}}`))
	assert.NoError(t, tplReset.Unpack(`{{.header.Get "X-Rate-Limit-Reset"}}`))
	assert.NoError(t, tplRemaining.Unpack(`{{.header.Get "X-Rate-Limit-Remaining"}}`))
	rateLimit := &rateLimiter{
		limit:     tplLimit,
		reset:     tplReset,
		remaining: tplRemaining,
		log:       logp.NewLogger(""),
	}
	resp := &http.Response{Header: header}
	epoch, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, epoch)
}

func TestGetRateLimitReturns0IfEpochInPast(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	assert.NoError(t, tplLimit.Unpack(`{{.header.Get "X-Rate-Limit-Limit"}}`))
	assert.NoError(t, tplReset.Unpack(`{{.header.Get "X-Rate-Limit-Reset"}}`))
	assert.NoError(t, tplRemaining.Unpack(`{{.header.Get "X-Rate-Limit-Remaining"}}`))
	rateLimit := &rateLimiter{
		limit:     tplLimit,
		reset:     tplReset,
		remaining: tplRemaining,
		log:       logp.NewLogger(""),
	}
	resp := &http.Response{Header: header}
	epoch, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, epoch)
}

func TestGetRateLimitReturnsResetValue(t *testing.T) {
	epoch := int64(1604582732 + 100)
	timeNow = func() time.Time { return time.Unix(1604582732, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(epoch, 10))
	tplLimit := &valueTpl{}
	tplReset := &valueTpl{}
	tplRemaining := &valueTpl{}
	assert.NoError(t, tplLimit.Unpack(`{{.header.Get "X-Rate-Limit-Limit"}}`))
	assert.NoError(t, tplReset.Unpack(`{{.header.Get "X-Rate-Limit-Reset"}}`))
	assert.NoError(t, tplRemaining.Unpack(`{{.header.Get "X-Rate-Limit-Remaining"}}`))
	rateLimit := &rateLimiter{
		limit:     tplLimit,
		reset:     tplReset,
		remaining: tplRemaining,
		log:       logp.NewLogger(""),
	}
	resp := &http.Response{Header: header}
	epoch2, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.EqualValues(t, 1604582832, epoch2)
}
