// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

// Test getRateLimit function with a remaining quota, expect to receive 0, nil.
func TestGetRateLimitCase1(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "118")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	rateLimit := &rateLimiter{
		limit:     "X-Rate-Limit-Limit",
		reset:     "X-Rate-Limit-Reset",
		remaining: "X-Rate-Limit-Remaining",
	}
	epoch, err := rateLimit.getRateLimit(header)
	if err != nil || epoch != 0 {
		t.Fatal("Failed to test getRateLimit.")
	}
}

// Test getRateLimit function with a past time, expect to receive 0, nil.
func TestGetRateLimitCase2(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	rateLimit := &rateLimiter{
		limit:     "X-Rate-Limit-Limit",
		reset:     "X-Rate-Limit-Reset",
		remaining: "X-Rate-Limit-Remaining",
	}
	epoch, err := rateLimit.getRateLimit(header)
	if err != nil || epoch != 0 {
		t.Fatal("Failed to test getRateLimit.")
	}
}

// Test getRateLimit function with a time yet to come, expect to receive <reset-value>, nil.
func TestGetRateLimitCase3(t *testing.T) {
	epoch := time.Now().Unix() + 100
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(epoch, 10))
	rateLimit := &rateLimiter{
		limit:     "X-Rate-Limit-Limit",
		reset:     "X-Rate-Limit-Reset",
		remaining: "X-Rate-Limit-Remaining",
	}
	epoch2, err := rateLimit.getRateLimit(header)
	if err != nil || epoch2 != epoch {
		t.Fatal("Failed to test getRateLimit.")
	}
}
