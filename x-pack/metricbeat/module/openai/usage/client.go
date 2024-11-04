package usage

import (
	"context"
	"net/http"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"golang.org/x/time/rate"
)

// RLHTTPClient implements a rate-limited HTTP client that wraps the standard http.Client
// with a rate limiter to control API request frequency.
type RLHTTPClient struct {
	ctx         context.Context
	client      *http.Client
	logger      *logp.Logger
	Ratelimiter *rate.Limiter
}

// Do executes an HTTP request while respecting rate limits.
// It waits for rate limit token before proceeding with the request.
// Returns the HTTP response and any error encountered.
func (c *RLHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.logger.Warn("Waiting for rate limit token")
	err := c.Ratelimiter.Wait(context.TODO())
	if err != nil {
		return nil, err
	}
	c.logger.Warn("Rate limit token acquired")
	return c.client.Do(req)
}

// newClient creates a new rate-limited HTTP client with specified rate limiter and timeout.
func newClient(ctx context.Context, logger *logp.Logger, rl *rate.Limiter, timeout time.Duration) *RLHTTPClient {
	var client = http.DefaultClient
	client.Timeout = timeout
	return &RLHTTPClient{
		ctx:         ctx,
		client:      client,
		logger:      logger,
		Ratelimiter: rl,
	}
}
