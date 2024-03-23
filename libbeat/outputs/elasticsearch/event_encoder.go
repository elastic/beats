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

package elasticsearch

import (
	"bytes"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type eventEncoder struct {
	buf              *bytes.Buffer
	enc              eslegclient.BodyEncoder
	pipelineSelector *outil.Selector
	indexSelector    outputs.IndexSelector
}

type encodedEvent struct {
	// If err is set, the event couldn't be encoded, and other fields should
	// not be relied on.
	err error

	// If deadLetter is true, this event produced an ingestion error on a
	// previous attempt, and is now being retried as a bare event with all
	// contents included as a raw string in the "message" field.
	deadLetter bool

	// timestamp is the timestamp from the source beat.Event. It's only used
	// when reencoding for the dead letter index, so it isn't strictly needed
	// but it avoids deserializing the encoded event to recover one field if
	// there's an ingestion error.
	timestamp time.Time

	id       string
	opType   events.OpType
	pipeline string
	index    string
	encoding []byte
}

func newEventEncoder(escapeHTML bool,
	indexSelector outputs.IndexSelector,
	pipelineSelector *outil.Selector,
) queue.Encoder {
	buf := bytes.NewBuffer(nil)
	enc := eslegclient.NewJSONEncoder(buf, escapeHTML)
	return &eventEncoder{
		buf:              buf,
		enc:              enc,
		pipelineSelector: pipelineSelector,
		indexSelector:    indexSelector,
	}
}

func (pe *eventEncoder) EncodeEntry(entry queue.Entry) (queue.Entry, int) {
	e, ok := entry.(publisher.Event)
	if !ok {
		// Currently all queue entries are publisher.Events but let's be cautious.
		return entry, 0
	}

	encodedEvent := pe.encodeRawEvent(&e.Content)
	e.EncodedEvent = encodedEvent
	e.Content = beat.Event{}
	return e, len(encodedEvent.encoding)
}

// Note: we can't early-encode the bulk metadata that goes with an event,
// because it depends on the upstream Elasticsearch version and thus requires
// a live client connection. However, benchmarks show that even for a known
// version, encoding the bulk metadata and the event together gives slightly
// worse performance, so there's no reason to try optimizing around this
// dependency.
func (pe *eventEncoder) encodeRawEvent(e *beat.Event) *encodedEvent {
	opType := events.GetOpType(*e)
	pipeline, err := getPipeline(e, pe.pipelineSelector)
	if err != nil {
		return &encodedEvent{err: fmt.Errorf("failed to select event pipeline: %w", err)}
	}
	index, err := pe.indexSelector.Select(e)
	if err != nil {
		return &encodedEvent{err: fmt.Errorf("failed to select event index: %w", err)}
	}

	id, _ := events.GetMetaStringValue(*e, events.FieldMetaID)

	err = pe.enc.Marshal(e)
	if err != nil {
		return &encodedEvent{err: fmt.Errorf("failed to encode event for output: %w", err)}
	}
	bufBytes := pe.buf.Bytes()
	bytes := make([]byte, len(bufBytes))
	copy(bytes, bufBytes)
	return &encodedEvent{
		id:        id,
		timestamp: e.Timestamp,
		opType:    opType,
		pipeline:  pipeline,
		index:     index,
		encoding:  bytes,
	}
}
