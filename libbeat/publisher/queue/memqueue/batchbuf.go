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

type batchBuffer struct {
	next    *batchBuffer
	flushed bool
	entries []queueEntry
}

func newBatchBuffer(sz int) *batchBuffer {
	b := &batchBuffer{}
	b.entries = make([]queueEntry, 0, sz)
	return b
}

func (b *batchBuffer) add(entry queueEntry) {
	b.entries = append(b.entries, entry)
}

func (b *batchBuffer) length() int {
	return len(b.entries)
}

func (b *batchBuffer) cancel(producer *ackProducer) int {
	entries := b.entries[:0]

	removedCount := 0
	for _, entry := range b.entries {
		if entry.producer == producer {
			removedCount++
			continue
		}
		entries = append(entries, entry)
	}
	b.entries = entries
	return removedCount
}
