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

	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ http.RoundTripper = (*LoggingRoundTripper)(nil)

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
//	event.original (the request without body from httputil.DumpRequestOut)
//
// Fields logged in responses:
//
//	http.response.status_code
//	http.response.body.content
//	http.response.body.bytes
//	http.response.mime_type
//	event.original (the response without body from httputil.DumpResponse)
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
	message, err := httputil.DumpRequestOut(req, false)
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
	message, err = httputil.DumpResponse(resp, false)
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
