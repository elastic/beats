// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"bytes"
	"context"
	"io"
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
	headers := formHeader(s.cfg)
	c, resp, err := websocket.DefaultDialer.DialContext(ctx, url, headers)
	if resp != nil && resp.Body != nil {
		var buf bytes.Buffer
		if s.log.Core().Enabled(zapcore.DebugLevel) {
			const limit = 1e4
			io.CopyN(&buf, resp.Body, limit)
		}
		if n, _ := io.Copy(io.Discard, resp.Body); n != 0 && buf.Len() != 0 {
			buf.WriteString("... truncated")
		}
		s.log.Debugw("websocket connection response", "body", &buf)
		resp.Body.Close()
	}
	if err != nil {
		s.metrics.errorsTotal.Inc()
		s.log.Errorw("failed to establish websocket connection", "error", err)
		return err
	}
	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			s.metrics.errorsTotal.Inc()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				s.log.Errorw("websocket connection closed", "error", err)
			} else {
				s.log.Errorw("failed to read websocket data", "error", err)
			}
			return err
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
