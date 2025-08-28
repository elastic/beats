// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/elastic-agent-libs/logp"
)

// RLHTTPClient wraps the standard http.Client with a rate limiter to control API request frequency.
type RLHTTPClient struct {
	client      *http.Client
	logger      *logp.Logger
	Ratelimiter *rate.Limiter
}

// Do executes an HTTP request while respecting rate limits.
// It waits for rate limit token before proceeding with the request.
// Returns the HTTP response and any error encountered.
func (c *RLHTTPClient) Do(req *http.Request) (*http.Response, error) {
	start := time.Now()

	c.logger.Debug("Waiting for rate limit token")

	err := c.Ratelimiter.Wait(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to acquire rate limit token: %w", err)
	}

	c.logger.Debug("Rate limit token acquired")

	waitDuration := time.Since(start)

	if waitDuration > time.Minute {
		c.logger.Infof("Rate limit wait exceeded threshold: %v", waitDuration)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// newClient creates a new rate-limited HTTP client with specified rate limiter and timeout.
func newClient(logger *logp.Logger, rl *rate.Limiter, timeout time.Duration) *RLHTTPClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return &RLHTTPClient{
		client:      client,
		logger:      logger,
		Ratelimiter: rl,
	}
}
