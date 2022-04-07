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

	"github.com/elastic/beats/v8/libbeat/common/atomic"
	"github.com/elastic/beats/v8/libbeat/publisher"
	"github.com/elastic/beats/v8/libbeat/publisher/queue"
)

type consumer struct {
	broker *broker
	resp   chan getResponse

	done   chan struct{}
	closed atomic.Bool
}

type batch struct {
	consumer     *consumer
	events       []publisher.Event
	clientStates []clientState
	ack          *ackChan
	state        ackState
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
	if c.closed.Load() {
		return nil, io.EOF
	}

	select {
	case c.broker.requests <- getRequest{sz: sz, resp: c.resp}:
	case <-c.done:
		return nil, io.EOF
	}

	// if request has been send, we do have to wait for a response
	resp := <-c.resp
	return &batch{
		consumer: c,
		events:   resp.buf,
		ack:      resp.ack,
		state:    batchActive,
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
