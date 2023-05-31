// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package httplog provides http request and response transaction logging.
package httplog

import (
	"bytes"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"

	"github.com/rcrowley/go-metrics"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

var _ http.RoundTripper = (*LoggingRoundTripper)(nil)
var _ http.RoundTripper = (*MetricsRoundTripper)(nil)

// TraceIDKey is key used to add a trace.id value to the context of HTTP
// requests. The value will be logged by LoggingRoundTripper.
const TraceIDKey = contextKey("trace.id")

type contextKey string

// NewLoggingRoundTripper returns a LoggingRoundTripper that logs requests and
// responses to the provided logger.
func NewLoggingRoundTripper(next http.RoundTripper, logger *zap.Logger) *LoggingRoundTripper {
	return &LoggingRoundTripper{
		transport:   next,
		logger:      logger,
		txBaseID:    newID(),
		txIDCounter: atomic.NewUint64(0),
	}
}

// LoggingRoundTripper is an http.RoundTripper that logs requests and responses.
type LoggingRoundTripper struct {
	transport   http.RoundTripper
	logger      *zap.Logger    // Destination logger.
	txBaseID    string         // Random value to make transaction IDs unique.
	txIDCounter *atomic.Uint64 // Transaction ID counter that is incremented for each request.
}

// RoundTrip implements the http.RoundTripper interface, logging
// the request and response to the underlying logger.
//
// Fields logged in requests:
//
//	url.original
//	url.scheme
//	url.path
//	url.domain
//	url.port
//	url.query
//	http.request
//	user_agent.original
//	http.request.body.content
//	http.request.body.bytes
//	http.request.mime_type
//	event.original (the full request and body from httputil.DumpRequestOut)
//
// Fields logged in responses:
//
//	http.response.status_code
//	http.response.body.content
//	http.response.body.bytes
//	http.response.mime_type
//	event.original (the full response and body from httputil.DumpResponse)
func (rt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a child logger for this request.
	log := rt.logger.With(
		zap.String("transaction.id", rt.nextTxID()),
	)

	if v := req.Context().Value(TraceIDKey); v != nil {
		if traceID, ok := v.(string); ok {
			log = log.With(zap.String("trace.id", traceID))
		}
	}

	reqParts := []zapcore.Field{
		zap.String("url.original", req.URL.String()),
		zap.String("url.scheme", req.URL.Scheme),
		zap.String("url.path", req.URL.Path),
		zap.String("url.domain", req.URL.Hostname()),
		zap.String("url.port", req.URL.Port()),
		zap.String("url.query", req.URL.RawQuery),
		zap.String("http.request.method", req.Method),
		zap.String("user_agent.original", req.Header.Get("User-Agent")),
	}
	var (
		body           []byte
		err            error
		errorsMessages []string
	)
	req.Body, body, err = copyBody(req.Body)
	if err != nil {
		errorsMessages = append(errorsMessages, fmt.Sprintf("failed to read request body: %s", err))
	} else {
		reqParts = append(reqParts,
			zap.ByteString("http.request.body.content", body),
			zap.Int("http.request.body.bytes", len(body)),
			zap.String("http.request.mime_type", req.Header.Get("Content-Type")),
		)
	}
	message, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		errorsMessages = append(errorsMessages, fmt.Sprintf("failed to dump request: %s", err))
	} else {
		reqParts = append(reqParts, zap.ByteString("event.original", message))
	}
	switch len(errorsMessages) {
	case 0:
	case 1:
		reqParts = append(reqParts, zap.String("error.message", errorsMessages[0]))
	default:
		reqParts = append(reqParts, zap.Strings("error.message", errorsMessages))
	}
	log.Debug("HTTP request", reqParts...)

	resp, err := rt.transport.RoundTrip(req)
	if err != nil {
		log.Debug("HTTP response error", zap.NamedError("error.message", err))
		return resp, err
	}
	if resp == nil {
		log.Debug("HTTP response error", noResponse)
		return resp, err
	}
	respParts := append(reqParts[:0],
		zap.Int("http.response.status_code", resp.StatusCode),
	)
	errorsMessages = errorsMessages[:0]
	resp.Body, body, err = copyBody(resp.Body)
	if err != nil {
		errorsMessages = append(errorsMessages, fmt.Sprintf("failed to read response body: %s", err))
	} else {
		respParts = append(respParts,
			zap.ByteString("http.response.body.content", body),
			zap.Int("http.response.body.bytes", len(body)),
			zap.String("http.response.mime_type", resp.Header.Get("Content-Type")),
		)
	}
	message, err = httputil.DumpResponse(resp, true)
	if err != nil {
		errorsMessages = append(errorsMessages, fmt.Sprintf("failed to dump response: %s", err))
	} else {
		respParts = append(respParts, zap.ByteString("event.original", message))
	}
	switch len(errorsMessages) {
	case 0:
	case 1:
		respParts = append(reqParts, zap.String("error.message", errorsMessages[0]))
	default:
		respParts = append(reqParts, zap.Strings("error.message", errorsMessages))
	}
	log.Debug("HTTP response", respParts...)

	return resp, err
}

// nextTxID returns the next transaction.id value. It increments the internal
// request counter.
func (rt *LoggingRoundTripper) nextTxID() string {
	count := rt.txIDCounter.Inc()
	return rt.txBaseID + "-" + strconv.FormatUint(count, 10)
}

var noResponse = zap.NamedError("error.message", errors.New("unexpected nil response"))

// newID returns an ID derived from the current time.
func newID() string {
	var data [8]byte
	binary.LittleEndian.PutUint64(data[:], uint64(time.Now().UnixNano()))
	return base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(data[:])
}

// copyBody is derived from drainBody in net/http/httputil/dump.go
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// copyBody reads all of b to memory and then returns a
// ReadCloser yielding the same bytes, and the bytes themselves.
//
// It returns an error if the initial slurp of all bytes fails.
func copyBody(b io.ReadCloser) (r io.ReadCloser, body []byte, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, nil, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, buf.Bytes(), err
	}
	if err = b.Close(); err != nil {
		return nil, buf.Bytes(), err
	}
	return io.NopCloser(&buf), buf.Bytes(), nil
}

// MetricsRoundTripper is an http.RoundTripper that monitors requests and responses.
type MetricsRoundTripper struct {
	transport http.RoundTripper

	metrics *httpMetrics
}

type httpMetrics struct {
	reqs          *monitoring.Uint // total number of requests
	reqErrs       *monitoring.Uint // total number of request errors
	reqDelete     *monitoring.Uint // number of DELETE requests
	reqGet        *monitoring.Uint // number of GET requests
	reqHead       *monitoring.Uint // number of HEAD requests
	reqOptions    *monitoring.Uint // number of OPTIONS requests
	reqPatch      *monitoring.Uint // number of PATCH requests
	reqPost       *monitoring.Uint // number of POST requests
	reqPut        *monitoring.Uint // number of PUT requests
	reqsAccSize   *monitoring.Uint // accumulated request body size
	reqsSize      metrics.Sample   // histogram of the request body size
	resps         *monitoring.Uint // total number of responses
	respErrs      *monitoring.Uint // total number of response errors
	resp1xx       *monitoring.Uint // number of 1xx responses
	resp2xx       *monitoring.Uint // number of 2xx responses
	resp3xx       *monitoring.Uint // number of 3xx responses
	resp4xx       *monitoring.Uint // number of 4xx responses
	resp5xx       *monitoring.Uint // number of 5xx responses
	respsAccSize  *monitoring.Uint // accumulated response body size
	respsSize     metrics.Sample   // histogram of the response body size
	roundTripTime metrics.Sample   // histogram of the round trip (request -> response) time
}

// NewMetricsRoundTripper returns a MetricsRoundTripper that sends requests and
// responses metrics to the provided input monitoring registry.
// It will register all http related metrics into the provided registry, but it is not responsible
// for its lifecyle.
func NewMetricsRoundTripper(next http.RoundTripper, reg *monitoring.Registry) *MetricsRoundTripper {
	return &MetricsRoundTripper{
		transport: next,
		metrics:   newHTTPMetrics(reg),
	}
}

func newHTTPMetrics(reg *monitoring.Registry) *httpMetrics {
	if reg == nil {
		return nil
	}

	out := &httpMetrics{
		reqs:          monitoring.NewUint(reg, "http_request_total"),
		reqErrs:       monitoring.NewUint(reg, "http_request_errors_total"),
		reqDelete:     monitoring.NewUint(reg, "http_request_delete_total"),
		reqGet:        monitoring.NewUint(reg, "http_request_get_total"),
		reqHead:       monitoring.NewUint(reg, "http_request_head_total"),
		reqOptions:    monitoring.NewUint(reg, "http_request_options_total"),
		reqPatch:      monitoring.NewUint(reg, "http_request_patch_total"),
		reqPost:       monitoring.NewUint(reg, "http_request_post_total"),
		reqPut:        monitoring.NewUint(reg, "http_request_put_total"),
		reqsAccSize:   monitoring.NewUint(reg, "http_request_body_bytes_total"),
		reqsSize:      metrics.NewUniformSample(1024),
		resps:         monitoring.NewUint(reg, "http_response_total"),
		respErrs:      monitoring.NewUint(reg, "http_response_errors_total"),
		resp1xx:       monitoring.NewUint(reg, "http_response_1xx_total"),
		resp2xx:       monitoring.NewUint(reg, "http_response_2xx_total"),
		resp3xx:       monitoring.NewUint(reg, "http_response_3xx_total"),
		resp4xx:       monitoring.NewUint(reg, "http_response_4xx_total"),
		resp5xx:       monitoring.NewUint(reg, "http_response_5xx_total"),
		respsAccSize:  monitoring.NewUint(reg, "http_response_body_bytes_total"),
		respsSize:     metrics.NewUniformSample(1024),
		roundTripTime: metrics.NewUniformSample(1024),
	}

	_ = adapter.GetGoMetrics(reg, "http_request_body_bytes", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.reqsSize))
	_ = adapter.GetGoMetrics(reg, "http_response_body_bytes", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.respsSize))
	_ = adapter.GetGoMetrics(reg, "http_round_trip_time", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.roundTripTime))

	return out
}

// RoundTrip implements the http.RoundTripper interface, sending
// request and response metrics to the underlying registry.
//
//	http_request_total
//	http_request_errors_total
//	http_request_delete_total
//	http_request_get_total
//	http_request_head_total
//	http_request_options_total
//	http_request_patch_total
//	http_request_post_total
//	http_request_put_total
//	http_request_body_bytes_total
//	http_request_body_bytes
//	http_response_total
//	http_response_errors_total
//	http_response_1xx_total
//	http_response_2xx_total
//	http_response_3xx_total
//	http_response_4xx_total
//	http_response_5xx_total
//	http_response_body_bytes_total
//	http_response_body_bytes
//	http_round_trip_time
func (rt *MetricsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.metrics == nil {
		return rt.transport.RoundTrip(req)
	}

	rt.metrics.reqs.Add(1)

	rt.monitorByMethod(req.Method)

	var (
		body []byte
		err  error
	)

	req.Body, body, err = copyBody(req.Body)
	if err != nil {
		rt.metrics.reqErrs.Add(1)
	} else {
		rt.metrics.reqsAccSize.Add(uint64(len(body)))
		rt.metrics.reqsSize.Update(int64(len(body)))
	}

	reqStart := time.Now()
	resp, err := rt.transport.RoundTrip(req)
	rt.metrics.roundTripTime.Update(time.Since(reqStart).Nanoseconds())

	if resp != nil {
		rt.metrics.resps.Add(1)
	}

	if resp == nil || err != nil {
		rt.metrics.respErrs.Add(1)
		return resp, err
	}

	rt.monitorByStatusCode(resp.StatusCode)

	resp.Body, body, err = copyBody(resp.Body)
	if err != nil {
		rt.metrics.respErrs.Add(1)
	} else {
		rt.metrics.respsAccSize.Add(uint64(len(body)))
		rt.metrics.respsSize.Update(int64(len(body)))
	}

	return resp, err
}

func (rt *MetricsRoundTripper) monitorByMethod(method string) {
	switch method {
	case http.MethodDelete:
		rt.metrics.reqDelete.Add(1)
	case http.MethodGet:
		rt.metrics.reqGet.Add(1)
	case http.MethodHead:
		rt.metrics.reqHead.Add(1)
	case http.MethodOptions:
		rt.metrics.reqOptions.Add(1)
	case http.MethodPatch:
		rt.metrics.reqPatch.Add(1)
	case http.MethodPost:
		rt.metrics.reqPost.Add(1)
	case http.MethodPut:
		rt.metrics.reqPut.Add(1)
	}
}

func (rt *MetricsRoundTripper) monitorByStatusCode(code int) {
	switch code / 100 {
	case 1:
		rt.metrics.resp1xx.Add(1)
	case 2:
		rt.metrics.resp2xx.Add(1)
	case 3:
		rt.metrics.resp3xx.Add(1)
	case 4:
		rt.metrics.resp4xx.Add(1)
	case 5:
		rt.metrics.resp5xx.Add(1)
	}
}
