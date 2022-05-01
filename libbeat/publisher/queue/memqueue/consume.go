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

package memqueue

import (
	"errors"
	"io"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type consumer struct {
	broker *broker
	resp   chan getResponse

	done   chan struct{}
	closed atomic.Bool
}

type batch struct {
	consumer *consumer
	events   []publisher.Event
	ackChan  chan batchAckMsg
}

func newConsumer(b *broker) *consumer {
	return &consumer{
		broker: b,
		resp:   make(chan getResponse),
		done:   make(chan struct{}),
	}
}

func (c *consumer) Get(sz int) (queue.Batch, error) {
	if c.closed.Load() {
		return nil, io.EOF
	}

	select {
	case c.broker.getChan <- getRequest{entryCount: sz, responseChan: c.resp}:
	case <-c.done:
		return nil, io.EOF
	}

	// if request has been send, we do have to wait for a response
	resp := <-c.resp
	events := make([]publisher.Event, 0, len(resp.entries))
	for _, entry := range resp.entries {
		if event, ok := entry.event.(*publisher.Event); ok {
			events = append(events, *event)
		}
	}
	return &batch{
		consumer: c,
		events:   events,
		ackChan:  resp.ackChan,
	}, nil
}

func (c *consumer) Close() error {
	if c.closed.Swap(true) {
		return errors.New("already closed")
	}
	close(c.done)
	return nil
}

func (b *batch) Events() []publisher.Event {
	return b.events
}

func (b *batch) ACK() {
	b.ackChan <- batchAckMsg{}
}
