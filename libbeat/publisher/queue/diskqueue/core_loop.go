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

package diskqueue

import "github.com/elastic/beats/v7/libbeat/publisher"

// A frame waiting to be written to disk
type writeFrame struct {
	// The original event provided by the client to diskQueueProducer
	event publisher.Event

	// The event, serialized for writing to disk and wrapped in a frame
	// header / footer.
	serialized []byte
}

// A frame that has been read from disk
type readFrame struct {
}

// A request sent from a producer to the core loop to add a frame to the queue.
//
type writeRequest struct {
	frame        *writeFrame
	shouldBlock  bool
	responseChan chan bool
}

// A readRequest is sent from the reader loop to the core loop when it
// needs a new segment file to read.
type readRequest struct {
	responseChan chan *readResponse
}

type readResponse struct {
}

type cancelRequest struct {
	producer *diskQueueProducer
	// If producer.config.DropOnCancel is true, then the core loop will respond
	// on responseChan with the number of dropped events.
	// Otherwise, this field may be nil.
	responseChan chan int
}
