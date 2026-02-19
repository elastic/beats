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
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

var _ http.RoundTripper = (*LoggingRoundTripper)(nil)

// TraceIDKey is key used to add a trace.id value to the context of HTTP
// requests. The value will be logged by LoggingRoundTripper.
const TraceIDKey = contextKey("trace.id")

type contextKey string

// IsPathInLogsFor returns whether path is a valid path for logs written by the
// specified input after resolving symbolic links in path.
func IsPathInLogsFor(input, path string) (ok bool, err error) {
	root := paths.Resolve(paths.Logs, input)
	if !filepath.IsAbs(path) && !isRooted(path) {
		path = filepath.Join(root, path)
	}
	return IsPathIn(root, path)
}

// ResolvePathInLogsFor resolves path relative to the logs directory for the
// specified input and reports whether the result is within that directory.
func ResolvePathInLogsFor(input, path string) (resolved string, ok bool, err error) {
	root := paths.Resolve(paths.Logs, input)
	if !filepath.IsAbs(path) && !isRooted(path) {
		path = filepath.Join(root, path)
	}
	ok, err = IsPathIn(root, path)
	return path, ok, err
}

// isRooted reports whether path begins with a path separator, i.e. it is
// rooted at the filesystem root even if it is not absolute (no drive letter
// on Windows). Such paths must not be joined to a base directory.
func isRooted(path string) bool {
	return len(path) > 0 && os.IsPathSeparator(path[0])
}

// IsPathIn returns whether path is a valid path within root after resolving
// symbolic links in root and path.
func IsPathIn(root, path string) (ok bool, err error) {
	// Get all paths in absolute.
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	absRoot, err = resolveSymlinks(absRoot)
	if err != nil {
		return false, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	absPath, err = resolveSymlinks(absPath)
	if err != nil {
		return false, err
	}
	// Find the traverse from the root to the path.
	traversal, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false, err
	}
	return traversal != ".." && !strings.HasPrefix(traversal, ".."+string(filepath.Separator)), nil
}

func resolveSymlinks(path string) (string, error) {
	targ, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If the path doesn't exist or has invalid syntax for opening
		// (e.g. Windows rejects paths containing * or ? with
		// ERROR_INVALID_NAME), resolve the directory and join the base
		// so we still follow symlinks in the directory part.
		if errors.Is(err, fs.ErrNotExist) || isInvalidWindowsName(err) {
			targ, err := resolveSymlinks(filepath.Dir(path))
			if err != nil {
				return "", err
			}
			return filepath.Join(targ, filepath.Base(path)), nil
		}
		return "", err
	}
	return targ, nil
}

// NewLoggingRoundTripper returns a LoggingRoundTripper that logs requests and
// responses to the provided logger. Transaction creation is logged to log.
func NewLoggingRoundTripper(next http.RoundTripper, logger *zap.Logger, maxBodyLen int, log *logp.Logger) *LoggingRoundTripper {
	return &LoggingRoundTripper{
		transport:  next,
		maxBodyLen: maxBodyLen,
		txLog:      logger,
		txBaseID:   newID(),
		log:        log,
	}
}

// LoggingRoundTripper is an http.RoundTripper that logs requests and responses.
type LoggingRoundTripper struct {
	transport   http.RoundTripper
	maxBodyLen  int           // The maximum length of a body. Longer bodies will be truncated.
	txLog       *zap.Logger   // Destination logger.
	txBaseID    string        // Random value to make transaction IDs unique.
	txIDCounter atomic.Uint64 // Transaction ID counter that is incremented for each request.
	log         *logp.Logger
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
//	http.request.body.truncated
//	http.request.body.bytes
//	http.request.mime_type
//	http.request.header
//
// Fields logged in responses:
//
//	http.response.status_code
//	http.response.body.content
//	http.response.body.truncated
//	http.response.body.bytes
//	http.response.mime_type
//	http.response.header
func (rt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a child logger for this request.
	txID := rt.nextTxID()
	rt.log.Debugw("new request trace transaction", "id", txID)
	log := rt.txLog.With(
		zap.String("transaction.id", txID),
	)

	if v := req.Context().Value(TraceIDKey); v != nil {
		if traceID, ok := v.(string); ok {
			log = log.With(zap.String("trace.id", traceID))
		}
	}

	req, respParts, errorsMessages := logRequest(log, req, rt.maxBodyLen)

	resp, err := rt.transport.RoundTrip(req)
	if err != nil {
		log.Debug("HTTP response error", zap.NamedError("error.message", err))
		return resp, err
	}
	if resp == nil {
		log.Debug("HTTP response error", noResponse)
		return resp, err
	}
	respParts = append(respParts,
		zap.Int("http.response.status_code", resp.StatusCode),
	)
	errorsMessages = errorsMessages[:0]
	var body []byte
	resp.Body, body, err = copyBody(resp.Body)
	if err != nil {
		errorsMessages = append(errorsMessages, fmt.Sprintf("failed to read response body: %s", err))
	}
	respParts = append(respParts,
		zap.ByteString("http.response.body.content", body[:min(len(body), rt.maxBodyLen)]),
		zap.Bool("http.response.body.truncated", rt.maxBodyLen < len(body)),
		zap.Int("http.response.body.bytes", len(body)),
		zap.String("http.response.mime_type", resp.Header.Get("Content-Type")),
		zap.Any("http.response.header", resp.Header),
	)
	switch len(errorsMessages) {
	case 0:
	case 1:
		respParts = append(respParts, zap.String("error.message", errorsMessages[0]))
	default:
		respParts = append(respParts, zap.Strings("error.message", errorsMessages))
	}
	log.Debug("HTTP response", respParts...)

	return resp, err
}

// LogRequest logs an HTTP request to the provided logger.
//
// Fields logged:
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
//	http.request.body.truncated
//	http.request.body.bytes
//	http.request.mime_type
//	http.request.header
//
// Additional fields in extra will also be logged.
func LogRequest(log *zap.Logger, req *http.Request, maxBodyLen int, extra ...zapcore.Field) *http.Request {
	req, _, _ = logRequest(log, req, maxBodyLen, extra...)
	return req
}

func logRequest(log *zap.Logger, req *http.Request, maxBodyLen int, extra ...zapcore.Field) (_ *http.Request, parts []zapcore.Field, errorsMessages []string) {
	reqParts := append([]zapcore.Field{
		zap.String("url.original", req.URL.String()),
		zap.String("url.scheme", req.URL.Scheme),
		zap.String("url.path", req.URL.Path),
		zap.String("url.domain", req.URL.Hostname()),
		zap.String("url.port", req.URL.Port()),
		zap.String("url.query", req.URL.RawQuery),
		zap.String("http.request.method", req.Method),
		zap.Any("http.request.header", req.Header),
		zap.String("user_agent.original", req.Header.Get("User-Agent")),
	}, extra...)

	var (
		body []byte
		err  error
	)
	req.Body, body, err = copyBody(req.Body)
	if err != nil {
		errorsMessages = append(errorsMessages, fmt.Sprintf("failed to read request body: %s", err))
	}
	reqParts = append(reqParts,
		zap.ByteString("http.request.body.content", body[:min(len(body), maxBodyLen)]),
		zap.Bool("http.request.body.truncated", maxBodyLen < len(body)),
		zap.Int("http.request.body.bytes", len(body)),
		zap.String("http.request.mime_type", req.Header.Get("Content-Type")),
	)
	switch len(errorsMessages) {
	case 0:
	case 1:
		reqParts = append(reqParts, zap.String("error.message", errorsMessages[0]))
	default:
		reqParts = append(reqParts, zap.Strings("error.message", errorsMessages))
	}
	log.Debug("HTTP request", reqParts...)

	return req, reqParts[:0], errorsMessages
}

// TxID returns the current transaction.id value. If rt is nil, the empty string is returned.
func (rt *LoggingRoundTripper) TxID() string {
	if rt == nil {
		return ""
	}
	count := rt.txIDCounter.Load()
	return rt.formatTxID(count)
}

// nextTxID returns the next transaction.id value. It increments the internal
// request counter.
func (rt *LoggingRoundTripper) nextTxID() string {
	count := rt.txIDCounter.Add(1)
	return rt.formatTxID(count)
}

func (rt *LoggingRoundTripper) formatTxID(count uint64) string {
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
