// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httpmon"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

const (
	siemAPIPath = "/siem/v1/configs/"
)

// APIError represents an error response from the Akamai API.
type APIError struct {
	StatusCode int
	Status     string
	Detail     string
	Body       string
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("akamai API error: %s (%d): %s", e.Status, e.StatusCode, e.Detail)
	}
	return fmt.Sprintf("akamai API error: %s (%d)", e.Status, e.StatusCode)
}

// IsInvalidTimestamp returns true if the error indicates an invalid timestamp.
func (e *APIError) IsInvalidTimestamp() bool {
	return e.StatusCode == 400 && strings.Contains(strings.ToLower(e.Detail), "invalid timestamp")
}

// IsOffsetOutOfRange returns true if the error indicates the offset is out of range.
func (e *APIError) IsOffsetOutOfRange() bool {
	return e.StatusCode == 416
}

// SIEMEvent represents a single event from the Akamai SIEM API.
type SIEMEvent struct {
	Offset string          `json:"offset,omitempty"`
	Raw    json.RawMessage `json:"-"`
}

// SIEMResponse represents the response from fetching events.
type SIEMResponse struct {
	Events     []SIEMEvent
	LastOffset string
	HasMore    bool
}

// FetchParams contains parameters for fetching events.
type FetchParams struct {
	// Offset is the offset to continue from (mutually exclusive with From/To)
	Offset string
	// From is the start timestamp in Unix seconds (mutually exclusive with Offset)
	From int64
	// To is the end timestamp in Unix seconds (mutually exclusive with Offset)
	To int64
	// Limit is the maximum number of events to fetch
	Limit int
}

// Client is an Akamai SIEM API client.
type Client struct {
	httpClient *http.Client
	signer     *EdgeGridSigner
	baseURL    *url.URL
	configIDs  string
	log        *logp.Logger
	metrics    *inputMetrics
}

// ClientOption is a functional option for configuring the client.
type ClientOption func(*Client)

// WithMetrics sets the metrics for the client.
func WithMetrics(m *inputMetrics) ClientOption {
	return func(c *Client) {
		c.metrics = m
	}
}

// NewClient creates a new Akamai SIEM API client.
func NewClient(cfg config, log *logp.Logger, reg *monitoring.Registry, opts ...ClientOption) (*Client, error) {
	// Create the base HTTP client
	transport := cfg.Resource.Transport
	httpClient, err := transport.Client(
		httpcommon.WithLogger(log),
		httpcommon.WithAPMHTTPInstrumentation(),
		cfg.Resource.KeepAlive.settings(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create the EdgeGrid signer
	signer := NewEdgeGridSigner(
		cfg.getClientToken(),
		cfg.getClientSecret(),
		cfg.getAccessToken(),
	)

	// Wrap transport with EdgeGrid authentication
	httpClient.Transport = &EdgeGridTransport{
		Transport: httpClient.Transport,
		Signer:    signer,
	}

	// Add metrics monitoring
	if reg != nil {
		httpClient.Transport = httpmon.NewMetricsRoundTripper(httpClient.Transport, reg, log)
	}

	// Configure retries
	if cfg.Resource.Retry.getMaxAttempts() > 1 {
		retryClient := &retryablehttp.Client{
			HTTPClient:   httpClient,
			Logger:       newRetryLogger(log),
			RetryWaitMin: cfg.Resource.Retry.getWaitMin(),
			RetryWaitMax: cfg.Resource.Retry.getWaitMax(),
			RetryMax:     cfg.Resource.Retry.getMaxAttempts(),
			CheckRetry:   retryablehttp.DefaultRetryPolicy,
			Backoff:      retryablehttp.DefaultBackoff,
		}
		httpClient = retryClient.StandardClient()
	}

	client := &Client{
		httpClient: httpClient,
		signer:     signer,
		baseURL:    cfg.APIHost.URL,
		configIDs:  cfg.ConfigIDs,
		log:        log.Named("client"),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// FetchEvents fetches events from the Akamai SIEM API.
func (c *Client) FetchEvents(ctx context.Context, params FetchParams) (*SIEMResponse, error) {
	reqURL := c.buildRequestURL(params)
	c.log.Debugw("fetching events", "url", reqURL)

	if c.metrics != nil {
		c.metrics.AddRequest()
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		if c.metrics != nil {
			c.metrics.AddRequestError()
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.metrics != nil {
			c.metrics.AddRequestError()
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if c.metrics != nil {
		c.metrics.RecordResponseLatency(time.Since(start))
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(body),
		}

		// Try to parse error detail
		if len(body) > 0 {
			var errResp struct {
				Detail string `json:"detail"`
			}
			if json.Unmarshal(body, &errResp) == nil {
				apiErr.Detail = errResp.Detail
			}
		}

		if c.metrics != nil {
			c.metrics.AddRequestError()
		}
		return nil, apiErr
	}

	if c.metrics != nil {
		c.metrics.AddRequestSuccess()
	}

	// Parse the response (newline-delimited JSON)
	response, err := c.parseResponse(resp.Body, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if c.metrics != nil {
		c.metrics.RecordRequestTime(time.Since(start))
	}

	return response, nil
}

// buildRequestURL constructs the request URL with query parameters.
func (c *Client) buildRequestURL(params FetchParams) string {
	u := *c.baseURL
	u.Path = siemAPIPath + c.configIDs

	query := url.Values{}
	query.Set("limit", strconv.Itoa(params.Limit))

	if params.Offset != "" {
		query.Set("offset", params.Offset)
	} else {
		if params.From > 0 {
			query.Set("from", strconv.FormatInt(params.From, 10))
		}
		if params.To > 0 {
			query.Set("to", strconv.FormatInt(params.To, 10))
		}
	}

	u.RawQuery = query.Encode()
	return u.String()
}

// parseResponse parses the newline-delimited JSON response.
func (c *Client) parseResponse(body io.Reader, limit int) (*SIEMResponse, error) {
	response := &SIEMResponse{
		Events: make([]SIEMEvent, 0),
	}

	scanner := bufio.NewScanner(body)
	// Increase buffer size for potentially large events
	const maxTokenSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxTokenSize)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse the event to extract offset
		var event SIEMEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			c.log.Warnw("failed to parse event JSON", "error", err, "line", truncate(line, 200))
			continue
		}

		// Store the raw JSON
		event.Raw = json.RawMessage(line)
		response.Events = append(response.Events, event)

		// Track the last offset
		if event.Offset != "" {
			response.LastOffset = event.Offset
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Determine if there might be more events
	response.HasMore = len(response.Events) >= limit

	return response, nil
}

// Close closes the client.
func (c *Client) Close() error {
	// HTTP clients don't need explicit closing, but we might add cleanup later
	return nil
}

// truncate truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// retryLogger implements the retryablehttp.LeveledLogger interface.
type retryLogger struct {
	log *logp.Logger
}

func newRetryLogger(log *logp.Logger) *retryLogger {
	return &retryLogger{log: log.Named("retry")}
}

func (l *retryLogger) Error(msg string, keysAndValues ...interface{}) {
	l.log.Errorw(msg, keysAndValues...)
}

func (l *retryLogger) Info(msg string, keysAndValues ...interface{}) {
	l.log.Infow(msg, keysAndValues...)
}

func (l *retryLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.log.Debugw(msg, keysAndValues...)
}

func (l *retryLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.log.Warnw(msg, keysAndValues...)
}

// IsRecoverableError returns true if the error should trigger recovery mode.
func IsRecoverableError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsInvalidTimestamp() || apiErr.IsOffsetOutOfRange()
	}
	return false
}
