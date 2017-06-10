package membroker

import (
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/broker"
)

type forgetfullProducer struct {
	broker *Broker
}

type ackProducer struct {
	broker *Broker
	cancel bool
	seq    uint32
	state  produceState
}

type produceState struct {
	cb        ackHandler
	dropCB    func(int)
	cancelled bool
	lastACK   uint32
}

type ackHandler func(count int)

func newProducer(b *Broker, cb ackHandler, dropCB func(int)) broker.Producer {
	if cb != nil {
		p := &ackProducer{broker: b, seq: 1, cancel: true}
		p.state.cb = cb
		p.state.dropCB = dropCB
		return p
	}
	return &forgetfullProducer{broker: b}
}

func (p *forgetfullProducer) Publish(event publisher.Event) {
	p.broker.publish(p.makeRequest(event))
}

func (p *forgetfullProducer) TryPublish(event publisher.Event) bool {
	return p.broker.tryPublish(p.makeRequest(event))
}

func (p *forgetfullProducer) makeRequest(event publisher.Event) pushRequest {
	return pushRequest{event: event}
}

func (*forgetfullProducer) Cancel() int { return 0 }

func (p *ackProducer) Publish(event publisher.Event) {
	p.broker.publish(p.makeRequest(event))
}

func (p *ackProducer) TryPublish(event publisher.Event) bool {
	return p.broker.tryPublish(p.makeRequest(event))
}

func (p *ackProducer) makeRequest(event publisher.Event) pushRequest {
	req := pushRequest{
		event: event,
		seq:   p.seq,
		state: &p.state,
	}
	p.seq++
	return req
}

func (p *ackProducer) Cancel() int {
	if p.cancel {
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

func (b *Broker) publish(req pushRequest) { b.events <- req }
func (b *Broker) tryPublish(req pushRequest) bool {
	select {
	case b.events <- req:
		return true
	default:
		return false
	}
}
