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

import "fmt"

// This file contains the queue's "core loop" -- the central goroutine
// that owns all queue state that is not encapsulated in one of the
// self-contained helper loops. This is the only file that is allowed to
// modify the queue state after its creation, and it contains the full
// logical "state transition diagram" for queue operation.

func (dq *diskQueue) run() {
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

			// The data that was just written is now available for reading, so check
			// if we should start a new read request.
			dq.maybeReadPending()

			// pendingFrames should now be empty. If any producers were blocked
			// because pendingFrames hit settings.WriteAheadLimit, wake them up.
			dq.maybeUnblockProducers()

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

		case metricsReq := <-dq.metricsRequestChan:
			dq.handleMetricsRequest(metricsReq)
		}
	}
}

// handleMetricsRequest responds to an event on the metricsRequestChan chan
func (dq *diskQueue) handleMetricsRequest(request metricsRequest) {
	resp := metricsRequestResponse{
		sizeOnDisk: dq.segments.sizeOnDisk(),
	}
	request.response <- resp
}

func (dq *diskQueue) handleProducerWriteRequest(request producerWriteRequest) {
	// Pathological case checking: make sure the incoming frame isn't bigger
	// than an entire segment all by itself (as long as it isn't, it is
	// guaranteed to eventually enter the queue assuming no disk errors).
	frameSize := request.frame.sizeOnDisk()
	if frameSize > dq.settings.maxValidFrameSize() {
		dq.logger.Warnf(
			"Rejecting event with size %v because the segment buffer limit is %v",
			frameSize, dq.settings.maxValidFrameSize())
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

func (dq *diskQueue) handleWriterLoopResponse(response writerLoopResponse) {
	dq.writing = false

	// The writer loop response contains the number of bytes written to
	// each segment that appeared in the request. Entries always appear in
	// the same sequence as (the beginning of) segments.writing.
	for index, segmentEntry := range response.segments {
		// Update the segment with its new size.
		dq.segments.writing[index].byteCount += segmentEntry.bytesWritten
		dq.segments.writing[index].frameCount += segmentEntry.framesWritten
	}

	// If there is more than one segment in the response, then all but the
	// last have been closed and are ready to move to the reading list.
	closedCount := len(response.segments) - 1
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
	dq.segments.nextReadPosition += response.byteCount

	segment := dq.segments.readingSegment()
	segment.framesRead += response.frameCount
	if response.err != nil {
		// If there's an error, we advance to the end of the current segment.
		// If the segment is in the reading list, it will be removed on the
		// next call to maybeReadPending.
		// If the segment is still in the writing list, we can't discard it
		// until the writer loop is done with it, but we can hope that advancing
		// to the current write position will get us out of our error state.
		dq.segments.nextReadPosition = segment.byteCount

		dq.logger.Errorf(
			"Error reading segment file %s: %v",
			dq.settings.segmentPath(segment.id), response.err)
	}
}

func (dq *diskQueue) handleDeleterLoopResponse(response deleterLoopResponse) {
	dq.deleting = false
	newAckedSegments := []*queueSegment{}
	errors := []error{}
	for i, err := range response.results {
		if err != nil {
			// This segment had an error, so it stays in the acked list.
			newAckedSegments = append(newAckedSegments, dq.segments.acked[i])
			errors = append(errors,
				fmt.Errorf("couldn't delete segment %d: %w",
					dq.segments.acked[i].id, err))
		}
	}
	if len(dq.segments.acked) > len(response.results) {
		// Preserve any new acked segments that were added during the deletion
		// request.
		tail := dq.segments.acked[len(response.results):]
		newAckedSegments = append(newAckedSegments, tail...)
	}
	dq.segments.acked = newAckedSegments
	if len(errors) > 0 {
		dq.logger.Errorw("deleting segment files", "errors", errors)
	}
}

func (dq *diskQueue) handleSegmentACK(ackedSegmentID segmentID) {
	acking := dq.segments.acking
	if len(acking) == 0 {
		return
	}
	ackedSegmentCount := 0
	for ; ackedSegmentCount < len(acking); ackedSegmentCount++ {
		if acking[ackedSegmentCount].id > ackedSegmentID {
			// This segment has not been acked yet, we're done.
			break
		}
	}
	if ackedSegmentCount > 0 {
		// Move fully acked segments to the acked list and remove them
		// from the acking list.
		dq.segments.acked =
			append(dq.segments.acked, acking[:ackedSegmentCount]...)
		dq.segments.acking = acking[ackedSegmentCount:]
	}
}

func (dq *diskQueue) handleShutdown() {
	// Shutdown: first, we wait for any outstanding requests to complete, to
	// make sure the helper loops are idle and all state is finalized, then
	// we do final cleanup and write our position to disk.

	// Close the reader loop's request channel to signal an abort in case it's
	// still processing a request (we don't need any more frames).
	// We still wait for acknowledgement afterwards: if there is a request in
	// progress, it's possible that a consumer already read and acknowledged
	// some of its data, so we want the final metadata before we write our
	// closing state.
	close(dq.readerLoop.requestChan)
	if dq.reading {
		response := <-dq.readerLoop.responseChan
		dq.handleReaderLoopResponse(response)
	}

	// We are assured by our callers within Beats that we will not be sent a
	// shutdown signal until all our producers have been finalized /
	// shut down -- thus, there should be no writer requests outstanding, and
	// writerLoop.requestChan should be idle. But just in case (and in
	// particular to handle the case where a request is stuck retrying a fatal
	// error), we signal abort by closing the request channel, and read the
	// final state if there is any.
	close(dq.writerLoop.requestChan)
	if dq.writing {
		response := <-dq.writerLoop.responseChan
		dq.handleWriterLoopResponse(response)
	}

	// We let the deleter loop finish its current request, but we don't send
	// the abort signal yet, since we might want to do one last deletion
	// after checking the final consumer ACK state.
	if dq.deleting {
		response := <-dq.deleterLoop.responseChan
		dq.handleDeleterLoopResponse(response)
	}

	// If there are any blocked producers still hoping for space to open up
	// in the queue, send them the bad news.
	for _, request := range dq.blockedProducers {
		request.responseChan <- false
	}
	dq.blockedProducers = nil

	// The reader and writer loops are now shut down, and the deleter loop is
	// idle. The remaining cleanup is in finalizing the read position in the
	// queue (the first event that hasn't been acknowledged by consumers), and
	// in deleting any older segment files that may be left.
	//
	// Events read by consumers have been accumulating their ACK data in
	// dq.acks. During regular operation the core loop is not allowed to use
	// this data, since it requires holding a mutex, but during shutdown we're
	// allowed to block to acquire it. However, we still must close its done
	// channel first, otherwise the lock may be held by a consumer that is
	// blocked trying to send us a message we're no longer listening to...
	close(dq.acks.done)
	dq.acks.lock.Lock()
	finalPosition := dq.acks.nextPosition
	// We won't be updating the position anymore, so we can close the file.
	_ = dq.acks.positionFile.Sync()
	dq.acks.positionFile.Close()
	dq.acks.lock.Unlock()

	// First check for the rare and fortunate case that every single event we
	// wrote to the queue was ACKed. In this case it is safe to delete
	// everything up to and including the current segment. Otherwise, we only
	// delete things before the current segment.
	if len(dq.segments.writing) > 0 &&
		finalPosition.segmentID == dq.segments.writing[0].id &&
		finalPosition.byteIndex >= dq.segments.writing[0].byteCount {
		dq.handleSegmentACK(finalPosition.segmentID)
	} else if finalPosition.segmentID > 0 {
		dq.handleSegmentACK(finalPosition.segmentID - 1)
	}

	// Do one last round of deletions, then shut down the deleter loop.
	dq.maybeDeleteACKed()
	if dq.deleting {
		response := <-dq.deleterLoop.responseChan
		dq.handleDeleterLoopResponse(response)
	}
	close(dq.deleterLoop.requestChan)
}

// If the pendingFrames list is nonempty, and there are no outstanding
// requests to the writer loop, send the next batch of frames.
func (dq *diskQueue) maybeWritePending() {
	if dq.writing || len(dq.pendingFrames) == 0 {
		// Nothing to do right now
		return
	}

	// Remove everything from pendingFrames and forward it to the writer loop.
	frames := dq.pendingFrames
	dq.pendingFrames = nil
	dq.writerLoop.requestChan <- writerLoopRequest{frames: frames}

	// Compute the size of the request so we know how full the queue is going
	// to be.
	totalSize := uint64(0)
	for _, sf := range frames {
		totalSize += sf.frame.sizeOnDisk()
	}
	dq.writeRequestSize = totalSize
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

// If the first entry of the reading list has been completely consumed,
// move it to the acking list and update the read position.
func (dq *diskQueue) maybeAdvanceReadingList() {
	if len(dq.segments.reading) > 0 {
		segment := dq.segments.reading[0]
		if dq.segments.nextReadPosition >= segment.byteCount {
			dq.segments.acking = append(dq.segments.acking, dq.segments.reading[0])
			dq.segments.reading = dq.segments.reading[1:]
			dq.segments.nextReadPosition = 0
		}
	}
}

// If the reading list is nonempty, and there are no outstanding read
// requests, send one.
func (dq *diskQueue) maybeReadPending() {
	if dq.reading {
		// A read request is already pending
		return
	}
	// If the current segment has already been completely read, move to
	// the next one.
	dq.maybeAdvanceReadingList()

	// Get the next available segment from the reading or writing lists.
	segment := dq.segments.readingSegment()
	if segment == nil ||
		dq.segments.nextReadPosition >= segment.byteCount {
		// Nothing to read
		return
	}
	if dq.segments.nextReadPosition == 0 {
		// If we're reading this segment for the first time, assign its
		// firstFrameID so we can recognize its acked frames later, and advance
		// the reading position to the end of the segment header.
		// The first segment we read might not have the initial nextReadPosition
		// set to 0 if it was already partially read on a previous run.
		// However that can only happen when nextReadFrameID == 0, so in that
		// case firstFrameID is already initialized to the correct value.
		segment.firstFrameID = dq.segments.nextReadFrameID
		dq.segments.nextReadPosition = segment.headerSize()
	}
	request := readerLoopRequest{
		segment:       segment,
		startFrameID:  dq.segments.nextReadFrameID,
		startPosition: dq.segments.nextReadPosition,
		endPosition:   segment.byteCount,
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
	newSegmentSize := dq.segments.writingSegmentSize + frame.sizeOnDisk()
	// If segment is nil, or the new segment exceeds its bounds,
	// we need to create a new writing segment.
	if segment == nil ||
		newSegmentSize > dq.settings.MaxSegmentSize {
		segment = &queueSegment{id: dq.segments.nextID}
		dq.segments.writing = append(dq.segments.writing, segment)
		dq.segments.nextID++
		// Reset the on-disk size to its initial value, the file's header size
		// with no frame data.
		newSegmentSize = segmentHeaderSize
	}

	dq.segments.writingSegmentSize = newSegmentSize
	dq.pendingFrames = append(dq.pendingFrames, segmentedFrame{
		frame:   frame,
		segment: segment,
	})
}

// canAcceptFrameOfSize checks whether there is enough free space in the queue
// (subject to settings.{MaxBufferSize, WriteAheadLimit}) to accept a new
// frame with the given size. Size includes both the serialized data and the
// frame header / footer; the easy way to do this for a writeFrame is to pass
// in frame.sizeOnDisk().
// Capacity calculations do not include requests in the blockedProducers
// list (that data is owned by its callers and we can't touch it until
// we are ready to respond). That allows this helper to be used both while
// handling producer requests and while deciding whether to unblock
// producers after free capacity increases.
func (dq *diskQueue) canAcceptFrameOfSize(frameSize uint64) bool {
	// If pendingFrames is already at the WriteAheadLimit, we can't accept
	// any new frames right now.
	if len(dq.pendingFrames) >= dq.settings.WriteAheadLimit {
		return false
	}

	// If the queue size is unbounded (max == 0), we accept.
	if dq.settings.MaxBufferSize == 0 {
		return true
	}

	// Compute the current queue size. We accept if there is enough capacity
	// left in the queue after accounting for the existing segments and the
	// pending writes that were already accepted.
	pendingBytes := uint64(0)
	for _, sf := range dq.pendingFrames {
		pendingBytes += sf.frame.sizeOnDisk()
	}
	// If a writing request is outstanding, include it in the size total.
	if dq.writing {
		pendingBytes += dq.writeRequestSize
	}
	currentSize := pendingBytes + dq.segments.sizeOnDisk()

	return currentSize+frameSize <= dq.settings.MaxBufferSize
}
