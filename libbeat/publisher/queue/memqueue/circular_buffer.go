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

// the queue's underlying array buffer needs to coordinate concurrent
// access by:
//
//				runLoop
//				 - when a pushRequest is accepted, writes to the newly created entry index.
//				 - when a producer is cancelled, reads and writes to entry indices that
//				   have been created but not yet consumed, to discard events from that
//				   producer.
//				 - when entries are deleted (after consumed events have been
//		      acknowledged), reads from the deleted entry indices.
//			  - when a pushRequest requires resizing of the array, expands and/or
//			    replaces the buffer.
//
//				the queue's consumer (in a live Beat this means queueReader in
//				libbeat/publisher/pipeline/queue_reader.go) which reads from entry
//				indices that have been consumed but not deleted via (*batch).Entry().
//
//	 ackLoop, which reads producer metadata from acknowledged entry
//	 indices before they are deleted so acknowledgment callbacks can be
//	 invoked.
//
// Most of these are not in conflict since they access disjoint array indices.
// The exception is growing the circular buffer, which conflicts with read
// access from batches of consumed events.
type circularBuffer struct {
	// Do not access this array directly! use (circularBuffer).entry().
	_entries []queueEntry
}

type entryIndex int

func newCircularBuffer(size int) circularBuffer {
	return circularBuffer{
		_entries: make([]queueEntry, size),
	}
}

func (cb circularBuffer) size() int {
	return len(cb._entries)
}

func (cb circularBuffer) entry(i entryIndex) *queueEntry {
	rawIndex := int(i) % len(cb._entries)
	return &cb._entries[rawIndex]
}

func (ei entryIndex) plus(offset int) entryIndex {
	return entryIndex(int(ei) + offset)
}
