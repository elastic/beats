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

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
)

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

func TestHandleReaderLoopResponse(t *testing.T) {
	// handleReaderLoopResponse should:
	// - advance segments.{nextReadFrameID, nextReadOffset} by the values in
	//   response.{frameCount, byteCount}
	// - advance the target segment's framesRead field by response.frameCount
	// - if reading[0] encountered an error or was completely read, move it from
	//   the reading list to the acking list and reset nextReadOffset to zero
	// - if writing[0] encountered an error, advance nextReadOffset to the
	//   segment's current endOffset (we can't discard the active writing
	//   segment like we do for errors in the reading list, but we can still
	//   mark the remaining data as processed)

	testCases := map[string]struct {
		// The segment structure to start with before calling maybeReadPending
		segments diskQueueSegments
		response readerLoopResponse

		expectedFrameID       frameID
		expectedOffset        segmentOffset
		expectedACKingSegment *segmentID
	}{
		"completely read first reading segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 10,
				byteCount:  1000,
			},
			expectedFrameID:       15,
			expectedOffset:        0,
			expectedACKingSegment: segmentIDRef(1),
		},
		"read first half of first reading segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 5,
				byteCount:  500,
			},
			expectedFrameID: 10,
			expectedOffset:  500,
		},
		"read second half of first reading segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				nextReadFrameID: 5,
				nextReadOffset:  500,
			},
			response: readerLoopResponse{
				frameCount: 5,
				byteCount:  500,
			},
			expectedFrameID:       10,
			expectedOffset:        0,
			expectedACKingSegment: segmentIDRef(1),
		},
		"read of first reading segment aborted by error": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 1,
				byteCount:  100,
				err:        fmt.Errorf("something bad happened"),
			},
			expectedFrameID:       6,
			expectedOffset:        0,
			expectedACKingSegment: segmentIDRef(1),
		},
		"completely read first writing segment": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 10,
				byteCount:  1000,
			},
			expectedFrameID: 15,
			expectedOffset:  1000,
		},
		"read first half of first writing segment": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 5,
				byteCount:  500,
			},
			expectedFrameID: 10,
			expectedOffset:  500,
		},
		"read second half of first writing segment": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				nextReadOffset:  500,
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 5,
				byteCount:  500,
			},
			expectedFrameID: 10,
			expectedOffset:  1000,
		},
		"error reading a writing segments skips remaining data": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 1,
				byteCount:  100,
				err:        fmt.Errorf("something bad happened"),
			},
			expectedFrameID: 6,
			expectedOffset:  1000,
		},
	}

	for description, test := range testCases {
		dq := &diskQueue{
			logger:   logp.L(),
			settings: DefaultSettings(),
			segments: test.segments,
		}
		dq.handleReaderLoopResponse(test.response)

		if dq.segments.nextReadFrameID != test.expectedFrameID {
			t.Errorf("%s: expected nextReadFrameID = %d, got %d",
				description, test.expectedFrameID, dq.segments.nextReadFrameID)
		}
		if dq.segments.nextReadOffset != test.expectedOffset {
			t.Errorf("%s: expected nextReadOffset = %d, got %d",
				description, test.expectedOffset, dq.segments.nextReadOffset)
		}
		if test.expectedACKingSegment != nil {
			if len(dq.segments.acking) == 0 {
				t.Errorf("%s: expected acking segment %d, got none",
					description, *test.expectedACKingSegment)
			} else if dq.segments.acking[0].id != *test.expectedACKingSegment {
				t.Errorf("%s: expected acking segment %d, got %d",
					description, *test.expectedACKingSegment, dq.segments.acking[0].id)
			}
		} else if len(dq.segments.acking) != 0 {
			t.Errorf("%s: expected no acking segment, got %v",
				description, *dq.segments.acking[0])
		}
	}
}

func TestMaybeReadPending(t *testing.T) {
	// maybeReadPending should:
	// - If any unread data is available in a reading or writing segment,
	//   send a readerLoopRequest for the full amount available in the
	//   first such segment.
	// - When creating a readerLoopRequest that includes the beginning of
	//   a segment (startOffset == 0), set that segment's firstFrameID
	//   to segments.nextReadFrameID (so ACKs based on frame ID can be linked
	//   back to the segment that generated them).
	// - If the first reading segment has already been completely read (which
	//   can happen if it was read while still in the writing list), move it to
	//   the acking list and set segments.nextReadOffset to 0.

	testCases := map[string]struct {
		// The segment structure to start with before calling maybeReadPending
		segments diskQueueSegments
		// The request we expect to see on the reader loop's request channel,
		// or nil if there should be none.
		expectedRequest *readerLoopRequest
		// The segment ID we expect to see in the acking list, or nil for none.
		expectedACKingSegment *segmentID
	}{
		"read one full segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				// The next read request should start with frame 5
				nextReadFrameID: 5,
			},
			expectedRequest: &readerLoopRequest{
				segment:      &queueSegment{id: 1},
				startFrameID: 5,
				startOffset:  0,
				endOffset:    1000,
			},
		},
		"read the end of a segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				// The next read request should start with frame 5
				nextReadFrameID: 5,
				// Start reading at position 500
				nextReadOffset: 500,
			},
			expectedRequest: &readerLoopRequest{
				segment:      &queueSegment{id: 1},
				startFrameID: 5,
				// Should be reading from nextReadOffset (500) to the end of
				// the segment (1000).
				startOffset: 500,
				endOffset:   1000,
			},
		},
		"ignore writing segments if reading is available": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				writing: []*queueSegment{
					{id: 2, endOffset: 1000},
				},
			},
			expectedRequest: &readerLoopRequest{
				segment:     &queueSegment{id: 1},
				startOffset: 0,
				endOffset:   1000,
			},
		},
		"do nothing if no segments are available": {
			segments:        diskQueueSegments{},
			expectedRequest: nil,
		},
		"read the writing segment if no reading segments are available": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 2, endOffset: 1000},
				},
				nextReadOffset: 500,
			},
			expectedRequest: &readerLoopRequest{
				segment:     &queueSegment{id: 2},
				startOffset: 500,
				endOffset:   1000,
			},
		},
		"do nothing if the writing segment has already been fully read": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 2, endOffset: 1000},
				},
				nextReadOffset: 1000,
			},
			expectedRequest: nil,
		},
		"skip the first reading segment if it's already been fully read": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, endOffset: 1000},
					{id: 2, endOffset: 500},
				},
				nextReadOffset: 1000,
			},
			expectedRequest: &readerLoopRequest{
				segment:     &queueSegment{id: 2},
				startOffset: 0,
				endOffset:   500,
			},
			expectedACKingSegment: segmentIDRef(1),
		},
		"move empty reading segment to the acking list if it's the only one": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, endOffset: 1000},
				},
				nextReadOffset: 1000,
			},
			expectedRequest:       nil,
			expectedACKingSegment: segmentIDRef(1),
		},
	}

	for description, test := range testCases {
		dq := &diskQueue{
			settings: DefaultSettings(),
			segments: test.segments,
			readerLoop: &readerLoop{
				requestChan: make(chan readerLoopRequest, 1),
			},
		}
		firstFrameID := test.segments.nextReadFrameID
		dq.maybeReadPending()
		select {
		case request := <-dq.readerLoop.requestChan:
			if test.expectedRequest == nil {
				t.Errorf("%s: expected no read request, got %v",
					description, request)
				break
			}
			if !equalReaderLoopRequests(request, *test.expectedRequest) {
				t.Errorf("%s: expected request %v, got %v",
					description, *test.expectedRequest, request)
			}
			if request.startOffset == 0 &&
				request.segment.firstFrameID != firstFrameID {
				t.Errorf(
					"%s: maybeReadPending should update firstFrameID", description)
			}
		default:
			if test.expectedRequest != nil {
				t.Errorf("%s: expected read request %v, got none",
					description, test.expectedRequest)
			}
		}
		if test.expectedACKingSegment != nil {
			if len(dq.segments.acking) != 1 {
				t.Errorf("%s: expected acking segment %v, got none",
					description, *test.expectedACKingSegment)
			} else if dq.segments.acking[0].id != *test.expectedACKingSegment {
				t.Errorf("%s: expected acking segment %v, got %v",
					description, *test.expectedACKingSegment, dq.segments.acking[0].id)
			}
			if dq.segments.nextReadOffset != 0 {
				t.Errorf("%s: expected read offset 0 after acking segment, got %v",
					description, dq.segments.nextReadOffset)
			}
		} else if len(dq.segments.acking) != 0 {
			t.Errorf("%s: expected no acking segment, got %v",
				description, *dq.segments.acking[0])
		}
	}
}

func segmentIDRef(id segmentID) *segmentID {
	return &id
}

func equalReaderLoopRequests(
	r0 readerLoopRequest, r1 readerLoopRequest,
) bool {
	// We compare segment ids rather than segment pointers because it's
	// awkward to include the same pointer repeatedly in the test definition.
	return r0.startOffset == r1.startOffset &&
		r0.endOffset == r1.endOffset &&
		r0.segment.id == r1.segment.id &&
		r0.startFrameID == r1.startFrameID
}
