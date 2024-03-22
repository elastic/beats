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

import "github.com/elastic/beats/v7/libbeat/publisher/queue"

type batch struct {
	entries []queueEntry

	// Original number of entries (persists even if entries are freed).
	originalEntryCount int

	producerACKs []producerACKData

	// When a batch is acknowledged, doneChan is closed to tell
	// the queue to call the appropriate producer and metrics callbacks.
	doneChan chan struct{}

	// Batches are collected in linked lists to preserve the order of
	// acknowledgments. This field should only be used by batchList.
	next *batch
}

type batchList struct {
	first *batch
	last  *batch
}

// producerACKData tracks the number of events that need to be acknowledged
// from a single batch targeting a single producer.
type producerACKData struct {
	producer *producer
	count    int
}

func (b *batch) Count() int {
	return b.originalEntryCount
}

func (b *batch) Entry(i int) queue.Event {
	return b.entries[i].event
}

func (b *batch) FreeEntries() {
	b.entries = nil
}

func (b *batch) Done() {
	close(b.doneChan)
}

func acksForEntries(entries []queueEntry) []producerACKData {
	results := []producerACKData{}
	// We traverse the list back to front, so we can coalesce multiple events
	// into a single entry in the ACK data.
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if producer := entry.producer; producer != nil {
			if producer.producedCount > producer.consumedCount {
				results = append(results, producerACKData{
					producer: producer,
					count:    int(producer.producedCount - producer.consumedCount),
				})
				producer.consumedCount = producer.producedCount
			}
		}
	}
	return results
}

func (l *batchList) add(b *batch) {
	b.next = nil // Should be unneeded but let's be cautious
	if l.last != nil {
		l.last.next = b
	} else {
		l.first = b
	}
	l.last = b
}

func (l *batchList) remove() *batch {
	result := l.first
	if l.first != nil {
		l.first = l.first.next
		if l.first == nil {
			l.last = nil
		}
	}
	return result
}

func (l *batchList) nextDoneChan() chan struct{} {
	if l.first != nil {
		return l.first.doneChan
	}
	return nil
}
