// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
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

// IsFromTooOld returns true if the error indicates the from parameter is too
// old (beyond the Akamai max lookback window).
func (e *APIError) IsFromTooOld() bool {
	if e.StatusCode != 400 {
		return false
	}
	lower := strings.ToLower(e.Detail)
	return strings.Contains(lower, "out of range") || strings.Contains(lower, "too old")
}

// offsetContext is the final context object returned by the SIEM API response.
// It contains pagination metadata and should not be emitted as an event.
type offsetContext struct {
	Offset string `json:"offset"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
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
	limiter    *rate.Limiter
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
	transport := cfg.Resource.Transport
	httpClient, err := transport.Client(
		httpcommon.WithLogger(log),
		httpcommon.WithAPMHTTPInstrumentation(),
		cfg.Resource.KeepAlive.settings(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	signer := NewEdgeGridSigner(
		cfg.Auth.EdgeGrid.ClientToken,
		cfg.Auth.EdgeGrid.ClientSecret,
		cfg.Auth.EdgeGrid.AccessToken,
	)

	httpClient.Transport = &EdgeGridTransport{
		Transport: httpClient.Transport,
		Signer:    signer,
	}

	if cfg.Tracer != nil && cfg.Tracer.enabled() {
		w := zapcore.AddSync(cfg.Tracer)
		core := ecszap.NewCore(
			ecszap.NewDefaultEncoderConfig(),
			w,
			zap.DebugLevel,
		)
		traceLogger := zap.New(core)
		maxBodyLen := cfg.Tracer.MaxSize * 1e6 / 10
		if maxBodyLen <= 0 {
			maxBodyLen = 100000
		}
		httpClient.Transport = httplog.NewLoggingRoundTripper(httpClient.Transport, traceLogger, maxBodyLen, log)
	}

	if reg != nil {
		httpClient.Transport = httpmon.NewMetricsRoundTripper(httpClient.Transport, reg, log)
	}

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

	var limiter *rate.Limiter
	if rl := cfg.Resource.RateLimit; rl != nil && rl.Limit != nil {
		burst := 1
		if rl.Burst != nil {
			burst = *rl.Burst
		}
		limiter = rate.NewLimiter(rate.Limit(*rl.Limit), burst)
	}

	client := &Client{
		httpClient: httpClient,
		signer:     signer,
		baseURL:    cfg.Resource.URL.URL,
		configIDs:  cfg.ConfigIDs,
		log:        log.Named("client"),
		limiter:    limiter,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// FetchResponse makes the HTTP request and returns the response body for
// streaming consumption. The caller must close the returned body. On error
// (non-200 status), the body is consumed internally and an *APIError is
// returned.
func (c *Client) FetchResponse(ctx context.Context, params FetchParams) (io.ReadCloser, error) {
	reqURL := c.buildRequestURL(params)
	c.log.Debugw("fetching events",
		"host", c.baseURL.Host,
		"path", siemAPIPath+c.configIDs,
		"mode", fetchMode(params),
		"limit", params.Limit,
		"offset", params.Offset,
		"from", params.From,
		"to", params.To,
	)

	if c.metrics != nil {
		c.metrics.AddRequest()
	}

	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter wait: %w", err)
		}
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

	if c.metrics != nil {
		c.metrics.RecordResponseLatency(time.Since(start))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(body),
		}

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

	return resp.Body, nil
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

// StreamEvents reads NDJSON lines from body, pushing event lines into eventCh
// using a one-line delay. The last line is treated as the offset context and
// unmarshalled rather than sent as an event. Returns the offset context, the
// number of events sent, and any scanner error.
//
// The caller must close eventCh after StreamEvents returns.
func StreamEvents(ctx context.Context, body io.Reader, eventCh chan<- json.RawMessage) (offsetContext, int, error) {
	scanner := bufio.NewScanner(body)
	const maxTokenSize = 10 * 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxTokenSize)

	var prev []byte
	count := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		current := make([]byte, len(line))
		copy(current, line)

		if prev != nil {
			select {
			case eventCh <- json.RawMessage(prev):
				count++
			case <-ctx.Done():
				return offsetContext{}, count, ctx.Err()
			}
		}
		prev = current
	}

	if err := scanner.Err(); err != nil {
		return offsetContext{}, count, fmt.Errorf("error reading response: %w", err)
	}

	if prev == nil {
		return offsetContext{}, 0, nil
	}

	// Last line: try to unmarshal as offset context. A valid offset context
	// always contains a limit field (> 0), which distinguishes it from regular
	// events that happen to have an "offset" key.
	var pageCtx offsetContext
	if err := json.Unmarshal(prev, &pageCtx); err == nil && pageCtx.Offset != "" && pageCtx.Limit > 0 {
		return pageCtx, count, nil
	}

	// Last line was a regular event, not offset context.
	select {
	case eventCh <- json.RawMessage(prev):
		count++
	case <-ctx.Done():
		return offsetContext{}, count, ctx.Err()
	}
	return offsetContext{}, count, nil
}

// Close releases resources held by the client.
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
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

func fetchMode(params FetchParams) string {
	if params.Offset != "" {
		return "offset"
	}
	return "time"
}
