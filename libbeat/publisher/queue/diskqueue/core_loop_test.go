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

import "testing"

func TestProducerWriteRequest(t *testing.T) {
	dq := &diskQueue{settings: DefaultSettings()}
	frame := &writeFrame{
		serialized: make([]byte, 100),
	}
	request := producerWriteRequest{
		frame:        frame,
		shouldBlock:  true,
		responseChan: make(chan bool, 1),
	}
	dq.handleProducerWriteRequest(request)

	// The request inserts 100 bytes into an empty queue, so it should succeed.
	// We expect:
	// - the response channel should contain the value true
	// - the frame should be added to pendingFrames and assigned to
	//   segment 0.
	success, ok := <-request.responseChan
	if !ok {
		t.Error("Expected a response from the producer write request.")
	}
	if !success {
		t.Error("Expected write request to succeed")
	}

	if len(dq.pendingFrames) != 1 {
		t.Error("Expected 1 pending frame after a write request.")
	}
	if dq.pendingFrames[0].frame != frame {
		t.Error("Expected pendingFrames to contain the new frame.")
	}
	if dq.pendingFrames[0].segment.id != 0 {
		t.Error("Expected new frame to be assigned to segment 0.")
	}
}

func TestHandleWriterLoopResponse(t *testing.T) {
	// Initialize the queue with two writing segments only.
	dq := &diskQueue{
		settings: DefaultSettings(),
		segments: diskQueueSegments{
			writing: []*queueSegment{
				{id: 1},
				{id: 2},
			},
		},
	}
	// This response says that the writer loop wrote 200 bytes to the first
	// segment and 100 bytes to the second.
	dq.handleWriterLoopResponse(writerLoopResponse{
		bytesWritten: []int64{200, 100},
	})

	// After the response is handled, we expect:
	// - Each segment's endOffset should be incremented by the bytes written
	// - Segment 1 should be moved to the reading list (because all but the
	//   last segment in a writer loop response has been closed)
	// - Segment 2 should remain in the writing list
	if len(dq.segments.reading) != 1 || dq.segments.reading[0].id != 1 {
		t.Error("Expected segment 1 to move to the reading list")
	}
	if len(dq.segments.writing) != 1 || dq.segments.writing[0].id != 2 {
		t.Error("Expected segment 2 to remain in the writing list")
	}
	if dq.segments.reading[0].endOffset != 200 {
		t.Errorf("Expected segment 1 endOffset 200, got %d",
			dq.segments.reading[0].endOffset)
	}
	if dq.segments.writing[0].endOffset != 100 {
		t.Errorf("Expected segment 2 endOffset 100, got %d",
			dq.segments.writing[0].endOffset)
	}
}
