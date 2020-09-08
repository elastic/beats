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
	"github.com/elastic/beats/v7/libbeat/publisher"
)

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

	// nextReadOffset is the segment offset to start reading at during
	// the next read request. This offset always refers to the first read
	// segment: either segments.reading[0], if that list is nonempty, or
	// segments.writing[0] (if all segments have been read except the one
	// currently being written).
	nextReadOffset segmentOffset

	// nextWriteOffset is the segment offset at which the next new frame
	// should be written. This offset always refers to the last entry of
	// segments.writing. This is distinct from the endOffset field
	// within a segment: endOffset tracks how much data _has_ been
	// written to disk, while nextWriteOffset also includes all pending
	// frames that are scheduled to be written to disk.
	nextWriteOffset segmentOffset

	// pendingFrames is a list of all incoming data frames that have been
	// accepted by the queue and are waiting to be sent to the writer loop.
	pendingFrames []segmentedFrame

	// blockedProducers is a list of all producer write requests that are
	// waiting for free space in the queue.
	blockedProducers []producerWriteRequest

	// This value represents the oldest frame ID for a segment that has not
	// yet been moved to the acked list. It is used to detect when the oldest
	// outstanding segment has been fully acknowledged by the consumer.
	oldestFrameID frameID
}

// A data frame created through the producer API and waiting to be
// written to disk.
type writeFrame struct {
	// The original event provided by the client to diskQueueProducer.
	// We keep this as well as the serialized form until we are done
	// writing, because we may need to send this value back to the producer
	// callback if it is cancelled.
	event publisher.Event

	// The event, serialized for writing to disk and wrapped in a frame
	// header / footer.
	serialized []byte

	// The producer that created this frame. This is included in the
	// frame structure itself because we may need the producer and / or
	// its config at any time up until it has been completely written:
	// - While the core loop is tracking frames to send to the writer,
	//   it may receive a Cancel request, which requires us to know
	//   the producer / config each frame came from.
	// - After the writer loop has finished writing the frame to disk,
	//   it needs to call the ACK function specified in ProducerConfig.
	producer *diskQueueProducer
}

// A frame that has been read from disk
type readFrame struct {
	event publisher.Event
	id    frameID

	// How much space this frame occupied on disk (before deserialization),
	// including the frame header / footer.
	bytesOnDisk int64
}

func (cl *coreLoop) run() {
	cl.queue.logger.Debug("Core loop starting up...")
	dq := cl.queue

	// Wake up the reader loop if there are segments available to read.
	cl.maybeReadPending()

	for {
		select {
		// Endpoints used by the producer / consumer API implementation.
		case producerWriteRequest := <-dq.producerWriteRequestChan:
			cl.handleProducerWriteRequest(producerWriteRequest)

		case cancelRequest := <-dq.producerCancelRequestChan:
			cl.handleProducerCancelRequest(cancelRequest)

		case ackedUpTo := <-dq.consumerAckChan:
			cl.handleConsumerAck(ackedUpTo)

		case <-dq.done:
			cl.handleShutdown()
			return

		// Writer loop handling
		case writerLoopResponse := <-dq.writerLoop.responseChan:
			cl.handleWriterLoopResponse(writerLoopResponse)

		// Reader loop handling
		case readerLoopResponse := <-dq.readerLoop.responseChan:
			cl.handleReaderLoopResponse(readerLoopResponse)

		// Deleter loop handling
		case deleterLoopResponse := <-dq.deleterLoop.responseChan:
			cl.handleDeleterLoopResponse(deleterLoopResponse)
		}
	}
}

func (cl *coreLoop) handleProducerWriteRequest(request producerWriteRequest) {
	if len(cl.blockedProducers) > 0 {
		// If other requests are still waiting for space, then there
		// definitely isn't enough for this one.
		if request.shouldBlock {
			// Blocked writes don't get a response until there is enough free
			// space and they are moved to pendingWrites.
			cl.blockedProducers = append(cl.blockedProducers, request)
		} else {
			// If the request is non-blocking, send immediate failure and discard it.
			request.responseChan <- false
		}
		return
	}

	// Pathological case checking: make sure the incoming frame isn't bigger
	// than an entire segment all by itself (as long as it isn't, it is
	// guaranteed to eventually enter the queue assuming no disk errors).
	frameSize := uint64(len(request.frame.serialized))
	if cl.queue.settings.MaxSegmentSize < frameSize {
		cl.queue.logger.Warnf(
			"Rejecting event with size %v because the maximum segment size is %v",
			frameSize, cl.queue.settings.MaxSegmentSize)
		request.responseChan <- false
		return
	}

	// We will accept this request if there is enough capacity left in
	// the queue (after accounting for the pending writes that were
	// already accepted).
	pendingBytes := uint64(0)
	for _, request := range cl.pendingFrames {
		pendingBytes += uint64(len(request.frame.serialized))
	}
	currentSize := pendingBytes + cl.queue.segments.sizeOnDisk()
	// cl.queue.logger.Debugf(
	// 	"currentSize: %v  frameSize: %v  MaxBufferSize: %v",
	// 	currentSize, frameSize, cl.queue.settings.MaxBufferSize)
	if cl.queue.settings.MaxBufferSize > 0 &&
		currentSize+frameSize > cl.queue.settings.MaxBufferSize {
		// The queue is too full. Either add the request to blockedWrites,
		// or send an immediate reject.
		if request.shouldBlock {
			cl.blockedProducers = append(cl.blockedProducers, request)
		} else {
			request.responseChan <- false
		}
	} else {
		// There is enough space for the new frame! Add it to the
		// pending list and report success, then dispatch it to the
		// writer loop if no other requests are outstanding.
		// Right now we accept any request if there is enough space for it
		// on disk. High-throughput inputs may produce events faster than
		// they can be written to disk, so it would make sense to
		// additionally bound the amount of data in pendingWrites to some
		// configurable limit to avoid out-of-memory errors.
		cl.enqueueProducerFrame(request.frame)
		request.responseChan <- true
		cl.maybeWritePending()
	}
}

func (cl *coreLoop) handleProducerCancelRequest(request producerCancelRequest) {
}

func (cl *coreLoop) handleWriterLoopResponse(response writerLoopResponse) {
	cl.writing = false

	for _, metadata := range response.segments {
		// Update the segments with their new size and, if the writer
		// closed them, move them to the reading list.
		// TODO: i don't like this. It is redundant, and in brittle ways:
		// segments are always written and closed in strict order, and the
		// core loop knows what that order is, but we let the writer loop
		// report them as independent parameters and then depend on those
		// instead? It works for the moment but needs to be fixed soon.
		metadata.segment.endOffset += segmentOffset(metadata.bytesWritten)
		if metadata.closed {
			cl.queue.segments.writing = cl.queue.segments.writing[1:]
			cl.queue.segments.reading =
				append(cl.queue.segments.reading, metadata.segment)
		}
	}

	// New data is available, so we also check if we should send a new
	// read request.
	cl.maybeReadPending()
	cl.maybeWritePending()
}

func (cl *coreLoop) handleReaderLoopResponse(response readerLoopResponse) {
	cl.reading = false
	segments := cl.queue.segments

	// Advance the read offset based on what was just completed.
	cl.nextReadOffset += segmentOffset(response.byteCount)

	var segment *queueSegment
	if len(segments.reading) > 0 {
		// A segment is finished if we have read all the data, or
		// the read response reports an error.
		// Segments in the reading list have been completely written,
		// so we can rely on their endOffset field to determine the
		// size of the data.
		segment = segments.reading[0]
		if cl.nextReadOffset >= segment.endOffset || response.err != nil {
			segments.reading = segments.reading[1:]
			segments.acking = append(segments.acking, segment)
			cl.nextReadOffset = 0
		}
	} else {
		// A segment in the writing list can't be finished writing,
		// so we don't check the endOffset.
		segment = segments.writing[0]
	}
	segment.framesRead += response.frameCount

	// If there is more data to read, start a new read request.
	cl.maybeReadPending()
}

func (cl *coreLoop) handleDeleterLoopResponse(response deleterLoopResponse) {
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
		dq.logger.Errorw("Couldn't delete old segment files",
			"errors", response.errors)
	}
	// If there are still files to delete, send the next request.
	cl.maybeDeleteAcked()

	// If there are blocked producers waiting for more queue space, this
	// deletion might have unblocked them.
	cl.maybeUnblockProducers()
}

func (cl *coreLoop) handleConsumerAck(ackedUpTo frameID) {
	acking := cl.queue.segments.acking
	if len(acking) == 0 {
		return
	}
	startFrame := cl.oldestFrameID
	endFrame := startFrame
	ackedSegmentCount := 0
	for ; ackedSegmentCount < len(acking); ackedSegmentCount++ {
		segment := acking[ackedSegmentCount]
		endFrame += frameID(segment.framesRead)
		if endFrame > ackedUpTo {
			// This segment is still waiting for acks, we're done.
			break
		}
	}
	if ackedSegmentCount > 0 {
		// Move fully acked segments to the acked list and remove them
		// from the acking list.
		cl.queue.segments.acked =
			append(cl.queue.segments.acked, acking[:ackedSegmentCount]...)
		cl.queue.segments.acking = acking[ackedSegmentCount:]
		cl.oldestFrameID = endFrame
		cl.maybeDeleteAcked()
	}
}

func (cl *coreLoop) handleShutdown() {
	// We need to close the input channels for all other goroutines and
	// wait for any outstanding responses. Order is important: handling
	// a read response may require the deleter, so the reader must be
	// shut down first.

	close(cl.queue.readerLoop.requestChan)
	if cl.reading {
		response := <-cl.queue.readerLoop.responseChan
		cl.handleReaderLoopResponse(response)
	}

	close(cl.queue.writerLoop.requestChan)
	if cl.writing {
		<-cl.queue.writerLoop.responseChan
		//cl.queue.segments.writing.writer.Close()
	}

	close(cl.queue.deleterLoop.requestChan)
	if cl.deleting {
		response := <-cl.queue.deleterLoop.responseChan
		// We can't retry any more if deletion failed, but we still check the
		// response so we can log any errors.
		if len(response.errors) > 0 {
			cl.queue.logger.Errorw("Couldn't delete old segment files",
				"errors", response.errors)
		}
	}

	// TODO: wait (with timeout?) for any outstanding acks?

	// TODO: write final queue state to the metadata file.
}

// If the pendingWrites list is nonempty, and there are no outstanding
// requests to the writer loop, send the next batch of frames.
func (cl *coreLoop) maybeWritePending() {
	if cl.writing || len(cl.pendingFrames) == 0 {
		// Nothing to do right now
		return
	}
	// Remove everything from pendingWrites and forward it to the writer loop.
	requests := cl.pendingFrames
	cl.pendingFrames = nil

	cl.queue.writerLoop.requestChan <- writerLoopRequest{
		frames: requests,
	}
	cl.writing = true
}

// Returns the active read segment, or nil if there is none.
func (segments *diskQueueSegments) readingSegment() *queueSegment {
	if len(segments.reading) > 0 {
		return segments.reading[0]
	}
	if len(segments.writing) > 0 {
		return segments.writing[0]
	}
	return nil
}

// If the reading list is nonempty, and there are no outstanding read
// requests, send one.
func (cl *coreLoop) maybeReadPending() {
	if cl.reading {
		// A read request is already pending
		return
	}
	segment := cl.queue.segments.readingSegment()
	if segment == nil || cl.nextReadOffset >= segmentOffset(segment.endOffset) {
		// Nothing to read
		return
	}
	request := readerLoopRequest{
		segment:     segment,
		startOffset: cl.nextReadOffset,
		endOffset:   segment.endOffset,
	}
	cl.queue.readerLoop.requestChan <- request
	cl.reading = true
}

// If the acked list is nonempty, and there are no outstanding deletion
// requests, send one.
func (cl *coreLoop) maybeDeleteAcked() {
	if !cl.deleting && len(cl.queue.segments.acked) > 0 {
		cl.queue.deleterLoop.requestChan <- deleterLoopRequest{segments: cl.queue.segments.acked}
		cl.deleting = true
	}
}

func (cl *coreLoop) maybeUnblockProducers() {
	// TODO: implement me
}

// enqueueProducerFrame determines which segment an incoming frame should be
// written to and adds the result to pendingWrites.
func (cl *coreLoop) enqueueProducerFrame(frame *writeFrame) {
	dq := cl.queue

	// Start with the most recent writing segment if there is one.
	var segment *queueSegment
	if len(dq.segments.writing) > 0 {
		segment = dq.segments.writing[len(dq.segments.writing)-1]
	}
	frameLen := segmentOffset(len(frame.serialized))
	// If segment is nil, or the new segment exceeds its bounds,
	// we need to create a new writing segment.
	if segment == nil ||
		cl.nextWriteOffset+frameLen > dq.settings.maxSegmentOffset() {
		segment = &queueSegment{id: dq.segments.nextID}
		dq.segments.writing = append(dq.segments.writing, segment)
		dq.segments.nextID++
		cl.nextWriteOffset = 0
	}

	cl.nextWriteOffset += frameLen
	cl.pendingFrames = append(cl.pendingFrames, segmentedFrame{
		frame:   frame,
		segment: segment,
	})
}
