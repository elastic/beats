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

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/match"
	"github.com/elastic/go-concert/ctxtool"
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
	ctx      ctxtool.CancelContext
	logger   *logp.Logger
	r        reader.Reader
	matchers []match.Matcher
}

func NewParser(r reader.Reader, c *Config) *FilterParser {
	return &FilterParser{
		ctx:      ctxtool.WithCancelContext(context.Background()),
		logger:   logp.NewLogger("filter_parser"),
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
		if p.matchAny(string(message.Content)) {
			return message, err
		}
		discardedOffset += message.Bytes
		p.logger.Debug("dropping message because it does not match any of the provided patterns [%v]: %s", p.matchers, string(message.Content))
	}
	return reader.Message{}, io.EOF
}

func (p *FilterParser) matchAny(text string) bool {
	for _, m := range p.matchers {
		if m.MatchString(text) {
			return true
		}
	}
	return false
}

func (p *FilterParser) Close() error {
	p.ctx.Cancel()
	return p.r.Close()
}
