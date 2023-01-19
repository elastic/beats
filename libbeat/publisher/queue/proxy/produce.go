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
	"github.com/elastic/elastic-agent-libs/logp"
)

type forgetfulProducer struct {
	broker    *broker
	openState openState
}

type batchID uint64

// producerBatchData is a linked list tracking how many events from
// this producer are in each pending batch. It's used to track event
// acknowledgments.
type producerBatchData struct {
	next       *producerBatchData
	batch      batchID
	entryCount int
}

type ackProducer struct {
	broker *broker
	/*dropOnCancel  bool
	producedCount uint64
	state         produceState*/
	openState  openState
	ackHandler ackHandler

	// This field is used to track ACKs for pending events.
	// It can only be safely accessed from the ackLoop.
	pendingBatches *producerBatchData
}

type openState struct {
	log    *logp.Logger
	done   chan struct{}
	events chan pushRequest
}

/*type produceState struct {
	cb        ackHandler
	dropCB    func(interface{})
	cancelled bool
	lastACK   producerID
}*/

type ackHandler func(count int)

func newProducer(b *broker, cb ackHandler, dropCB func(interface{}), dropOnCancel bool) queue.Producer {
	openState := openState{
		log:    b.logger,
		done:   make(chan struct{}),
		events: b.pushChan,
	}

	if cb != nil {
		p := &ackProducer{broker: b, openState: openState, ackHandler: cb}
		//p.state.cb = cb
		//p.state.dropCB = dropCB
		return p
	}
	return &forgetfulProducer{broker: b, openState: openState}
}

func (p *forgetfulProducer) makePushRequest(event interface{}) pushRequest {
	resp := make(chan queue.EntryID, 1)
	return pushRequest{
		event:        event,
		responseChan: resp}
}

func (p *forgetfulProducer) Publish(event interface{}) (queue.EntryID, bool) {
	return p.openState.publish(p.makePushRequest(event))
}

func (p *forgetfulProducer) TryPublish(event interface{}) (queue.EntryID, bool) {
	return p.openState.tryPublish(p.makePushRequest(event))
}

func (p *forgetfulProducer) Cancel() int {
	p.openState.Close()
	return 0
}

func (p *ackProducer) makePushRequest(event interface{}) pushRequest {
	resp := make(chan queue.EntryID, 1)
	return pushRequest{
		event:    event,
		producer: p,
		// We add 1 to the id so the default lastACK of 0 is a
		// valid initial state and 1 is the first real id.
		producerID:   producerID(p.producedCount + 1),
		responseChan: resp}
}

func (p *ackProducer) Publish(event interface{}) (queue.EntryID, bool) {
	id, published := p.openState.publish(p.makePushRequest(event))
	if published {
		p.producedCount++
	}
	return id, published
}

func (p *ackProducer) TryPublish(event interface{}) (queue.EntryID, bool) {
	id, published := p.openState.tryPublish(p.makePushRequest(event))
	if published {
		p.producedCount++
	}
	return id, published
}

func (p *ackProducer) Cancel() int {
	p.openState.Close()

	if p.dropOnCancel {
		ch := make(chan producerCancelResponse)
		p.broker.cancelChan <- producerCancelRequest{
			producer: p,
			resp:     ch,
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

func (st *openState) publish(req pushRequest) (queue.EntryID, bool) {
	select {
	case st.events <- req:
		return <-req.responseChan, true
	case <-st.done:
		st.events = nil
		return 0, false
	}
}

func (st *openState) tryPublish(req pushRequest) (queue.EntryID, bool) {
	select {
	case st.events <- req:
		return <-req.responseChan, true
	case <-st.done:
		st.events = nil
		return 0, false
	default:
		st.log.Debugf("Dropping event, queue is blocked")
		return 0, false
	}
}
