// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap/zapcore"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/elastic-agent-libs/logp"
)

type websocketStream struct {
	processor

	id     string
	cfg    config
	cursor map[string]any

	time func() time.Time
}

// NewWebsocketFollower performs environment construction including CEL
// program and regexp compilation, and input metrics set-up for a websocket
// stream follower.
func NewWebsocketFollower(ctx context.Context, id string, cfg config, cursor map[string]any, pub inputcursor.Publisher, log *logp.Logger, now func() time.Time) (StreamFollower, error) {
	s := websocketStream{
		id:     id,
		cfg:    cfg,
		cursor: cursor,
		processor: processor{
			ns:      "websocket",
			pub:     pub,
			log:     log,
			redact:  cfg.Redact,
			metrics: newInputMetrics(id),
		},
	}
	s.metrics.url.Set(cfg.URL.String())
	s.metrics.errorsTotal.Set(0)

	patterns, err := regexpsFromConfig(cfg)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		s.Close()
		return nil, err
	}

	s.prg, s.ast, err = newProgram(ctx, cfg.Program, root, patterns, log)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		s.Close()
		return nil, err
	}

	return &s, nil
}

// FollowStream receives, processes and publishes events from the subscribed
// websocket stream.
func (s *websocketStream) FollowStream(ctx context.Context) error {
	state := s.cfg.State
	if state == nil {
		state = make(map[string]any)
	}
	if s.cursor != nil {
		state["cursor"] = s.cursor
	}

	// initialize the input url with the help of the url_program.
	url, err := getURL(ctx, "websocket", s.cfg.URLProgram, s.cfg.URL.String(), state, s.cfg.Redact, s.log, s.now)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		return err
	}

	// websocket client
	c, resp, err := connectWebSocket(ctx, s.cfg, url, s.log)
	handleConnectionResponse(resp, s.metrics, s.log)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		s.log.Errorw("failed to establish websocket connection", "error", err)
		return err
	}

	// ensures this is the last connection closed when the function returns
	defer func() {
		if err := c.Close(); err != nil {
			s.metrics.errorsTotal.Inc()
			s.log.Errorw("encountered an error while closing the websocket connection", "error", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			s.log.Debugw("context cancelled, closing websocket connection")
			return ctx.Err()
		default:
			_, message, err := c.ReadMessage()
			if err != nil {
				s.metrics.errorsTotal.Inc()
				if isRetryableError(err) {
					s.log.Debugw("websocket connection encountered an error, attempting to reconnect...", "error", err)
					// close the old connection and reconnect
					if err := c.Close(); err != nil {
						s.metrics.errorsTotal.Inc()
						s.log.Errorw("encountered an error while closing the websocket connection", "error", err)
					}
					// since c is already a pointer, we can reassign it to the new connection and the defer func will still handle it
					c, resp, err = connectWebSocket(ctx, s.cfg, url, s.log)
					handleConnectionResponse(resp, s.metrics, s.log)
					if err != nil {
						s.metrics.errorsTotal.Inc()
						s.log.Errorw("failed to reconnect websocket connection", "error", err)
						return err
					}
				} else {
					s.log.Errorw("failed to read websocket data", "error", err)
					return err
				}
			}
			s.metrics.receivedBytesTotal.Add(uint64(len(message)))
			state["response"] = message
			s.log.Debugw("received websocket message", logp.Namespace("websocket"), string(message))
			err = s.process(ctx, state, s.cursor, s.now().In(time.UTC))
			if err != nil {
				s.metrics.errorsTotal.Inc()
				s.log.Errorw("failed to process and publish data", "error", err)
				return err
			}
		}
	}
}

// isRetryableError checks if the error is retryable based on the error type.
func isRetryableError(err error) bool {
	// check for specific network errors
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		switch {
		case netErr.Op == "dial" && netErr.Err.Error() == "i/o timeout",
			netErr.Op == "read" && netErr.Err.Error() == "i/o timeout",
			netErr.Op == "read" && netErr.Err.Error() == "connection reset by peer",
			netErr.Op == "read" && netErr.Err.Error() == "connection refused",
			netErr.Op == "read" && netErr.Err.Error() == "connection reset",
			netErr.Op == "read" && netErr.Err.Error() == "connection closed":
			return true
		}
	}

	// check for specific websocket close errors
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		switch closeErr.Code {
		case websocket.CloseGoingAway,
			websocket.CloseNormalClosure,
			websocket.CloseInternalServerErr,
			websocket.CloseTryAgainLater,
			websocket.CloseServiceRestart,
			websocket.CloseTLSHandshake:
			return true
		}
	}

	// check for common error patterns
	if strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "temporary failure") ||
		strings.Contains(err.Error(), "server is busy") {
		return true
	}

	return false
}

// handleConnectionResponse logs the response body of the websocket connection.
func handleConnectionResponse(resp *http.Response, metrics *inputMetrics, log *logp.Logger) {
	if resp != nil && resp.Body != nil {
		var buf bytes.Buffer
		defer resp.Body.Close()

		if log.Core().Enabled(zapcore.DebugLevel) {
			const limit = 1e4
			if _, err := io.CopyN(&buf, resp.Body, limit); err != nil && !errors.Is(err, io.EOF) {
				metrics.errorsTotal.Inc()
				log.Errorw("failed to read websocket response body", "error", err)
				return
			}
		}

		// discard the remaining part of the body and check for truncation.
		if n, err := io.Copy(io.Discard, resp.Body); err != nil {
			metrics.errorsTotal.Inc()
			log.Errorw("failed to discard remaining response body", "error", err)
		} else if n != 0 && buf.Len() != 0 {
			buf.WriteString("... truncated")
		}

		log.Debugw("websocket connection response", "body", &buf)
	}
}

// connectWebSocket attempts to connect to the websocket server with exponential backoff if retry config is available else it connects without retry.
func connectWebSocket(ctx context.Context, cfg config, url string, log *logp.Logger) (*websocket.Conn, *http.Response, error) {
	var conn *websocket.Conn
	var response *http.Response
	var err error
	headers := formHeader(cfg)

	if cfg.Retry != nil {
		retryConfig := cfg.Retry
		for attempt := 1; attempt <= retryConfig.MaxAttempts; attempt++ {
			conn, response, err = websocket.DefaultDialer.Dial(url, nil)
			if err == nil {
				return conn, response, nil
			}
			log.Debugw("attempt %d: webSocket connection failed. retrying...\n", attempt)
			waitTime := calculateWaitTime(retryConfig.WaitMin, retryConfig.WaitMax, attempt)
			time.Sleep(waitTime)
		}
		return nil, nil, fmt.Errorf("failed to establish WebSocket connection after %d attempts with error %w", retryConfig.MaxAttempts, err)
	}

	return websocket.DefaultDialer.DialContext(ctx, url, headers)
}

// calculateWaitTime calculates the wait time for the next attempt based on the exponential backoff algorithm.
func calculateWaitTime(waitMin, waitMax time.Duration, attempt int) time.Duration {
	// calculate exponential backoff
	base := float64(waitMin)
	backoff := base * math.Pow(2, float64(attempt-1))

	// calculate jitter proportional to the backoff
	maxJitter := float64(waitMax-waitMin) * math.Pow(2, float64(attempt-1))
	jitter := rand.Float64() * maxJitter

	waitTime := time.Duration(backoff + jitter)

	return waitTime
}

// now is time.Now with a modifiable time source.
func (s *websocketStream) now() time.Time {
	if s.time == nil {
		return time.Now()
	}
	return s.time()
}

func (s *websocketStream) Close() error {
	s.metrics.Close()
	return nil
}
