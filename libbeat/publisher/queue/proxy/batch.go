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
	entryCount int

	producerACKs       []producerACKData
	metricsACKListener queue.ACKListener
}

// producerACKData tracks the number of events that need to be acknowledged
// from a single batch targeting a single producer.
type producerACKData struct {
	producer *producer
	count    int
}

func (b *batch) Count() int {
	return b.entryCount
}

func (b *batch) Entry(i int) interface{} {
	return b.entries[i].event
}

func (b *batch) FreeEntries() {
	b.entries = nil
}

func (b *batch) Done() {
	for _, ack := range b.producerACKs {
		ack.producer.ackHandler(ack.count)
	}
	if b.metricsACKListener != nil {
		b.metricsACKListener.OnACK(b.entryCount)
	}
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
