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

package pq

import "time"

// Observer defines common callbacks to observe operations, outcomes and stats
// on queues.
// Each callback reports the header offset for uniquely identifying a queue in
// case a file holds many queues.
type Observer interface {
	OnQueueInit(headerOffset uintptr, version uint32, available uint)

	OnQueueFlush(headerOffset uintptr, stats FlushStats)

	OnQueueRead(headerOffset uintptr, stats ReadStats)

	OnQueueACK(headerOffset uintptr, stats ACKStats)
}

// FlushStats reports internal stats on the most recent flush operation.
type FlushStats struct {
	Duration       time.Duration // duration of flush operation
	Oldest, Newest time.Time     // timestamp of oldest/newest event in buffer

	Failed      bool // set to true if flush operation failed
	OutOfMemory bool // set to true if flush failed due to the file being full

	Pages    uint // number of pages to be flushed
	Allocate uint // number of pages to allocate during flush operation
	Events   uint // number of events to be flushed

	BytesTotal uint // total number of bytes written (ignoring headers, just event sizes)
	BytesMin   uint // size of 'smallest' event in current transaction
	BytesMax   uint // size of 'biggest' event in current transaction
}

// ReadStats reports stats on the most recent transaction for reading events.
type ReadStats struct {
	Duration time.Duration // duration of read transaction

	Skipped uint // number of events skipped (e.g. upon error while reading/parsing)
	Read    uint // number of events read

	BytesTotal   uint // total number of bytes read (ignoring headers). Include partially but skipped events
	BytesSkipped uint // number of event bytes skipped
	BytesMin     uint // size of 'smallest' event fully read in current transaction
	BytesMax     uint // size of 'biggest' event fully read in current transaction
}

// ACKStats reports stats on the most recent ACK transaction.
type ACKStats struct {
	Duration time.Duration
	Failed   bool

	Events uint // number of released events
	Pages  uint // number of released pages
}
