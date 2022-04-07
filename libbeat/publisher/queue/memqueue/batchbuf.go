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

import "github.com/elastic/beats/v8/libbeat/publisher"

type batchBuffer struct {
	next    *batchBuffer
	flushed bool
	events  []publisher.Event
	clients []clientState
}

func newBatchBuffer(sz int) *batchBuffer {
	b := &batchBuffer{}
	b.init(sz)
	return b
}

func (b *batchBuffer) init(sz int) {
	b.events = make([]publisher.Event, 0, sz)
	b.clients = make([]clientState, 0, sz)
}

func (b *batchBuffer) initWith(sz int, old batchBuffer) {
	events, clients := old.events, old.clients
	L := len(events)

	b.events = make([]publisher.Event, L, sz)
	b.clients = make([]clientState, L, sz)

	copy(b.events, events)
	copy(b.clients, clients)
}

func (b *batchBuffer) add(event publisher.Event, st clientState) {
	b.events = append(b.events, event)
	b.clients = append(b.clients, st)
}

func (b *batchBuffer) length() int {
	return len(b.events)
}

func (b *batchBuffer) capacity() int {
	return cap(b.events)
}

func (b *batchBuffer) cancel(st *produceState) int {
	events := b.events[:0]
	clients := b.clients[:0]

	removed := 0
	for i := range b.clients {
		if b.clients[i].state == st {
			removed++
			continue
		}

		events = append(events, b.events[i])
		clients = append(clients, b.clients[i])
	}

	b.events = events
	b.clients = clients
	return removed
}
