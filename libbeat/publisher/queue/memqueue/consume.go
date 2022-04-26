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
	"fmt"
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
	ack      *ackChan
	state    ackState
}

type ackState uint8

const (
	batchActive ackState = iota
	batchACK
)

func newConsumer(b *broker) *consumer {
	return &consumer{
		broker: b,
		resp:   make(chan getResponse),
		done:   make(chan struct{}),
	}
}

func (c *consumer) Get(sz int) (queue.Batch, error) {
	fmt.Printf("consumer.Get(%d)\n", sz)
	defer fmt.Printf("consumer.Get done\n")
	if c.closed.Load() {
		return nil, io.EOF
	}

	select {
	case c.broker.requests <- getRequest{sz: sz, resp: c.resp}:
		fmt.Printf("sent request\n")
	case <-c.done:
		fmt.Printf("done channel closed\n")
		return nil, io.EOF
	}

	// if request has been send, we do have to wait for a response
	resp := <-c.resp
	fmt.Printf("received response\n")
	events := make([]publisher.Event, 0, len(resp.entries))
	for _, entry := range resp.entries {
		if event, ok := entry.event.(*publisher.Event); ok {
			events = append(events, *event)
		}
	}
	return &batch{
		consumer: c,
		events:   events,
		ack:      resp.ack,
		state:    batchActive,
	}, nil
}

func (c *consumer) Close() error {
	fmt.Printf("\033[0;32mmemqueue consumer close\033[0m\n")

	if c.closed.Swap(true) {
		fmt.Printf("\033[0;32malready closed\033[0m\n")
		return errors.New("already closed")
	}

	close(c.done)
	return nil
}

func (b *batch) Events() []publisher.Event {
	if b.state != batchActive {
		panic("Get Events from inactive batch")
	}
	return b.events
}

func (b *batch) ACK() {
	if b.state != batchActive {
		switch b.state {
		case batchACK:
			panic("Can not acknowledge already acknowledged batch")
		default:
			panic("inactive batch")
		}
	}

	b.report()
}

func (b *batch) report() {
	b.ack.ch <- batchAckMsg{}
}
