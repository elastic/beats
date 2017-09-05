package memqueue

import (
	"github.com/elastic/beats/libbeat/publisher"
)

type eventBuffer struct {
	logger logger

	events  []publisher.Event
	clients []clientState
}

type clientState struct {
	seq   uint32        // event sequence number
	state *produceState // the producer it's state used to compute and signal the ACK count
}

func (b *eventBuffer) init(size int) {
	b.events = make([]publisher.Event, size)
	b.clients = make([]clientState, size)
}

func (b *eventBuffer) Len() int {
	return len(b.events)
}

func (b *eventBuffer) Set(idx int, event publisher.Event, st clientState) {
	// b.logger.Debugf("insert event: idx=%v, seq=%v\n", idx, st.seq)

	b.events[idx] = event
	b.clients[idx] = st
}
