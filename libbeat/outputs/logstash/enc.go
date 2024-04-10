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

package logstash

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

type eventEncoder struct {
	log   *logp.Logger
	enc   *json.Encoder
	index string
}

type encodedEvent struct {
	encoding []byte
	err      error
}

func newEventEncoderFactory(
	log *logp.Logger,
	info beat.Info,
	escapeHTML bool,
	index string,
) queue.EncoderFactory {
	return func() queue.Encoder {
		return newEventEncoder(log, info, escapeHTML, index)
	}
}

func newEventEncoder(
	log *logp.Logger,
	info beat.Info,
	escapeHTML bool,
	index string,
) queue.Encoder {
	enc := json.New(info.Version, json.Config{
		Pretty:     false,
		EscapeHTML: escapeHTML,
	})
	return &eventEncoder{
		log:   log,
		enc:   enc,
		index: index,
	}
}

func (e *eventEncoder) EncodeEntry(entry queue.Entry) (queue.Entry, int) {
	pubEvent, ok := entry.(publisher.Event)
	if !ok {
		// Currently all queue entries are publisher.Events but let's be cautious.
		return entry, 0
	}
	encoding, err := e.enc.Encode(e.index, &pubEvent.Content)
	if err != nil {
		e.log.Debugf("Failed to encode event: %v", pubEvent.Content)
	}
	pubEvent.EncodedEvent = &encodedEvent{
		encoding: encoding,
		err:      err,
	}
	pubEvent.Content = beat.Event{}
	return pubEvent, len(encoding)
}

func logstashEventUnwrapper(event interface{}) ([]byte, error) {
	encoded, ok := event.(*encodedEvent)
	if !ok {
		return nil, fmt.Errorf("event is wrong type (expected *encodedEvent)")
	}
	return encoded.encoding, encoded.err
}
