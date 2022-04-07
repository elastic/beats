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

	"github.com/elastic/beats/v8/libbeat/logp"
)

func TestHandleProducerWriteRequest(t *testing.T) {
	// handleProducerWriteRequest should:
	// - Immediately reject any frame larger than settings.MaxSegmentSize.
	// - If dq.blockedProducers is nonempty (indicating that other frames are
	//   already waiting for empty space in the queue), or the queue doesn't
	//   have room for the new frame (see canAcceptFrameOfSize), then it is
	//   appended to blockedProducers if request.shouldBlock is true, and
	//   otherwise is rejected immediately.
	// - Otherwise, the request is assigned a target segment and appended
	//   to pendingFrames.
	//   * If the frame fits in the current writing segment, it is assigned
	//     to that segment. Otherwise, it is assigned to segments.nextID
	//     and segments.nextID is incremented (see enqueueWriteFrame).

	// For this test setup, the queue is initialized with a max segment
	// offset of 1000 and a max total size of 10000.
	testCases := map[string]struct {
		// The segment structure to start with before calling
		// handleProducerWriteRequest
		segments diskQueueSegments

		// Whether the blockedProducers list should be nonempty in the
		// initial queue state.
		blockedProducers bool

		// The size of the frame to send in the producer write request
		frameSize int

		// The value to set shouldBlock to in the producer write request
		shouldBlock bool

		// The result we expect on the requests's response channel, or
		// nil if there should be none.
		expectedResult *bool

		// The segment the frame should be assigned to in pendingFrames.
		// This is ignored unless expectedResult is &true.
		expectedSegment segmentID
	}{
		"accept single frame when empty": {
			segments:        diskQueueSegments{nextID: 5},
			frameSize:       1000,
			shouldBlock:     false,
			expectedResult:  boolRef(true),
			expectedSegment: 5,
		},
		"reject immediately when frame is larger than segment limit": {
			// max segment buffer size for the test wrapper is 1000.
			frameSize:      1001,
			shouldBlock:    true,
			expectedResult: boolRef(false),
		},
		"accept with frame in new segment if current segment is full": {
			segments: diskQueueSegments{
				writing:            []*queueSegment{{}},
				writingSegmentSize: 600,
				nextID:             1,
			},
			frameSize:       500,
			shouldBlock:     false,
			expectedResult:  boolRef(true),
			expectedSegment: 1,
		},
		"reject when full and shouldBlock=false": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{byteCount: 9600},
				},
			},
			frameSize:      500,
			shouldBlock:    false,
			expectedResult: boolRef(false),
		},
		"block when full and shouldBlock=true": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{byteCount: 9600},
				},
			},
			frameSize:      500,
			shouldBlock:    true,
			expectedResult: nil,
		},
		"reject when blockedProducers is nonempty and shouldBlock=false": {
			blockedProducers: true,
			frameSize:        500,
			shouldBlock:      false,
			expectedResult:   boolRef(false),
		},
		"block when blockedProducers is nonempty and shouldBlock=true": {
			blockedProducers: true,
			frameSize:        500,
			shouldBlock:      true,
			expectedResult:   nil,
		},
	}

	settings := DefaultSettings()
	settings.MaxSegmentSize = 1000 + segmentHeaderSize
	settings.MaxBufferSize = 10000
	for description, test := range testCases {
		dq := &diskQueue{
			logger:   logp.L(),
			settings: settings,
			segments: test.segments,
		}
		if test.blockedProducers {
			// Set an empty placeholder write request
			dq.blockedProducers = []producerWriteRequest{{}}
		}
		initialBlockedProducerCount := len(dq.blockedProducers)

		// Construct a frame of the requested size. We subtract the
		// metadata size from the buffer length, so test.frameSize
		// corresponds to the "real" on-disk size of the frame.
		request := producerWriteRequest{
			frame:        makeWriteFrameWithSize(test.frameSize),
			shouldBlock:  test.shouldBlock,
			responseChan: make(chan bool, 1),
		}

		dq.handleProducerWriteRequest(request)

		var result *bool
		select {
		case r := <-request.responseChan:
			result = &r
		default:
			// No response, result can stay nil.
		}

		// Check that the result itself is correct.
		if result != nil && test.expectedResult != nil {
			if *result != *test.expectedResult {
				t.Errorf("%s: expected response %v, got %v",
					description, *test.expectedResult, *result)
			}
		} else if result == nil && test.expectedResult != nil {
			t.Errorf("%s: expected response %v, got none",
				description, *test.expectedResult)
		} else if result != nil && test.expectedResult == nil {
			t.Errorf("%s: expected no response, got %v",
				description, *result)
		}
		// Check whether the request was added to blockedProducers.
		if test.expectedResult != nil &&
			len(dq.blockedProducers) > initialBlockedProducerCount {
			// Requests with responses shouldn't be added to
			// blockedProducers.
			t.Errorf("%s: request shouldn't be added to blockedProducers",
				description)
		} else if test.expectedResult == nil &&
			len(dq.blockedProducers) <= initialBlockedProducerCount {
			// Requests without responses should be added to
			// blockedProducers.
			t.Errorf("%s: request should be added to blockedProducers",
				description)
		}
		// Check whether the frame was added to pendingFrames.
		var lastPendingFrame *segmentedFrame
		if len(dq.pendingFrames) != 0 {
			lastPendingFrame = &dq.pendingFrames[len(dq.pendingFrames)-1]
		}
		if test.expectedResult != nil && *test.expectedResult {
			// If the result is success, the frame should now be
			// enqueued.
			if lastPendingFrame == nil ||
				lastPendingFrame.frame != request.frame {
				t.Errorf("%s: frame should be added to pendingFrames",
					description)
			} else if lastPendingFrame.segment.id != test.expectedSegment {
				t.Errorf("%s: expected frame to be in segment %v, got %v",
					description, test.expectedSegment,
					lastPendingFrame.segment.id)
			}
			// Check that segments.nextID is one more than the segment that
			// was just assigned.
			if lastPendingFrame != nil &&
				dq.segments.nextID != test.expectedSegment+1 {
				t.Errorf("%s: expected segments.nextID to be %v, got %v",
					description, test.expectedSegment+1, dq.segments.nextID)
			}
		}
	}
}

func TestHandleWriterLoopResponse(t *testing.T) {
	// handleWriterLoopResponse should:
	// - Add the values in the bytesWritten array, in order, to the byteCount
	//   of the segments in segments.writing (these represent the amount
	//   written to each segment as a result of the preceding writer loop
	//   request).
	// - If bytesWritten covers more than one writing segment, then move
	//   all except the last one from segments.writing to segments.reading.
	// These invariants are relatively simple so this test is "by hand"
	// rather than using a structured list of sub-cases.

	dq := &diskQueue{
		settings: DefaultSettings(),
		segments: diskQueueSegments{
			writing: []*queueSegment{
				{id: 1, byteCount: 100},
				{id: 2},
				{id: 3},
				{id: 4},
			},
		},
	}

	// Write to one segment (no segments should be moved to reading list)
	dq.handleWriterLoopResponse(writerLoopResponse{
		segments: []writerLoopSegmentResponse{
			{bytesWritten: 100},
		},
	})
	if len(dq.segments.writing) != 4 || len(dq.segments.reading) != 0 {
		t.Fatalf("expected 4 writing and 0 reading segments, got %v writing "+
			"and %v reading", len(dq.segments.writing), len(dq.segments.reading))
	}
	if dq.segments.writing[0].byteCount != 200 {
		t.Errorf("expected first writing segment to be size 200, got %v",
			dq.segments.writing[0].byteCount)
	}

	// Write to two segments (the first one should be moved to reading list)
	dq.handleWriterLoopResponse(writerLoopResponse{
		segments: []writerLoopSegmentResponse{
			{bytesWritten: 100},
			{bytesWritten: 100},
		},
	})
	if len(dq.segments.writing) != 3 || len(dq.segments.reading) != 1 {
		t.Fatalf("expected 3 writing and 1 reading segments, got %v writing "+
			"and %v reading", len(dq.segments.writing), len(dq.segments.reading))
	}
	if dq.segments.reading[0].byteCount != 300 {
		t.Errorf("expected first reading segment to be size 300, got %v",
			dq.segments.reading[0].byteCount)
	}
	if dq.segments.writing[0].byteCount != 100 {
		t.Errorf("expected first writing segment to be size 100, got %v",
			dq.segments.writing[0].byteCount)
	}

	// Write to three segments (the first two should be moved to reading list)
	dq.handleWriterLoopResponse(writerLoopResponse{
		segments: []writerLoopSegmentResponse{
			{bytesWritten: 100},
			{bytesWritten: 100},
			{bytesWritten: 500},
		},
	})
	if len(dq.segments.writing) != 1 || len(dq.segments.reading) != 3 {
		t.Fatalf("expected 1 writing and 3 reading segments, got %v writing "+
			"and %v reading", len(dq.segments.writing), len(dq.segments.reading))
	}
	if dq.segments.reading[0].byteCount != 300 {
		t.Errorf("expected first reading segment to be size 300, got %v",
			dq.segments.reading[0].byteCount)
	}
	if dq.segments.reading[1].byteCount != 200 {
		t.Errorf("expected second reading segment to be size 200, got %v",
			dq.segments.reading[1].byteCount)
	}
	if dq.segments.reading[2].byteCount != 100 {
		t.Errorf("expected third reading segment to be size 100, got %v",
			dq.segments.reading[2].byteCount)
	}
	if dq.segments.writing[0].byteCount != 500 {
		t.Errorf("expected first writing segment to be size 500, got %v",
			dq.segments.writing[0].byteCount)
	}
}

func TestHandleReaderLoopResponse(t *testing.T) {
	// handleReaderLoopResponse should:
	// - advance segments.{nextReadFrameID, nextReadPosition} by the values in
	//   response.{frameCount, byteCount}
	// - advance the target segment's framesRead field by response.frameCount
	// - if there was an error reading the current segment, set
	//   nextReadPosition to the end of the segment.

	testCases := map[string]struct {
		// The segment structure to start with before calling
		// handleReaderLoopResponse.
		segments diskQueueSegments
		response readerLoopResponse

		expectedFrameID  frameID
		expectedPosition uint64
	}{
		"completely read first reading segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 10,
				byteCount:  1000,
			},
			expectedFrameID:  15,
			expectedPosition: 1000,
		},
		"read first half of first reading segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 5,
				byteCount:  500,
			},
			expectedFrameID:  10,
			expectedPosition: 500,
		},
		"read second half of first reading segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				nextReadFrameID:  5,
				nextReadPosition: 500,
			},
			response: readerLoopResponse{
				frameCount: 5,
				byteCount:  500,
			},
			expectedFrameID:  10,
			expectedPosition: 1000,
		},
		"read of first reading segment aborted by error": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 1,
				byteCount:  100,
				err:        fmt.Errorf("something bad happened"),
			},
			expectedFrameID:  6,
			expectedPosition: 1000,
		},
		"completely read first writing segment": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 10,
				byteCount:  1000,
			},
			expectedFrameID:  15,
			expectedPosition: 1000,
		},
		"read first half of first writing segment": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 5,
				byteCount:  500,
			},
			expectedFrameID:  10,
			expectedPosition: 500,
		},
		"read second half of first writing segment": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				nextReadPosition: 500,
				nextReadFrameID:  5,
			},
			response: readerLoopResponse{
				frameCount: 5,
				byteCount:  500,
			},
			expectedFrameID:  10,
			expectedPosition: 1000,
		},
		"error reading a writing segment skips remaining data": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				nextReadFrameID: 5,
			},
			response: readerLoopResponse{
				frameCount: 1,
				byteCount:  100,
				err:        fmt.Errorf("something bad happened"),
			},
			expectedFrameID:  6,
			expectedPosition: 1000,
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
		if dq.segments.nextReadPosition != test.expectedPosition {
			t.Errorf("%s: expected nextReadPosition = %d, got %d",
				description, test.expectedPosition, dq.segments.nextReadPosition)
		}
	}
}

func TestMaybeReadPending(t *testing.T) {
	// maybeReadPending should:
	// - If diskQueue.reading is true, do nothing and return immediately.
	// - If the first reading segment has already been completely read,
	//   move it to the acking list and set segments.nextReadPosition to 0.
	// - If nextReadPosition is / becomes 0, and a segment is available to
	//   read, set that segment's firstFrameID to segments.nextReadFrameID
	//   (so ACKs based on frame ID can be linked
	//   back to the segment that generated them), and set nextReadPosition
	//   to the end of the segment header.
	// - If there is unread data in the next available segment,
	//   send a readerLoopRequest for the full amount and set
	//   diskQueue.reading to true.

	testCases := map[string]struct {
		// The segment structure to start with before calling maybeReadPending
		segments diskQueueSegments
		// The value of the diskQueue.reading flag before calling maybeReadPending
		reading bool
		// The request we expect to see on the reader loop's request channel,
		// or nil if there should be none.
		expectedRequest *readerLoopRequest
		// The segment ID we expect to see in the acking list, or nil for none.
		expectedACKingSegment *segmentID
	}{
		"read one full segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				// The next read request should start with frame 5
				nextReadFrameID: 5,
			},
			expectedRequest: &readerLoopRequest{
				segment:      &queueSegment{id: 1},
				startFrameID: 5,
				// startPosition is 8, the end of the segment header in the
				// current file schema.
				startPosition: 8,
				endPosition:   1000,
			},
		},
		"do nothing if reading flag is set": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
			},
			reading:         true,
			expectedRequest: nil,
		},
		"read the end of a segment": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				// The next read request should start with frame 5
				nextReadFrameID: 5,
				// Start reading at position 500
				nextReadPosition: 500,
			},
			expectedRequest: &readerLoopRequest{
				segment:      &queueSegment{id: 1},
				startFrameID: 5,
				// Should be reading from nextReadPosition (500) to the end of
				// the segment (1000).
				startPosition: 500,
				endPosition:   1000,
			},
		},
		"ignore writing segments if reading is available": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				writing: []*queueSegment{
					{id: 2, byteCount: 1000},
				},
			},
			expectedRequest: &readerLoopRequest{
				segment:       &queueSegment{id: 1},
				startPosition: 8,
				endPosition:   1000,
			},
		},
		"do nothing if no segments are available": {
			segments:        diskQueueSegments{},
			expectedRequest: nil,
		},
		"read the writing segment if no reading segments are available": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 2, byteCount: 1000},
				},
				nextReadPosition: 500,
			},
			expectedRequest: &readerLoopRequest{
				segment:       &queueSegment{id: 2},
				startPosition: 500,
				endPosition:   1000,
			},
		},
		"do nothing if the writing segment has already been fully read": {
			segments: diskQueueSegments{
				writing: []*queueSegment{
					{id: 2, byteCount: 1000},
				},
				nextReadPosition: 1000,
			},
			expectedRequest: nil,
		},
		"skip the first reading segment if it's already been fully read": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
					{id: 2, byteCount: 500},
				},
				nextReadPosition: 1000,
			},
			expectedRequest: &readerLoopRequest{
				segment:       &queueSegment{id: 2},
				startPosition: 8,
				endPosition:   500,
			},
			expectedACKingSegment: segmentIDRef(1),
		},
		"move empty reading segment to the acking list if it's the only one": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{id: 1, byteCount: 1000},
				},
				nextReadPosition: 1000,
			},
			expectedRequest:       nil,
			expectedACKingSegment: segmentIDRef(1),
		},
		"reading the beginning of an old segment file uses the right header size": {
			segments: diskQueueSegments{
				reading: []*queueSegment{
					{
						id:            1,
						byteCount:     1000,
						schemaVersion: makeUint32Ptr(0)},
				},
				// The next read request should start with frame 5
				nextReadFrameID: 5,
			},
			expectedRequest: &readerLoopRequest{
				segment:      &queueSegment{id: 1},
				startFrameID: 5,
				// The header size for schema version 0 was 4 bytes.
				startPosition: 4,
				endPosition:   1000,
			},
		},
	}

	for description, test := range testCases {
		dq := &diskQueue{
			settings: DefaultSettings(),
			segments: test.segments,
			readerLoop: &readerLoop{
				requestChan: make(chan readerLoopRequest, 1),
			},
			reading: test.reading,
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
			if request.startPosition == 0 &&
				request.segment.firstFrameID != firstFrameID {
				t.Errorf(
					"%s: maybeReadPending should update firstFrameID", description)
			}
			if !dq.reading {
				t.Errorf(
					"%s: maybeReadPending should set the reading flag", description)
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
		} else if len(dq.segments.acking) != 0 {
			t.Errorf("%s: expected no acking segment, got %v",
				description, *dq.segments.acking[0])
		}
	}
}

func TestMaybeWritePending(t *testing.T) {
	// maybeWritePending should:
	// - If diskQueue.writing is true, do nothing and return immediately.
	// - Otherwise, if diskQueue.pendingFrames is nonempty:
	//   * send its contents as a writer loop request
	//   * set diskQueue.writeRequestSize to the total size of the
	//     request's frames
	//   * reset diskQueue.pendingFrames to nil
	//   * set diskQueue.writing to true.
	dq := &diskQueue{
		settings: DefaultSettings(),
		writerLoop: &writerLoop{
			requestChan: make(chan writerLoopRequest, 1),
		},
	}

	// First call: pendingFrames is empty, should do nothing.
	dq.maybeWritePending()
	select {
	case request := <-dq.writerLoop.requestChan:
		t.Errorf("expected no request on empty pendingFrames, got %v", request)
	default:
		if dq.writing {
			t.Errorf(
				"maybeWritePending shouldn't set writing flag without a request")
		}
	}

	// Set up some frame data for the remaining calls.
	pendingFrames := []segmentedFrame{
		{frame: makeWriteFrameWithSize(100)},
		{frame: makeWriteFrameWithSize(200)}}
	// The size on disk should be the summed buffer lengths plus
	// frameMetadataSize times the number of frames
	expectedSize := uint64(300)

	// Second call: writing is true, should do nothing.
	dq.pendingFrames = pendingFrames
	dq.writing = true
	dq.maybeWritePending()
	select {
	case request := <-dq.writerLoop.requestChan:
		t.Errorf("expected no request with writing flag set, got %v", request)
	default:
	}

	// Third call: writing is false, should send a request with pendingFrames.
	dq.writing = false
	dq.maybeWritePending()
	select {
	case request := <-dq.writerLoop.requestChan:
		// We are extra strict, because we can afford to be: the request should
		// contain not just the same elements, but the exact same array (slice)
		// as the previous value of pendingFrames.
		if len(request.frames) != len(pendingFrames) ||
			&request.frames[0] != &pendingFrames[0] {
			t.Errorf(
				"expected request containing pendingFrames, got a different array")
		}
		if dq.writeRequestSize != expectedSize {
			t.Errorf("expected writeRequestSize to equal %v, got %v",
				expectedSize, dq.writeRequestSize)
		}
		if len(dq.pendingFrames) != 0 {
			t.Errorf("pendingFrames should be reset after a write request")
		}
		if !dq.writing {
			t.Errorf("the writing flag should be set after a write request")
		}
	default:
	}
}

func TestMaybeUnblockProducers(t *testing.T) {
	// maybeUnblockProducers should:
	// - As long as diskQueue.blockedProducers is nonempty and the queue has
	//   capacity to add its first element (see TestCanAcceptFrameOfSize):
	//   * Add the request's frame to diskQueue.pendingFrames (see
	//     enqueueWriteFrame)
	//   * Report success (true) to the producer's response channel
	//   * Remove the request from blockedProducers
	// When complete, either blockedProducers should be empty or its first
	// element should be too big to add to the queue.

	settings := DefaultSettings()
	settings.MaxBufferSize = 1000
	responseChans := []chan bool{
		make(chan bool, 1), make(chan bool, 1), make(chan bool, 1)}
	dq := &diskQueue{
		settings: settings,
		segments: diskQueueSegments{
			writing: []*queueSegment{segmentWithSize(100)},
		},
		blockedProducers: []producerWriteRequest{
			{
				frame:        makeWriteFrameWithSize(200),
				responseChan: responseChans[0],
			},
			{
				frame:        makeWriteFrameWithSize(200),
				responseChan: responseChans[1],
			},
			{
				frame:        makeWriteFrameWithSize(501),
				responseChan: responseChans[2],
			},
		},
	}

	// First call: we expect two producers to be unblocked, because the third
	// one would push us one byte above the 1000 byte limit.
	dq.maybeUnblockProducers()
	if len(dq.pendingFrames) != 2 || len(dq.blockedProducers) != 1 {
		t.Fatalf("Expected 2 pending frames and 1 blocked producer, got %v and %v",
			len(dq.pendingFrames), len(dq.blockedProducers))
	}
	for i := 0; i < 3; i++ {
		select {
		case response := <-responseChans[i]:
			if i < 2 && !response {
				t.Errorf("Expected success response for producer %v, got failure", i)
			} else if i == 2 {
				t.Fatalf("Expected no response for producer 2, got %v", response)
			}
		default:
			if i < 2 {
				t.Errorf("Expected success response for producer %v, got none", i)
			}
		}
	}

	dq.blockedProducers[0].frame = makeWriteFrameWithSize(500)
	// Second call: with the blocked request one byte smaller, it should fit
	// into the queue, and be added with the other pending frames.
	dq.maybeUnblockProducers()
	if len(dq.pendingFrames) != 3 || len(dq.blockedProducers) != 0 {
		t.Fatalf("Expected 3 pending frames and 0 blocked producers, got %v and %v",
			len(dq.pendingFrames), len(dq.blockedProducers))
	}
	for i := 0; i < 3; i++ {
		// This time the first two response channels should get nothing and the
		// third should get success.
		select {
		case response := <-responseChans[i]:
			if i < 2 {
				t.Errorf("Expected no response for producer %v, got %v", i, response)
			} else if !response {
				t.Errorf("Expected success response for producer 2, got failure")
			}
		default:
			if i == 2 {
				t.Errorf("Expected success response for producer 2, got none")
			}
		}
	}
}

func TestCanAcceptFrameOfSize(t *testing.T) {
	// canAcceptFrameOfSize decides whether the queue has enough free capacity
	// to accept an incoming frame of the given size. It should:
	// - If the length of pendingFrames is >= settings.WriteAheadLimit,
	//   return false.
	// - If the queue size is unbounded (MaxBufferSize == 0), return true.
	// - Otherwise, return true iff the total size of the queue plus the new
	//   frame is <= settings.MaxBufferSize.
	//   The size of the queue is calculated as the summed size of:
	//   * All segments listed in diskQueue.segments (writing, reading, acking,
	//     acked)
	//   * All frames in diskQueue.pendingFrames (which have been accepted but
	//     not yet written)
	//   * If a write request is outstanding (diskQueue.writing == true),
	//     diskQueue.writeRequestSize, which is the size of the data that is
	//     being written by writerLoop but hasn't yet been completed.
	// All test cases are run with WriteAheadLimit = 2.

	testCases := map[string]struct {
		// The value of settings.MaxBufferSize in the test queue.
		maxBufferSize uint64
		// The value of the segments field in the test queue.
		segments diskQueueSegments
		// The value of pendingFrames in the test queue.
		pendingFrames []segmentedFrame
		// The value of writeRequestSize (the size of the most recent write
		// request) in the test queue.
		writeRequestSize uint64
		// The value of the writing flag in the test queue (writeRequestSize is
		// included in the queue size calculation only if there is an active
		// writing request).
		writing bool

		// If expectedOutcomes[v] = b then canAcceptFrameOfSize(v) should return b.
		expectedOutcomes map[uint64]bool
	}{
		"always reject when at the write ahead limit": {
			maxBufferSize: 1000,
			pendingFrames: []segmentedFrame{
				{frame: makeWriteFrameWithSize(10)},
				{frame: makeWriteFrameWithSize(10)},
			},
			expectedOutcomes: map[uint64]bool{10: false},
		},
		"always accept when queue size is unbounded": {
			maxBufferSize: 0,
			expectedOutcomes: map[uint64]bool{
				1: true, 1000: true, 1000000: true, 1000000000: true,
			},
		},
		// The remaining cases are all below the write ahead limit and have
		// bounded buffer size, we are just testing that the various
		// source values are all accounted for.
		"pendingFrames counts against buffer capacity": {
			maxBufferSize: 1000,
			pendingFrames: []segmentedFrame{
				{frame: makeWriteFrameWithSize(500)},
			},
			// There should be exactly 500 bytes capacity left
			expectedOutcomes: map[uint64]bool{
				500: true, 501: false,
			},
		},
		"diskQueue.segments counts against buffer capacity": {
			maxBufferSize: 1000,
			segments: diskQueueSegments{
				writing: []*queueSegment{segmentWithSize(100)},
				reading: []*queueSegment{segmentWithSize(100)},
				acking:  []*queueSegment{segmentWithSize(100)},
				acked:   []*queueSegment{segmentWithSize(100)},
			},
			// Four segments of size 100, should be exactly 600 bytes left
			expectedOutcomes: map[uint64]bool{
				600: true, 601: false,
			},
		},
		"writeRequestSize counts against buffer capacity when writing=true": {
			maxBufferSize:    1000,
			writeRequestSize: 600,
			writing:          true,
			expectedOutcomes: map[uint64]bool{
				400: true, 401: false,
			},
		},
		"writeRequestSize doesn't count against buffer capacity when writing=false": {
			maxBufferSize:    1000,
			writeRequestSize: 600,
			writing:          false,
			expectedOutcomes: map[uint64]bool{
				1000: true, 1001: false,
			},
		},
		"buffer capacity includes the sum of all sources": {
			// include all of them together.
			maxBufferSize: 1000,
			segments: diskQueueSegments{
				writing: []*queueSegment{segmentWithSize(100)},
				reading: []*queueSegment{segmentWithSize(100)},
				acking:  []*queueSegment{segmentWithSize(100)},
				acked:   []*queueSegment{segmentWithSize(100)},
			},
			pendingFrames: []segmentedFrame{
				{frame: makeWriteFrameWithSize(100)},
			},
			writeRequestSize: 200,
			writing:          true,
			expectedOutcomes: map[uint64]bool{
				300: true, 301: false,
			},
		},
	}

	for description, test := range testCases {
		settings := DefaultSettings()
		settings.WriteAheadLimit = 2
		settings.MaxBufferSize = test.maxBufferSize
		dq := &diskQueue{
			settings:         settings,
			segments:         test.segments,
			pendingFrames:    test.pendingFrames,
			writeRequestSize: test.writeRequestSize,
			writing:          test.writing,
		}
		for size, expected := range test.expectedOutcomes {
			result := dq.canAcceptFrameOfSize(size)
			if result != expected {
				t.Errorf("%v: expected canAcceptFrameOfSize(%v) = %v, got %v",
					description, size, expected, result)
			}
		}
	}
}

func boolRef(b bool) *bool {
	return &b
}

func segmentIDRef(id segmentID) *segmentID {
	return &id
}

// Convenience helper that creates a frame that will have the given size on
// disk after accounting for header / footer size.
func makeWriteFrameWithSize(size int) *writeFrame {
	if size <= frameMetadataSize {
		// Frames must have a nonempty data region.
		return nil
	}
	return &writeFrame{serialized: make([]byte, size-frameMetadataSize)}
}

func makeUint32Ptr(value uint32) *uint32 {
	return &value
}

func segmentWithSize(size int) *queueSegment {
	if size < segmentHeaderSize {
		// Can't have a segment smaller than the segment header
		return nil
	}
	return &queueSegment{byteCount: uint64(size)}
}

func equalReaderLoopRequests(
	r0 readerLoopRequest, r1 readerLoopRequest,
) bool {
	// We compare segment ids rather than segment pointers because it's
	// awkward to include the same pointer repeatedly in the test definition.
	return r0.startPosition == r1.startPosition &&
		r0.endPosition == r1.endPosition &&
		r0.segment.id == r1.segment.id &&
		r0.startFrameID == r1.startFrameID
}
