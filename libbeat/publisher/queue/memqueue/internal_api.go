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

// producer -> broker API

type pushRequest struct {
	event interface{}
	seq   uint32
	state *produceState
}

type producerCancelRequest struct {
	state *produceState
	resp  chan producerCancelResponse
}

type producerCancelResponse struct {
	removed int
}

// consumer -> broker API

type getRequest struct {
	entryCount   int              // request entryCount events from the broker
	responseChan chan getResponse // channel to send response to
}

type getResponse struct {
	ackChan chan batchAckMsg
	entries []queueEntry
}

type batchAckMsg struct{}

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
}
