// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package filter

import (
	"context"
	"io"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/match"
)

type Config struct {
	Patterns []match.Matcher `config:"patterns" validate:"required"`
}

func DefaultConfig() Config {
	return Config{}
}

// FilterParser accepts a list of matchers to determine if a line
// should be kept or not. If one of the patterns matches the
// contents of the message, it is returned to the next reader.
// If not, the message is dropped.
type FilterParser struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger   *logp.Logger
	r        reader.Reader
	matchers []match.Matcher
}

func NewParser(r reader.Reader, c *Config, logger *logp.Logger) *FilterParser {
	ctx, cancel := context.WithCancel(context.Background())
	return &FilterParser{
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger.Named("filter_parser"),
		r:        r,
		matchers: c.Patterns,
	}
}

func (p *FilterParser) Next() (message reader.Message, err error) {
	// discardedOffset accounts for the bytes of discarded messages. The inputs
	// need to correctly track the file offset, therefore if only the matching
	// message size is returned, the offset cannot be correctly updated.
	var discardedOffset int
	defer func() {
		message.Offset = discardedOffset
	}()

	for p.ctx.Err() == nil {
		message, err = p.r.Next()
		if err != nil {
			return message, err
		}
		if p.matchAny(message.Content) {
			return message, err
		}
		discardedOffset += message.Bytes
		if p.logger.IsDebug() {
			p.logger.Debugf("dropping message because it does not match any of the provided patterns [%v]: %s", p.matchers, message.Content)
		}
	}
	return reader.Message{}, io.EOF
}

var _ reader.BufferedFlusher = (*FilterParser)(nil)

// PendingDeadline implements reader.BufferedFlusher by delegating to the
// wrapped reader.
func (p *FilterParser) PendingDeadline() (bool, time.Time) {
	return reader.PendingDeadline(p.r)
}

// FlushPending implements reader.BufferedFlusher. A message flushed from the
// wrapped reader is filtered like one returned by Next. If it does not match,
// it is dropped: ok stays true and an empty message (Bytes == 0, so
// Message.IsEmpty reports true) carrying the dropped bytes in Offset is
// returned. The caller advances its offset by Bytes+Offset before skipping
// empty messages, so the dropped bytes are accounted for without publishing.
func (p *FilterParser) FlushPending(timedOut bool) (reader.Message, bool, error) {
	message, ok, err := reader.FlushPending(p.r, timedOut)
	if !ok || err != nil {
		return message, ok, err
	}

	if p.matchAny(message.Content) {
		return message, true, nil
	}

	if p.logger.IsDebug() {
		p.logger.Debugf("dropping message because it does not match any of the provided patterns [%v]: %s", p.matchers, message.Content)
	}
	return reader.Message{Offset: message.Bytes + message.Offset}, true, nil
}

func (p *FilterParser) matchAny(text []byte) bool {
	for _, m := range p.matchers {
		if m.Match(text) {
			return true
		}
	}
	return false
}

func (p *FilterParser) Close() error {
	p.cancel()
	return p.r.Close()
}

// SetReadDeadline delegates to the wrapped reader (see reader.DeadlineSetter).
func (p *FilterParser) SetReadDeadline(t time.Time) bool {
	return reader.SetReadDeadline(p.r, t)
}
