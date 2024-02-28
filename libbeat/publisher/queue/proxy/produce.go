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

func (p *producer) makePushRequest(event interface{}, canBlock bool) *pushRequest {
	req := &pushRequest{
		event:        event,
		responseChan: make(chan bool, 1),
		canBlock:     canBlock,
	}
	if p.ackHandler != nil {
		req.producer = p
	}
	return req
}

func (p *producer) Publish(event interface{}) bool {
	if p.cancelled {
		return false
	}
	return p.publish(p.makePushRequest(event, true))
}

func (p *producer) TryPublish(event interface{}) bool {
	if p.cancelled {
		return false
	}
	return p.publish(p.makePushRequest(event, false))
}

func (p *producer) Cancel() int {
	p.cancelled = true
	return 0
}

func (p *producer) publish(req *pushRequest) bool {
	select {
	case p.broker.pushChan <- req:
		return <-req.responseChan
	case <-p.broker.doneChan:
		// The queue is shutting down
		return false
	}
}
