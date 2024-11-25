// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import (
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/elastic-agent-libs/logp"
)

// RLHTTPClient implements a rate-limited HTTP client that wraps the standard http.Client
// with a rate limiter to control API request frequency.
type RLHTTPClient struct {
	client      *http.Client
	logger      *logp.Logger
	Ratelimiter *rate.Limiter
}

// Do executes an HTTP request while respecting rate limits.
// It waits for rate limit token before proceeding with the request.
// Returns the HTTP response and any error encountered.
func (c *RLHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.logger.Debug("Waiting for rate limit token")
	start := time.Now()
	err := c.Ratelimiter.Wait(req.Context())
	waitDuration := time.Since(start)
	if err != nil {
		return nil, err
	}
	c.logger.Debug("Rate limit token acquired")
	if waitDuration > time.Minute {
		c.logger.Infof("Rate limit wait exceeded threshold: %v", waitDuration)
	}
	return c.client.Do(req)
}

// newClient creates a new rate-limited HTTP client with specified rate limiter and timeout.
func newClient(logger *logp.Logger, rl *rate.Limiter, timeout time.Duration) *RLHTTPClient {
	var client = http.DefaultClient
	client.Timeout = timeout
	return &RLHTTPClient{
		client:      client,
		logger:      logger,
		Ratelimiter: rl,
	}
}
