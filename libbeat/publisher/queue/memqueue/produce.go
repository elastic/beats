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
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

type forgetfulProducer struct {
	broker    *broker
	openState openState
}

type ackProducer struct {
	broker        *broker
	dropOnCancel  bool
	producedCount uint64
	state         produceState
	openState     openState
}

type openState struct {
	log    *logp.Logger
	done   chan struct{}
	events chan pushRequest
}

type produceState struct {
	cb         ackHandler
	dropCB     func(beat.Event)
	cancelled  bool
	ackedCount uint64
}

type ackHandler func(count int)

func newProducer(b *broker, cb ackHandler, dropCB func(beat.Event), dropOnCancel bool) queue.Producer {
	openState := openState{
		log:    b.logger,
		done:   make(chan struct{}),
		events: b.pushChan,
	}

	if cb != nil {
		p := &ackProducer{broker: b, producedCount: 1, dropOnCancel: dropOnCancel, openState: openState}
		p.state.cb = cb
		p.state.dropCB = dropCB
		return p
	}
	return &forgetfulProducer{broker: b, openState: openState}
}

func (p *forgetfulProducer) Publish(event interface{}) (queue.EntryID, bool) {
	return 0, p.openState.publish(pushRequest{event: event})
}

func (p *forgetfulProducer) TryPublish(event interface{}) (queue.EntryID, bool) {
	return 0, p.openState.tryPublish(pushRequest{event: event})
}

func (p *forgetfulProducer) Cancel() int {
	p.openState.Close()
	return 0
}

func (p *ackProducer) Publish(event interface{}) (queue.EntryID, bool) {
	resp := make(chan queue.EntryID, 1)
	if p.openState.publish(pushRequest{
		event: event,
		state: &p.state,
		resp:  resp,
	}) {
		id := <-resp
		return id, true
	}
	return 0, false
}

func (p *ackProducer) TryPublish(event interface{}) (queue.EntryID, bool) {
	resp := make(chan queue.EntryID, 1)
	if p.openState.tryPublish(pushRequest{
		event: event,
		state: &p.state,
		resp:  resp,
	}) {
		id := <-resp
		return id, true
	}
	return 0, false
}

func (p *ackProducer) Cancel() int {
	p.openState.Close()

	if p.dropOnCancel {
		ch := make(chan producerCancelResponse)
		p.broker.cancelChan <- producerCancelRequest{
			state: &p.state,
			resp:  ch,
		}

		// wait for cancel to being processed
		resp := <-ch
		return resp.removed
	}
	return 0
}

func (st *openState) Close() {
	close(st.done)
}

func (st *openState) publish(req pushRequest) bool {
	select {
	case st.events <- req:
		return true
	case <-st.done:
		st.events = nil
		return false
	}
}

func (st *openState) tryPublish(req pushRequest) bool {
	select {
	case st.events <- req:
		return true
	case <-st.done:
		st.events = nil
		return false
	default:
		st.log.Debugf("Dropping event, queue is blocked")
		return false
	}
}
