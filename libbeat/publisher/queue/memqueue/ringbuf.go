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

	"github.com/elastic/elastic-agent-libs/logp"
)

// Internal event ring buffer.
// The ring is split into 2 contiguous regions.
// Events are appended to region A until it grows to the end of the internal
// buffer. Then region B is created at the beginning of the internal buffer,
// and events are inserted there until region A is emptied. When A becomes empty,
// we rename region B to region A, and the cycle repeats every time we wrap around
// the internal array storage.
type ringBuffer struct {
	logger *logp.Logger

	entries []queueEntry

	// The underlying array is divided up into two contiguous regions.
	regA, regB region

	// The number of events awaiting ACK at the beginning of region A.
	reserved int
}

// region represents a contiguous region in ringBuffer's internal storage (i.e.
// one that does not cross the end of the array).
type region struct {
	// The starting position of this region within the full event buffer.
	index int

	// The number of events currently stored in this region.
	size int
}

func (b *ringBuffer) init(logger *logp.Logger, size int) {
	*b = ringBuffer{
		logger:  logger,
		entries: make([]queueEntry, size),
	}
}

// Returns true if the ringBuffer is full after handling
// the given insertion, false otherwise.
func (b *ringBuffer) insert(entry queueEntry) {
	// always insert into region B, if region B exists.
	// That is, we have 2 regions and region A is currently processed by consumers
	if b.regB.size > 0 {
		// log.Debug("  - push into B region")

		idx := b.regB.index + b.regB.size
		avail := b.regA.index - idx
		if avail > 0 {
			b.entries[idx] = entry
			b.regB.size++
		}
		return
	}

	// region B does not exist yet, check if region A is available for use
	idx := b.regA.index + b.regA.size
	if b.regA.index+b.regA.size >= len(b.entries) {
		// region A extends to the end of the buffer
		if b.regA.index > 0 {
			// If there is space before region A, create
			// region B there.
			b.regB = region{index: 0, size: 1}
			b.entries[0] = entry
		}
		return
	}

	// space available in region A -> let's append the event
	// log.Debug("  - push into region A")
	b.entries[idx] = entry
	b.regA.size++
}

// cancel removes all buffered events matching `st`, not yet reserved by
// any consumer
func (b *ringBuffer) cancel(producer *ackProducer) int {
	cancelledB := b.cancelRegion(producer, b.regB)
	b.regB.size -= cancelledB

	cancelledA := b.cancelRegion(producer, region{
		index: b.regA.index + b.reserved,
		size:  b.regA.size - b.reserved,
	})
	b.regA.size -= cancelledA

	return cancelledA + cancelledB
}

// cancelRegion removes the events in the specified range having
// the specified produceState. It returns the number of events
// removed.
func (b *ringBuffer) cancelRegion(producer *ackProducer, reg region) int {
	start := reg.index
	end := start + reg.size
	entries := b.entries[start:end]

	toEntries := entries[:0]

	// filter loop
	for i := 0; i < reg.size; i++ {
		if entries[i].producer == producer {
			continue // remove
		}
		toEntries = append(toEntries, entries[i])
	}

	// re-initialize old buffer elements to help garbage collector
	entries = entries[len(toEntries):]
	for i := range entries {
		entries[i] = queueEntry{}
	}

	return len(entries)
}

// reserve returns up to `sz` events from the brokerBuffer,
// exclusively marking the events as 'reserved'. Subsequent calls to `reserve`
// will only return enqueued and non-reserved events from the buffer.
// If `sz == -1`, all available events will be reserved.
func (b *ringBuffer) reserve(sz int) (int, []queueEntry) {
	use := b.regA.size - b.reserved

	if sz > 0 && use > sz {
		use = sz
	}

	start := b.regA.index + b.reserved
	end := start + use
	b.reserved += use
	return start, b.entries[start:end]
}

// Remove the specified number of previously-reserved buffer entries from the
// start of region A. Called by the event loop when events are ACKed by
// consumers.
func (b *ringBuffer) removeEntries(count int) {
	if b.regA.size < count {
		panic(fmt.Errorf("commit region to big (commit region=%v, buffer size=%v)",
			count, b.regA.size,
		))
	}

	// clear region, so published events can be collected by the garbage collector:
	end := b.regA.index + count
	for i := b.regA.index; i < end; i++ {
		b.entries[i] = queueEntry{}
	}

	b.regA.index = end
	b.regA.size -= count
	b.reserved -= count
	if b.regA.size == 0 {
		// region A is empty, transfer region B into region A
		b.regA = b.regB
		b.regB.index = 0
		b.regB.size = 0
	}
}

// Number of events that consumers can currently request.
func (b *ringBuffer) Avail() int {
	return b.regA.size - b.reserved
}

func (b *ringBuffer) Full() bool {
	if b.regB.size > 0 {
		return b.regA.index == (b.regB.index + b.regB.size)
	}
	return b.regA.size == len(b.entries)
}

func (b *ringBuffer) Size() int {
	return len(b.entries)
}

// Items returns the count of events currently in the buffer
func (b *ringBuffer) Items() int {
	return b.regA.size + b.regB.size
}

func (b *ringBuffer) OldestEntry() *queueEntry {
	if b.regA.size == 0 {
		return nil
	}
	return &b.entries[b.regA.index]
}
