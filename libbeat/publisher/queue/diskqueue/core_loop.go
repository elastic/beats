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

// This file contains the queue's "core loop" -- the central goroutine
// that owns all queue state that is not encapsulated in one of the
// self-contained helper loops. This is the only file that is allowed to
// modify the queue state after its creation, and it contains the full
// logical "state transition diagram" for queue operation.

func (dq *diskQueue) run() {
	dq.logger.Debug("Core loop starting up...")

	// Wake up the reader and deleter loops if there are segments to process
	// from a previous instantiation of the queue.
	dq.maybeReadPending()
	dq.maybeDeleteACKed()

	for {
		select {
		// Endpoints used by the producer / consumer API implementation.
		case producerWriteRequest := <-dq.producerWriteRequestChan:
			dq.handleProducerWriteRequest(producerWriteRequest)

			// After a write request, there may be data ready to send to the
			// writer loop.
			dq.maybeWritePending()

		case cancelRequest := <-dq.producerCancelRequestChan:
			// TODO: this isn't really handled yet.
			dq.handleProducerCancelRequest(cancelRequest)

		case ackedSegmentID := <-dq.acks.segmentACKChan:
			dq.handleSegmentACK(ackedSegmentID)

			// After receiving new ACKs, a segment might be ready to delete.
			dq.maybeDeleteACKed()

		case <-dq.done:
			dq.handleShutdown()
			return

		// Writer loop handling
		case writerLoopResponse := <-dq.writerLoop.responseChan:
			dq.handleWriterLoopResponse(writerLoopResponse)

			// The writer loop completed a request, so check if there is more
			// data to be sent.
			dq.maybeWritePending()
			// We also check whether the reader loop is waiting for the data
			// that was just written.
			dq.maybeReadPending()

		// Reader loop handling
		case readerLoopResponse := <-dq.readerLoop.responseChan:
			dq.handleReaderLoopResponse(readerLoopResponse)

			// If there is more data to read, start a new read request.
			dq.maybeReadPending()

		// Deleter loop handling
		case deleterLoopResponse := <-dq.deleterLoop.responseChan:
			dq.handleDeleterLoopResponse(deleterLoopResponse)

			// If there are still files waiting to be deleted, send another request.
			dq.maybeDeleteACKed()

			// If there were blocked producers waiting for more queue space,
			// we might be able to unblock them now.
			dq.maybeUnblockProducers()
		}
	}
}

func (dq *diskQueue) handleProducerWriteRequest(request producerWriteRequest) {
	// Pathological case checking: make sure the incoming frame isn't bigger
	// than an entire segment all by itself (as long as it isn't, it is
	// guaranteed to eventually enter the queue assuming no disk errors).
	frameSize := request.frame.sizeOnDisk()
	if dq.settings.MaxSegmentSize < frameSize {
		dq.logger.Warnf(
			"Rejecting event with size %v because the maximum segment size is %v",
			frameSize, dq.settings.MaxSegmentSize)
		request.responseChan <- false
		return
	}

	// If no one else is blocked waiting for queue capacity, and there is
	// enough space, then we add the new frame and report success.
	// Otherwise, we either add to the end of blockedProducers to wait for
	// the requested space or report immediate failure, depending on the
	// producer settings.
	if len(dq.blockedProducers) == 0 && dq.canAcceptFrameOfSize(frameSize) {
		// There is enough space for the new frame! Add it to the
		// pending list and report success, then dispatch it to the
		// writer loop if no other requests are outstanding.
		dq.enqueueWriteFrame(request.frame)
		request.responseChan <- true
	} else {
		// The queue is too full. Either add the request to blockedProducers,
		// or send an immediate reject.
		if request.shouldBlock {
			dq.blockedProducers = append(dq.blockedProducers, request)
		} else {
			request.responseChan <- false
		}
	}
}

func (dq *diskQueue) handleProducerCancelRequest(
	request producerCancelRequest,
) {
	// TODO: implement me
}

func (dq *diskQueue) handleWriterLoopResponse(response writerLoopResponse) {
	dq.writing = false

	// The writer loop response contains the number of bytes written to
	// each segment that appeared in the request. Entries always appear in
	// the same sequence as (the beginning of) segments.writing.
	for index, bytesWritten := range response.bytesWritten {
		// Update the segment with its new size.
		dq.segments.writing[index].endOffset += segmentOffset(bytesWritten)
	}

	// If there is more than one segment in the response, then all but the
	// last have been closed and are ready to move to the reading list.
	closedCount := len(response.bytesWritten) - 1
	if closedCount > 0 {
		// Remove the prefix of the writing array and append to to reading.
		closedSegments := dq.segments.writing[:closedCount]
		dq.segments.writing = dq.segments.writing[closedCount:]
		dq.segments.reading =
			append(dq.segments.reading, closedSegments...)
	}
}

func (dq *diskQueue) handleReaderLoopResponse(response readerLoopResponse) {
	dq.reading = false

	// Advance the frame / offset based on what was just completed.
	dq.segments.nextReadFrameID += frameID(response.frameCount)
	dq.segments.nextReadOffset += segmentOffset(response.byteCount)

	var segment *queueSegment
	if len(dq.segments.reading) > 0 {
		// A segment is finished if we have read all the data, or
		// the read response reports an error.
		// Segments in the reading list have been completely written,
		// so we can rely on their endOffset field to determine their size.
		segment = dq.segments.reading[0]
		if dq.segments.nextReadOffset >= segment.endOffset || response.err != nil {
			dq.segments.reading = dq.segments.reading[1:]
			dq.segments.acking = append(dq.segments.acking, segment)
			dq.segments.nextReadOffset = 0
		}
	} else {
		// A segment in the writing list can't be finished writing,
		// so we don't check the endOffset.
		segment = dq.segments.writing[0]
	}
	segment.framesRead = int64(dq.segments.nextReadFrameID - segment.firstFrameID)

	// If there was an error, report it.
	if response.err != nil {
		dq.logger.Errorf(
			"Error reading segment file %s: %v",
			dq.settings.segmentPath(segment.id), response.err)
	}
}

func (dq *diskQueue) handleDeleterLoopResponse(response deleterLoopResponse) {
	dq.deleting = false
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
}

func (dq *diskQueue) handleSegmentACK(ackedSegmentID segmentID) {
	/*acking := dq.segments.acking
	if len(acking) == 0 {
		return
	}
	startFrame := dq.oldestFrameID
	endFrame := startFrame
	ackedSegmentCount := 0
	for ; ackedSegmentCount < len(acking); ackedSegmentCount++ {
		segment := acking[ackedSegmentCount]
		if endFrame+frameID(segment.framesRead) > ackedUpTo {
			// This segment is still waiting for acks, we're done.
			break
		}
		// Otherwise, advance the ending frame ID.
		endFrame += frameID(segment.framesRead)
	}
	if ackedSegmentCount > 0 {
		// Move fully acked segments to the acked list and remove them
		// from the acking list.
		dq.segments.acked =
			append(dq.segments.acked, acking[:ackedSegmentCount]...)
		dq.segments.acking = acking[ackedSegmentCount:]
		// Advance oldestFrameID past the segments we just removed.
		dq.oldestFrameID = endFrame
	}*/
}

func (dq *diskQueue) handleShutdown() {
	// We need to close the input channels for all other goroutines and
	// wait for any outstanding responses. Order is important: handling
	// a read response may require the deleter, so the reader must be
	// shut down first.

	close(dq.readerLoop.requestChan)
	if dq.reading {
		response := <-dq.readerLoop.responseChan
		dq.handleReaderLoopResponse(response)
	}

	close(dq.writerLoop.requestChan)
	if dq.writing {
		<-dq.writerLoop.responseChan
	}

	close(dq.deleterLoop.requestChan)
	if dq.deleting {
		response := <-dq.deleterLoop.responseChan
		// We can't retry any more if deletion failed, but we still check the
		// response so we can log any errors.
		if len(response.errors) > 0 {
			dq.logger.Errorw("Couldn't delete old segment files",
				"errors", response.errors)
		}
	}

	// TODO: wait (with timeout?) for any outstanding acks?

	// TODO: write final queue state to the metadata file.
}

// If the pendingFrames list is nonempty, and there are no outstanding
// requests to the writer loop, send the next batch of frames.
func (dq *diskQueue) maybeWritePending() {
	if dq.writing || len(dq.pendingFrames) == 0 {
		// Nothing to do right now
		return
	}
	// Remove everything from pendingFrames and forward it to the writer loop.
	requests := dq.pendingFrames
	dq.pendingFrames = nil

	dq.writerLoop.requestChan <- writerLoopRequest{
		frames: requests,
	}
	dq.writing = true
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
func (dq *diskQueue) maybeReadPending() {
	if dq.reading {
		// A read request is already pending
		return
	}
	segment := dq.segments.readingSegment()
	if segment == nil ||
		dq.segments.nextReadOffset >= segmentOffset(segment.endOffset) {
		// Nothing to read
		return
	}
	if dq.segments.nextReadOffset == 0 {
		// If we're reading the beginning of this segment, assign its firstFrameID.
		segment.firstFrameID = dq.segments.nextReadFrameID
	}
	request := readerLoopRequest{
		segment:      segment,
		startFrameID: dq.segments.nextReadFrameID,
		startOffset:  dq.segments.nextReadOffset,
		endOffset:    segment.endOffset,
	}
	dq.readerLoop.requestChan <- request
	dq.reading = true
}

// If the acked list is nonempty, and there are no outstanding deletion
// requests, send one.
func (dq *diskQueue) maybeDeleteACKed() {
	if !dq.deleting && len(dq.segments.acked) > 0 {
		dq.deleterLoop.requestChan <- deleterLoopRequest{
			segments: dq.segments.acked}
		dq.deleting = true
	}
}

// maybeUnblockProducers checks whether the queue has enough free space
// to accept any of the requests in the blockedProducers list, and if so
// accepts them in order and updates the list.
func (dq *diskQueue) maybeUnblockProducers() {
	unblockedCount := 0
	for _, request := range dq.blockedProducers {
		if !dq.canAcceptFrameOfSize(request.frame.sizeOnDisk()) {
			// Not enough space for this frame, we're done.
			break
		}
		// Add the frame to pendingFrames and report success.
		dq.enqueueWriteFrame(request.frame)
		request.responseChan <- true
		unblockedCount++
	}
	if unblockedCount > 0 {
		dq.blockedProducers = dq.blockedProducers[unblockedCount:]
	}
}

// enqueueWriteFrame determines which segment an incoming frame should be
// written to and adds the resulting segmentedFrame to pendingFrames.
func (dq *diskQueue) enqueueWriteFrame(frame *writeFrame) {
	// Start with the most recent writing segment if there is one.
	var segment *queueSegment
	if len(dq.segments.writing) > 0 {
		segment = dq.segments.writing[len(dq.segments.writing)-1]
	}
	frameLen := segmentOffset(frame.sizeOnDisk())
	// If segment is nil, or the new segment exceeds its bounds,
	// we need to create a new writing segment.
	if segment == nil ||
		dq.segments.nextWriteOffset+frameLen > dq.settings.maxSegmentOffset() {
		segment = &queueSegment{id: dq.segments.nextID}
		dq.segments.writing = append(dq.segments.writing, segment)
		dq.segments.nextID++
		dq.segments.nextWriteOffset = 0
	}

	dq.segments.nextWriteOffset += frameLen
	dq.pendingFrames = append(dq.pendingFrames, segmentedFrame{
		frame:   frame,
		segment: segment,
	})
}

// canAcceptFrameOfSize checks whether there is enough free space in the
// queue (subject to settings.MaxBufferSize) to accept a new frame with
// the given size. Size includes both the serialized data and the frame
// header / footer; the easy way to do this for a writeFrame is to pass
// in frame.sizeOnDisk().
// Capacity calculations do not include requests in the blockedProducers
// list (that data is owned by its callers and we can't touch it until
// we are ready to respond). That allows this helper to be used both while
// handling producer requests and while deciding whether to unblock
// producers after free capacity increases.
// If we decide to add limits on how many events / bytes can be stored
// in pendingFrames (to avoid unbounded memory use if the input is faster
// than the disk), this is the function to modify.
func (dq *diskQueue) canAcceptFrameOfSize(frameSize uint64) bool {
	if dq.settings.MaxBufferSize == 0 {
		// Currently we impose no limitations if the queue size is unbounded.
		return true
	}

	// Compute the current queue size. We accept if there is enough capacity
	// left in the queue after accounting for the existing segments and the
	// pending writes that were already accepted.
	pendingBytes := uint64(0)
	for _, request := range dq.pendingFrames {
		pendingBytes += request.frame.sizeOnDisk()
	}
	currentSize := pendingBytes + dq.segments.sizeOnDisk()

	return currentSize+frameSize <= dq.settings.MaxBufferSize
}
