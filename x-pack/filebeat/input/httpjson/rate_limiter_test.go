// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
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
	epoch2, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.EqualValues(t, 1604582832, epoch2)
}

// Test getRateLimit function with a remaining quota, using default early limit
// expect to receive 0, nil.
func TestGetRateLimitReturns0IfEarlyLimit0(t *testing.T) {
	resetEpoch := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "1")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(resetEpoch, 10))
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
	epoch, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, epoch)
}

// Test getRateLimit function with a remaining limit, but early limit
// expect to receive Reset Time
func TestGetRateLimitReturnsResetValueIfEarlyLimit1(t *testing.T) {
	resetEpoch := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "1")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(resetEpoch, 10))
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
	epoch, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.EqualValues(t, resetEpoch, epoch)
}

// Test getRateLimit function with a remaining quota, using 90% early limit
// expect to receive 0, nil.
func TestGetRateLimitReturns0IfEarlyLimitPercent(t *testing.T) {
	resetEpoch := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "13")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(resetEpoch, 10))
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
	epoch, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, epoch)
}

// Test getRateLimit function with a remaining limit, but early limit of 90%
// expect to receive Reset Time
func TestGetRateLimitReturnsResetValueIfEarlyLimitPercent(t *testing.T) {
	resetEpoch := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "12")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(resetEpoch, 10))
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
	epoch, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.EqualValues(t, resetEpoch, epoch)
}

// Test getRateLimit function when "Limit" header is missing, when using a Percentage early-limit
// expect to receive 0, nil. (default rate-limiting)
func TestGetRateLimitWhenMissingLimit(t *testing.T) {
	resetEpoch := int64(1634579974 + 100)
	timeNow = func() time.Time { return time.Unix(1634579974, 0).UTC() }
	t.Cleanup(func() { timeNow = time.Now })

	header := make(http.Header)
	header.Add("X-Rate-Limit-Remaining", "1")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(resetEpoch, 10))
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
	epoch, err := rateLimit.getRateLimit(resp)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, epoch)
}

// Test ClientLimit functionality, and that the amount of request tokens returns correctly.
func TestClientLimitRequest(t *testing.T) {
	registerRequestTransforms()
	t.Cleanup(func() { registeredTransforms = newRegistry() })

	// test with dateCursorHandler to have different payloads each request
	testServer := httptest.NewServer(clientLimitHandler())
	t.Cleanup(testServer.Close)

	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"interval":       1,
		"request.method": "GET",
		"request.url":    testServer.URL,
		"request.rate_limit.client_limit.interval": 1,
		"request.rate_limit.client_limit.requests": 2,
	})

	config := defaultConfig()
	assert.NoError(t, cfg.Unpack(&config))

	log := logp.NewLogger("")
	ctx := context.Background()
	client, err := newHTTPClient(ctx, config, log)
	assert.NoError(t, err)

	requestFactory, err := newRequestFactory(ctx, config, log)
	assert.NoError(t, err)
	pagination := newPagination(config, client, log)
	responseProcessor := newResponseProcessor(config, pagination, log)

	requester := newRequester(client, requestFactory, responseProcessor, log)
	trCtx := emptyTransformContext()

	var currentTokenCount int
	var lastTokenCount int

	// Making sure that token count = configured amount of requests
	currentTokenCount = int(requester.client.limiter.clientLimiter.Tokens())
	assert.EqualValues(t, 2, currentTokenCount)
	// Request one
	assert.NoError(t, requester.doRequest(ctx, trCtx, statelessPublisher{&beattest.FakeClient{}}))

	// Making sure that the current token count decreased from last request
	lastTokenCount = currentTokenCount
	currentTokenCount = int(requester.client.limiter.clientLimiter.Tokens())
	assert.Greater(t, lastTokenCount, currentTokenCount)

	// Second Request
	assert.NoError(t, requester.doRequest(ctx, trCtx, statelessPublisher{&beattest.FakeClient{}}))

	// Third Request, this should wait since token count should be < 0
	assert.NoError(t, requester.doRequest(ctx, trCtx, statelessPublisher{&beattest.FakeClient{}}))

	//Determine if a new token is available 1 second from now
	assert.GreaterOrEqual(t, int(requester.client.limiter.clientLimiter.TokensAt(time.Now().Add(1*time.Second))), 1)

}
