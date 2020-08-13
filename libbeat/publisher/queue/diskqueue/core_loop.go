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

func (dq *diskQueue) coreLoop() {
	// writing is true if a writeRequest is currently being processed by the
	// writer loop, false otherwise.
	writing := false

	// reading is true if the reader loop is processing a readBlock, false
	// otherwise.
	reading := false

	// deleting is true if the segment-deletion loop is processing a deletion
	// request, false otherwise.
	deleting := false

	// writeRequests is a list of all write requests that have been accepted
	// by the queue and are waiting to be written to disk.
	writeRequests := []*writeRequest{}

	for {
		select {
		// Endpoints used by the external API
		case writeRequest := <-dq.writeRequestChan:
			// We will accept this request if there is enough capacity left in
			// the queue (after accounting for the pending writes that were
			// already accepted).
			pendingBytes := 0
			for _, request := range writeRequests {
				pendingBytes += len(request.serialized)
			}

		case cancelRequest := <-dq.cancelRequestChan:

		// Writer loop handling
		case <-dq.writerLoop.finishedWriting:
			if len(writeRequests) > 0 {
				dq.forwardWriteRequest(writeRequests[0])
				writeRequests = writeRequests[1:]
			} else {
				writing = false
			}

		// Reader loop handling
		case readResponse := <-dq.readerLoop.finishedReading:

		// Deleter loop handling
		case deleteResponse := <-dq.deleterLoop.response:
			if len(deleteResponse.deleted) > 0 {
				// One or more segments were deleted, recompute the outstanding list.
				newAckedSegments := []*queueSegment{}
				for _, segment := range dq.segments.acked {
					if !deleteResponse.deleted[segment] {
						// This segment wasn't deleted, so it goes in the new list.
						newAckedSegments = append(newAckedSegments, segment)
					}
				}
				dq.segments.acked = newAckedSegments
			}
			if len(deleteResponse.errors) > 0 {
				dq.settings.Logger.Errorw("Couldn't delete old segment files",
					"errors", deleteResponse.errors)
			}

			if len(dq.segments.acked) > 0 {
				// There are still (possibly new) segments to delete, send the
				// next batch.
				dq.deleterLoop.input <- &deleteRequest{segments: dq.segments.acked}
			} else {
				// Nothing more to delete for now, update the deleting flag.
				deleting = false
			}
		}
	}
}

func (dq *diskQueue) forwardWriteRequest(request *writeRequest) {
	// First we must decide which segment the new frame should be written to.
	data := request.frame.serialized
	segment := dq.segments.writing

	if segment != nil &&
		segment.size+uint64(len(data)) > dq.settings.MaxSegmentSize {
		// The new frame is too big to fit in this segment, so close it and
		// move it to the read queue.
		segment.writer.Close()
		// TODO: make reasonable attempts to sync the closed file.
		dq.segments.reading = append(dq.segments.reading, segment)
		segment = nil
	}

	// If we don't have a segment, we need to create one.
	if segment == nil {
		segment = &queueSegment{id: dq.segments.nextID}
	}
}
