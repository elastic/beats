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

package split

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/reader"
)

type Config struct {
	Target string `config:"target" validation:"required"`
	// Type   string `config:"type"`
	Split      *Config `config:"split"`
	KeepParent bool    `config:"keep_parent"`
	// KeyField   string `config:"key_field"`
	// DelimiterString  string       `config:"delimiter"`
	IgnoreEmptyValue bool `config:"ignore_empty_value"`
}

// Validate validates the Config option for JSON reader.
func (c *Config) Validate() error {
	return nil
}

type splitterReader struct {
	reader reader.Reader
	ctx    context.Context
	cancel context.CancelFunc
	cfg    *Config
	buf    chan reader.Message
	logger *logp.Logger
}

var (
	splitOnce sync.Once
)

func New(r reader.Reader, cfg *Config) *splitterReader {
	return &splitterReader{
		reader: r,
		cfg:    cfg,
		logger: logp.NewLogger("parser_split"),
		buf:    make(chan reader.Message),
		ctx:    context.Background(),
	}
}

func (r *splitterReader) Next() (reader.Message, error) {
	splitOnce.Do(func() {
		go r.reading()
	})

	for r.ctx.Err() == nil {
		msg := <-r.buf
		return msg, nil
	}
	return reader.Message{}, nil
}

func (r *splitterReader) reading() {
	for r.ctx.Err() == nil {
		message, err := r.reader.Next()
		if err != nil {
			return
		}
		split, err := NewSplit(r.cfg, r.logger)
		if err != nil {
			return
		}
		// We want to be able to identify which split is the root of the chain.
		split.IsRoot = true
		data, _ := json.Marshal(message.Content)
		eventsCh, err := split.StartSplit(data)
		if err != nil {
			r.logger.Errorf("error splitting response: %v", err)
			return
		}
		for maybeMsg := range eventsCh {
			if maybeMsg.Failed() {
				r.logger.Errorf("error processing response: %v", maybeMsg)
				continue
			}
			r.buf <- reader.Message{
				Content: []byte(maybeMsg.Msg.String()),
			}
		}
	}
}

func (r *splitterReader) Close() error {
	return r.reader.Close()
	// return nil
}
