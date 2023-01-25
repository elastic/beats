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

package proxyqueue

import (
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type producer struct {
	broker    *broker
	cancelled bool
	// If ackHandler is nil then this producer does not listen to acks.
	ackHandler func(count int)

	// producedCount and consumedCount are used to assemble batches and
	// should only be accessed by the broker's main loop.
	producedCount uint64
	consumedCount uint64
}

func newProducer(b *broker, ackHandler func(count int)) queue.Producer {
	return &producer{
		broker:     b,
		ackHandler: ackHandler}
}

func (p *producer) makePushRequest(event interface{}) pushRequest {
	req := pushRequest{
		event:        event,
		responseChan: make(chan queue.EntryID, 1),
	}
	if p.ackHandler != nil {
		req.producer = p
	}
	return req
}

func (p *producer) Publish(event interface{}) (queue.EntryID, bool) {
	if p.cancelled {
		return 0, false
	}
	return p.broker.publish(p.makePushRequest(event))
}

func (p *producer) TryPublish(event interface{}) (queue.EntryID, bool) {
	if p.cancelled {
		return 0, false
	}
	return p.broker.tryPublish(p.makePushRequest(event))
}

func (p *producer) Cancel() int {
	p.cancelled = true
	return 0
}

func (b *broker) publish(req pushRequest) (queue.EntryID, bool) {
	select {
	case b.pushChan <- req:
		return <-req.responseChan, true
	case <-b.done:
		// The queue is shutting down
		return 0, false
	}
}

func (b *broker) tryPublish(req pushRequest) (queue.EntryID, bool) {
	select {
	case b.pushChan <- req:
		return <-req.responseChan, true
	case <-b.done:
		return 0, false
	default:
		b.logger.Debugf("Dropping event, queue is blocked")
		return 0, false
	}
}
