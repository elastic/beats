// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
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
	handleConnctionResponse(resp, s.log)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		s.log.Errorw("failed to establish websocket connection", "error", err)
		return err
	}
	// ensures this is the last connection closed when the function returns
	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			s.metrics.errorsTotal.Inc()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				s.log.Debugw("websocket connection closed, attempting to reconnect...", "error", err)
				// close the old connection and reconnect
				c.Close()
				// since c is already a pointer, we can reassign it to the new connection and the defer will still handle it
				c, resp, err = connectWebSocket(ctx, s.cfg, url, s.log)
				handleConnctionResponse(resp, s.log)
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

// handleConnctionResponse logs the response body of the websocket connection.
func handleConnctionResponse(resp *http.Response, log *logp.Logger) {
	if resp != nil && resp.Body != nil {
		var buf bytes.Buffer
		if log.Core().Enabled(zapcore.DebugLevel) {
			const limit = 1e4
			io.CopyN(&buf, resp.Body, limit)
		}
		if n, _ := io.Copy(io.Discard, resp.Body); n != 0 && buf.Len() != 0 {
			buf.WriteString("... truncated")
		}
		log.Debugw("websocket connection response", "body", &buf)
		resp.Body.Close()
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
		for attempt := 1; attempt <= *retryConfig.MaxAttempts; attempt++ {
			conn, response, err = websocket.DefaultDialer.Dial(url, nil)
			if err == nil {
				return conn, response, nil
			}
			log.Debugw("attempt %d: webSocket connection failed. retrying...\n", attempt)
			waitTime := calculateWaitTime(*retryConfig.WaitMin, *retryConfig.WaitMax, attempt)
			time.Sleep(waitTime)
		}
		return nil, nil, fmt.Errorf("failed to establish WebSocket connection after %d attempts with error %w", *retryConfig.MaxAttempts, err)
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
