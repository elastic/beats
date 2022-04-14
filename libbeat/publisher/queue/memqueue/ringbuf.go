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
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

// Internal event ring buffer.
// The ring is split into 2 regions.
// Region A contains active events to be send to consumers, while region B can
// only be filled by producers, if there is no more space in region A. Splitting
// the ring buffer into regions enables the broker to send batches of type
// []publisher.Event to the consumer without having to copy and/or grow/shrink the
// buffers.
type ringBuffer struct {
	buf eventBuffer

	regA, regB region
	reserved   int // amount of events in region A actively processed/reserved
}

type region struct {
	index int
	size  int
}

type eventBuffer struct {
	logger *logp.Logger

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

func (b *ringBuffer) init(log *logp.Logger, size int) {
	*b = ringBuffer{}
	b.buf.init(size)
	b.buf.logger = log
}

func (b *ringBuffer) insert(event publisher.Event, client clientState) (bool, int) {
	// log := b.buf.logger
	// log.Debug("insert:")
	// log.Debug("  region A:", b.regA)
	// log.Debug("  region B:", b.regB)
	// log.Debug("  reserved:", b.reserved)
	// defer func() {
	// 	log.Debug("  -> region A:", b.regA)
	// 	log.Debug("  -> region B:", b.regB)
	// 	log.Debug("  -> reserved:", b.reserved)
	// }()

	// always insert into region B, if region B exists.
	// That is, we have 2 regions and region A is currently processed by consumers
	if b.regB.size > 0 {
		// log.Debug("  - push into B region")

		idx := b.regB.index + b.regB.size
		avail := b.regA.index - idx
		if avail == 0 {
			return false, 0
		}

		b.buf.Set(idx, event, client)
		b.regB.size++

		return true, avail - 1
	}

	// region B does not exist yet, check if region A is available for use
	idx := b.regA.index + b.regA.size
	// log.Debug("  - index: ", idx)
	// log.Debug("  - buffer size: ", b.buf.Len())
	avail := b.buf.Len() - idx
	if avail == 0 { // no more space in region A
		// log.Debug("  - region A full")

		if b.regA.index == 0 {
			// space to create region B, buffer is full

			// log.Debug("  - no space in region B")

			return false, 0
		}

		// create region B and insert events
		// log.Debug("  - create region B")
		b.regB.index = 0
		b.regB.size = 1
		b.buf.Set(0, event, client)
		return true, b.regA.index - 1
	}

	// space available in region A -> let's append the event
	// log.Debug("  - push into region A")
	b.buf.Set(idx, event, client)
	b.regA.size++
	return true, avail - 1
}

// cancel removes all buffered events matching `st`, not yet reserved by
// any consumer
func (b *ringBuffer) cancel(st *produceState) int {
	// log := b.buf.logger
	// log.Debug("cancel:")
	// log.Debug("  region A:", b.regA)
	// log.Debug("  region B:", b.regB)
	// log.Debug("  reserved:", b.reserved)
	// defer func() {
	// 	log.Debug("  -> region A:", b.regA)
	// 	log.Debug("  -> region B:", b.regB)
	// 	log.Debug("  -> reserved:", b.reserved)
	// }()

	// TODO: return if st has no pending events

	cancelB := b.cancelRegion(st, b.regB)
	b.regB.size -= cancelB

	cancelA := b.cancelRegion(st, region{
		index: b.regA.index + b.reserved,
		size:  b.regA.size - b.reserved,
	})
	b.regA.size -= cancelA

	return cancelA + cancelB
}

func (b *ringBuffer) cancelRegion(st *produceState, reg region) (removed int) {
	start := reg.index
	end := start + reg.size
	events := b.buf.events[start:end]
	clients := b.buf.clients[start:end]

	toEvents := events[:0]
	toClients := clients[:0]

	// filter loop
	for i := 0; i < reg.size; i++ {
		if clients[i].state == st {
			continue // remove
		}

		toEvents = append(toEvents, events[i])
		toClients = append(toClients, clients[i])
	}

	// re-initialize old buffer elements to help garbage collector
	events = events[len(toEvents):]
	clients = clients[len(toClients):]
	for i := range events {
		events[i] = publisher.Event{}
		clients[i] = clientState{}
	}

	return len(events)
}

// reserve returns up to `sz` events from the brokerBuffer,
// exclusively marking the events as 'reserved'. Subsequent calls to `reserve`
// will only return enqueued and non-reserved events from the buffer.
// If `sz == -1`, all available events will be reserved.
func (b *ringBuffer) reserve(sz int) (int, []publisher.Event) {
	// log := b.buf.logger
	// log.Debug("reserve: ", sz)
	// log.Debug("  region A:", b.regA)
	// log.Debug("  region B:", b.regB)
	// log.Debug("  reserved:", b.reserved)
	// defer func() {
	// 	log.Debug("  -> region A:", b.regA)
	// 	log.Debug("  -> region B:", b.regB)
	// 	log.Debug("  -> reserved:", b.reserved)
	// }()

	use := b.regA.size - b.reserved
	// log.Debug("  - avail: ", use)

	if sz > 0 {
		if use > sz {
			use = sz
		}
	}

	start := b.regA.index + b.reserved
	end := start + use
	b.reserved += use
	// log.Debug("  - start:", start)
	// log.Debug("  - end:", end)
	return start, b.buf.events[start:end]
}

// ack up to sz events in region A
func (b *ringBuffer) ack(sz int) {
	// log := b.buf.logger
	// log.Debug("ack: ", sz)
	// log.Debug("  region A:", b.regA)
	// log.Debug("  region B:", b.regB)
	// log.Debug("  reserved:", b.reserved)
	// defer func() {
	// 	log.Debug("  -> region A:", b.regA)
	// 	log.Debug("  -> region B:", b.regB)
	// 	log.Debug("  -> reserved:", b.reserved)
	// }()

	if b.regA.size < sz {
		panic(fmt.Errorf("commit region to big (commit region=%v, buffer size=%v)",
			sz, b.regA.size,
		))
	}

	// clear region, so published events can be collected by the garbage collector:
	end := b.regA.index + sz
	for i := b.regA.index; i < end; i++ {
		b.buf.events[i] = publisher.Event{}
	}

	b.regA.index = end
	b.regA.size -= sz
	b.reserved -= sz
	if b.regA.size == 0 {
		// region A is empty, transfer region B into region A
		b.regA = b.regB
		b.regB.index = 0
		b.regB.size = 0
	}
}

func (b *ringBuffer) Avail() int {
	return b.regA.size - b.reserved
}

func (b *ringBuffer) Full() bool {
	var avail int
	if b.regB.size > 0 {
		avail = b.regA.index - b.regB.index - b.regB.size
	} else {
		avail = b.buf.Len() - b.regA.index - b.regA.size
	}
	return avail == 0
}

func (b *ringBuffer) Size() int {
	return b.buf.Len()
}
