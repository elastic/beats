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
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/publisher"
	"github.com/elastic/beats/v8/libbeat/publisher/queue"
)

type forgetfulProducer struct {
	broker    *broker
	openState openState
}

type ackProducer struct {
	broker       *broker
	dropOnCancel bool
	seq          uint32
	state        produceState
	openState    openState
}

type openState struct {
	log    logger
	done   chan struct{}
	events chan pushRequest
}

type produceState struct {
	cb        ackHandler
	dropCB    func(beat.Event)
	cancelled bool
	lastACK   uint32
}

type ackHandler func(count int)

func newProducer(b *broker, cb ackHandler, dropCB func(beat.Event), dropOnCancel bool) queue.Producer {
	openState := openState{
		log:    b.logger,
		done:   make(chan struct{}),
		events: b.events,
	}

	if cb != nil {
		p := &ackProducer{broker: b, seq: 1, dropOnCancel: dropOnCancel, openState: openState}
		p.state.cb = cb
		p.state.dropCB = dropCB
		return p
	}
	return &forgetfulProducer{broker: b, openState: openState}
}

func (p *forgetfulProducer) Publish(event publisher.Event) bool {
	return p.openState.publish(p.makeRequest(event))
}

func (p *forgetfulProducer) TryPublish(event publisher.Event) bool {
	return p.openState.tryPublish(p.makeRequest(event))
}

func (p *forgetfulProducer) makeRequest(event publisher.Event) pushRequest {
	return pushRequest{event: event}
}

func (p *forgetfulProducer) Cancel() int {
	p.openState.Close()
	return 0
}

func (p *ackProducer) Publish(event publisher.Event) bool {
	return p.updSeq(p.openState.publish(p.makeRequest(event)))
}

func (p *ackProducer) TryPublish(event publisher.Event) bool {
	return p.updSeq(p.openState.tryPublish(p.makeRequest(event)))
}

func (p *ackProducer) updSeq(ok bool) bool {
	if ok {
		p.seq++
	}
	return ok
}

func (p *ackProducer) makeRequest(event publisher.Event) pushRequest {
	req := pushRequest{
		event: event,
		seq:   p.seq,
		state: &p.state,
	}
	return req
}

func (p *ackProducer) Cancel() int {
	p.openState.Close()

	if p.dropOnCancel {
		ch := make(chan producerCancelResponse)
		p.broker.pubCancel <- producerCancelRequest{
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
		st.log.Debugf("Dropping event, queue is blocked (seq=%v) ", req.seq)
		return false
	}
}
