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

import "github.com/elastic/beats/v7/libbeat/publisher/queue"

// producer -> broker API

type pushRequest struct {
	event queue.Entry

	// The event's encoded size in bytes if the configured output supports
	// early encoding, 0 otherwise.
	eventSize int

	// If the queue doesn't have room for an incoming event and blockIfFull
	// is true, the request will be held until there is space in the queue.
	// Otherwise, the queue will return failure immediately.
	blockIfFull bool

	// The producer that generated this event, or nil if this producer does
	// not require ack callbacks.
	producer *ackProducer

	// The index of the event in this producer only. Used to condense
	// multiple acknowledgments for a producer to a single callback call.
	producerID producerID
	resp       chan queue.EntryID
}

type producerCancelRequest struct {
	producer *ackProducer
	resp     chan producerCancelResponse
}

type producerCancelResponse struct {
	removed int
}

// consumer -> broker API

type getRequest struct {
	// The number of entries to request, or <= 0 for no limit.
	entryCount int

	// The number of (encoded) event bytes to request, or <= 0 for no limit.
	byteCount int

	// The channel to send the new batch to.
	responseChan chan *batch
}

type batchDoneMsg struct{}

// Metrics API

type metricsRequest struct {
	responseChan chan memQueueMetrics
}

// memQueueMetrics tracks metrics that are returned by the individual memory queue implementations
type memQueueMetrics struct {
	// the size of items in the queue
	currentQueueSize int
	// the number of items that have been read by a consumer but not yet ack'ed
	occupiedRead int

	oldestEntryID queue.EntryID
}
