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

type coreLoop struct {
	// The queue that created this coreLoop. The core loop is the only one of
	// the main goroutines for the queue that has a pointer to the queue and
	// understands its logic / structure.
	// Possible TODO: the split between fields in coreLoop and fields in
	// diskQueue seems artificial. Maybe the values here should be promoted
	// to diskQueue fields, and the core loop should just be a function on
	// diskQueue.
	queue *diskQueue

	// writing is true if a writeRequest is currently being processed by the
	// writer loop, false otherwise.
	writing bool

	// reading is true if the reader loop is processing a readBlock, false
	// otherwise.
	reading bool

	// deleting is true if the segment-deletion loop is processing a deletion
	// request, false otherwise.
	deleting bool

	// pendingWrites is a list of all write requests that have been accepted
	// by the queue and are waiting to be written to disk.
	pendingWrites []*writeRequest

	// blockedWrites is a list of all write requests that are waiting for
	// free space in the queue.
	blockedWrites []*writeRequest

	// This value represents the oldest frame ID for a segment that has not
	// yet been moved to the acked list. It is used to detect when the oldest
	// outstanding segment has been fully acknowledged by the consumer.
	oldestFrameID frameID
}

// A frame waiting to be written to disk
type writeFrame struct {
	// The original event provided by the client to diskQueueProducer.
	// We keep this as well as the serialized form until we are done
	// writing, because we may need to send this value back to the producer
	// callback if it is cancelled.
	event publisher.Event

	// The event, serialized for writing to disk and wrapped in a frame
	// header / footer.
	serialized []byte
}

// A frame that has been read from disk
type readFrame struct {
}

// A request sent from a producer to the core loop to add a frame to the queue.
type writeRequest struct {
	frame        *writeFrame
	shouldBlock  bool
	responseChan chan bool
}

type cancelRequest struct {
	producer *diskQueueProducer
	// If producer.config.DropOnCancel is true, then the core loop will respond
	// on responseChan with the number of dropped events.
	// Otherwise, this field may be nil.
	responseChan chan int
}

func (cl *coreLoop) run() {
	dq := cl.queue

	for {
		select {
		// Endpoints used by the public API
		case writeRequest := <-dq.writeRequestChan:
			cl.handleProducerWriteRequest(writeRequest)

		case cancelRequest := <-dq.cancelRequestChan:
			cl.handleProducerCancelRequest(cancelRequest)

		case ackedUpTo := <-dq.consumerAckChan:
			cl.handleConsumerAck(ackedUpTo)

		case <-dq.done:
			cl.handleShutdown()
			return

		// Writer loop handling
		case <-dq.writerLoop.finishedWriting:
			// Reset the writing flag and check if there's another frame waiting
			// to be written.
			cl.writing = false
			cl.maybeWritePending()

		// Reader loop handling
		case readResponse := <-dq.readerLoop.finishedReading:
			cl.handleReadResponse(readResponse)

		// Deleter loop handling
		case deleteResponse := <-dq.deleterLoop.response:
			cl.handleDeleteResponse(deleteResponse)
		}
	}
}

func (cl *coreLoop) handleProducerWriteRequest(request *writeRequest) {
	if len(cl.blockedWrites) > 0 {
		// If other requests are still waiting for space, then there
		// definitely isn't enough for this one.
		if request.shouldBlock {
			cl.blockedWrites = append(cl.blockedWrites, request)
		} else {
			// If the request is non-blocking, send immediate failure and discard it.
			request.responseChan <- false
		}
		return
	}
	// We will accept this request if there is enough capacity left in
	// the queue (after accounting for the pending writes that were
	// already accepted).
	pendingBytes := uint64(0)
	for _, request := range cl.pendingWrites {
		pendingBytes += uint64(len(request.frame.serialized))
	}
	currentSize := pendingBytes + cl.queue.segments.sizeOnDisk()
	frameSize := uint64(len(request.frame.serialized))
	if currentSize+frameSize > cl.queue.settings.MaxBufferSize {
		// The queue is too full. Either add the request to blockedWrites,
		// or send an immediate reject.
		if request.shouldBlock {
			cl.blockedWrites = append(cl.blockedWrites, request)
		} else {
			request.responseChan <- false
		}
	} else {
		// There is enough space for the new frame! Add it to the
		// pending list and dispatch it to the writer loop if no other
		// writes are outstanding.
		cl.pendingWrites = append(cl.pendingWrites, request)
		cl.maybeWritePending()
	}
}

func (cl *coreLoop) handleProducerCancelRequest(request *cancelRequest) {
}

func (cl *coreLoop) handleReadResponse(response readResponse) {

}

func (cl *coreLoop) handleConsumerAck(ackedUpTo frameID) {
	acking := cl.queue.segments.acking
	if len(acking) == 0 {
		return
	}
	segmentsAcked := 0
	startFrame := cl.oldestFrameID
	for ; segmentsAcked < len(acking); segmentsAcked++ {
		segment := acking[segmentsAcked]
		endFrame := startFrame + frameID(segment.framesRead)
		if endFrame > ackedUpTo {
			// This segment has not been fully read, we're done.
			break
		}
	}
	if segmentsAcked > 0 {
		// Move fully acked segments to the acked list and remove them
		// from the acking list.
		cl.queue.segments.acked =
			append(cl.queue.segments.acked, acking[:segmentsAcked]...)
		cl.queue.segments.acking = acking[segmentsAcked:]
		cl.maybeDeleteAcked()
	}
}

func (cl *coreLoop) handleDeleteResponse(response *deleteResponse) {
	dq := cl.queue
	cl.deleting = false
	if len(response.deleted) > 0 {
		// One or more segments were deleted, recompute the outstanding list.
		newAckedSegments := []*queueSegment{}
		for _, segment := range dq.segments.acked {
			if !response.deleted[segment] {
				// This segment wasn't deleted, so it goes in the new list.
				newAckedSegments = append(newAckedSegments, segment)
			}
		}
		dq.segments.acked = newAckedSegments
	}
	if len(response.errors) > 0 {
		dq.settings.Logger.Errorw("Couldn't delete old segment files",
			"errors", response.errors)
	}
	// If there are still files to delete, send the next request.
	cl.maybeDeleteAcked()
}

func (cl *coreLoop) handleShutdown() {
	// We need to close the input channels for all other goroutines and
	// wait for any outstanding responses. Order is important: handling
	// a read response may require the deleter, so the reader must be
	// shut down first.

	close(cl.queue.readerLoop.nextReadBlock)
	if cl.reading {
		response := <-cl.queue.readerLoop.finishedReading
		cl.handleReadResponse(response)
	}

	close(cl.queue.writerLoop.input)
	if cl.writing {
		<-cl.queue.writerLoop.finishedWriting
		cl.queue.segments.writing.writer.Close()
	}

	close(cl.queue.deleterLoop.input)
	if cl.deleting {
		response := <-cl.queue.deleterLoop.response
		// We can't retry any more if deletion failed, but we still check the
		// response so we can log any errors.
		if len(response.errors) > 0 {
			cl.queue.settings.Logger.Errorw("Couldn't delete old segment files",
				"errors", response.errors)
		}
	}

	// TODO: wait (with timeout?) for any outstanding acks?
}

// If the pendingWrites list is nonempty, and there are no outstanding
// requests to the writer loop, send the next frame.
func (cl *coreLoop) maybeWritePending() {
	dq := cl.queue
	if cl.writing || len(cl.pendingWrites) == 0 {
		// Nothing to do right now
		return
	}
	// We are now definitely going to handle the next request, so
	// remove it from pendingWrites.
	request := cl.pendingWrites[0]
	cl.pendingWrites = cl.pendingWrites[1:]

	// We have a frame to write, but we need to decide which segment
	// it should go in.
	segment := dq.segments.writing

	// If the new frame exceeds the maximum segment size, close the current
	// writing segment.
	frameLen := uint64(len(request.frame.serialized))
	if segment != nil && segment.size+frameLen > dq.settings.MaxSegmentSize {
		segment.writer.Close()
		segment.writer = nil
		dq.segments.reading = append(dq.segments.reading, segment)
		segment = nil
	}

	// If there is no active writing segment need to create a new segment.
	if segment == nil {
		segment = &queueSegment{
			id:            dq.segments.nextID,
			queueSettings: &dq.settings,
		}
		dq.segments.writing = segment
		dq.segments.nextID++
	}

	cl.queue.writerLoop.input <- &writeBlock{
		request: cl.pendingWrites[0],
		segment: segment,
	}
	cl.writing = true
}

// If the reading list is nonempty, and there are no outstanding read
// requests, send one.
func (cl *coreLoop) maybeReadPending() {

}

// If the acked list is nonempty, and there are no outstanding deletion
// requests, send one.
func (cl *coreLoop) maybeDeleteAcked() {
	if !cl.deleting && len(cl.queue.segments.acked) > 0 {
		cl.queue.deleterLoop.input <- &deleteRequest{segments: cl.queue.segments.acked}
		cl.deleting = true
	}
}
