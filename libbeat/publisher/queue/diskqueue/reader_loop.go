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
	"io"
)

type readResponse struct {
	// The number of frames read from the last file the reader loop was given.
	frameCount int

	// If there was an error in the segment file (i.e. inconsistent data), the
	// err field is set.
	err error
}

type readBlock struct {
	reader io.Reader
	length uint64
}

type readerLoop struct {
	// When there is a block available for reading, it will be sent to
	// nextReadBlock. When the reader loop has finished processing it, it
	// sends the result to finishedReading. If there is more than one block
	// available for reading, the core loop will wait until it gets a
	// finishedReadingMessage before it
	nextReadBlock   chan readBlock
	finishedReading chan readResponse

	// Frames that have been read from disk are sent to this channel.
	// Unlike most of the queue's API channels, this one is buffered to allow
	// the reader to read ahead and cache pending frames before a consumer
	// explicitly requests them.
	output chan *readFrame
}

func (rl *readerLoop) run() {
	for {
		block, ok := <-rl.nextReadBlock
		if !ok {
			// The channel has closed, we are shutting down.
			return
		}
		rl.finishedReading <- rl.processBlock(block)
	}
}

func (rl *readerLoop) processBlock(block readBlock) readResponse {
	frameCount := 0
	for {
		frame, err := block.nextFrame()
		if err != nil {
			return readResponse{
				frameCount: frameCount,
				err:        err,
			}
		}
		if frame == nil {
			// There are no more frames in this block.
			return readResponse{
				frameCount: frameCount,
				err:        nil,
			}
		}
		// We've read the frame, try sending it to the output channel.
		select {
		case rl.output <- frame:
			// Success! Increment the total for this block.
			frameCount++
		case <-rl.nextReadBlock:
			// Since we haven't sent a finishedReading message yet, we can only
			// reach this case when the nextReadBlock channel is closed, indicating
			// queue shutdown. In this case we immediately return.
			return readResponse{
				frameCount: frameCount,
				err:        nil,
			}
		}

		// If the output channel's buffer is not full, the previous select
		// might not recognize when the queue is being closed, so check that
		// again separately before we move on to the next data frame.
		select {
		case <-rl.nextReadBlock:
			return readResponse{
				frameCount: frameCount,
				err:        nil,
			}
		default:
		}
	}
}

func (block *readBlock) nextFrame() (*readFrame, error) {
	return nil, nil
}

/*func (dq *diskQueue) readerLoop() {
	curFrameID := frameID(0)
	logger := dq.settings.Logger.Named("readerLoop")
	for {
		dq.frameWrittenCond.Wait()
		reader, errs := dq.nextSegmentReader()
		for _, err := range errs {
			// Errors encountered while reading should be logged.
			logger.Error(err)
		}
		if reader == nil {
			// We couldn't find a readable segment, wait for a new
			// data frame to be written.
			dq.frameWrittenCond.Wait()
			if dq.closedForRead.Load() {
				// The queue has been closed, shut down.
				// TODO: cleanup (write the final read position)
				return
			}
			continue
		}
		// If we made it here, we have a nonempty reader and we want
		// to send all its frames to dq.outChan.
		framesRead := int64(0)
		for {
			bytes, err := reader.nextDataFrame()
			if err != nil {
				// An error at this stage generally means there has been
				// data corruption. For now, in this case we log the
				// error, then discard any remaining frames. When all
				// successfully read frames have been acknowledged, we
				// delete the underlying file.
				break
			}
			if bytes == nil {
				// If bytes is nil with no error, we've reached the end
				// of this segmentReader. Update the segment's frame count.
				break
			}
			framesRead++
			output := diskQueueOutput{
				data:    bytes,
				segment: reader.segment,
				frame:   curFrameID,
			}
			select {
			case dq.outChan <- output:
				curFrameID++
			case <-dq.done:
			}
		}
		reader.segment.framesRead += framesRead
	}
}*/
