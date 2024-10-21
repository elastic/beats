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
