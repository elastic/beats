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
	"encoding/binary"
	"os"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// A segmentedFrame is a data frame waiting to be written to disk along with
// the segment it has been assigned to.
type segmentedFrame struct {
	// The frame to be written to disk.
	frame *writeFrame

	// The segment to which this frame should be written.
	segment *queueSegment
}

// A writer loop request contains a list of writeFrames with the
// segment each should be written to.
//
// Input invariant: If a frame f is included in a writerLoopRequest, then
// every subsequent frame in this and future requests must have
// frame id at least f.segment.id.
//
// That is: we must write all frames for segment 0 before we start writing
// to frame 1, etc. This assumption allows all file operations to happen
// safely in the writer loop without any knowledge of the broader queue state.
type writerLoopRequest struct {
	frames []segmentedFrame
}

// A writerLoopResponse reports the number of bytes written to each
// segment in the request. There is guaranteed to be one entry for each
// segment that appeared in the request, in the same order. If there is
// more than one entry, then all but the last segment have been closed.
type writerLoopResponse struct {
	bytesWritten []int64
}

type writerLoop struct {
	// The settings for the queue that created this loop.
	settings *Settings

	// The logger for the writer loop, assigned when the queue creates it.
	logger *logp.Logger

	// The writer loop listens on requestChan for write blocks, and
	// writes them to disk immediately (all queue capacity checking etc. is
	// done by the core loop before sending it to the writer).
	requestChan chan writerLoopRequest

	// The writer loop sends to responseChan when it has finished handling a
	// request, to signal the core loop that it is ready for the next one.
	responseChan chan writerLoopResponse

	// The most recent segment that has been written to, if there is one.
	// This segment
	currentSegment *queueSegment

	// The file handle corresponding to currentSegment. When currentSegment
	// changes, this handle is closed and a new one is created.
	outputFile *os.File
}

func (wl *writerLoop) run() {
	wl.logger.Debug("Writer loop starting up...")
	for {
		block, ok := <-wl.requestChan
		if !ok {
			// The input channel is closed, we are done
			return
		}
		bytesWritten := wl.processRequest(block)
		wl.responseChan <- writerLoopResponse{bytesWritten: bytesWritten}
	}
}

// processRequest writes the frames in the given request to disk and returns
// the number of bytes written to each segment, in the order they were
// encountered.
func (wl *writerLoop) processRequest(request writerLoopRequest) []int64 {
	var bytesWritten []int64    // Bytes written to all segments.
	curBytesWritten := int64(0) // Bytes written to the current segment.
	for _, frameRequest := range request.frames {
		// If the new segment doesn't match the last one, we need to open a new
		// file handle and possibly clean up the old one.
		if wl.currentSegment != frameRequest.segment {
			wl.logger.Debugf(
				"Creating new segment file with id %v\n", frameRequest.segment.id)
			if wl.outputFile != nil {
				wl.outputFile.Close()
				wl.outputFile = nil
				// TODO: try to sync?
				// We are done with this segment, add the byte count to the list and
				// reset the current counter.
				bytesWritten = append(bytesWritten, curBytesWritten)
				curBytesWritten = 0
			}
			wl.currentSegment = frameRequest.segment
			file, err := wl.currentSegment.getWriter(wl.settings)
			if err != nil {
				wl.logger.Errorf("Couldn't open new segment file: %v", err)
				// TODO: retry, etc
			}
			wl.outputFile = file
		}

		// We have the data and a file to write it to. We are now committed
		// to writing this block unless the queue is closed in the meantime.
		frameSize := uint32(frameRequest.frame.sizeOnDisk())
		binary.Write(wl.outputFile, binary.LittleEndian, frameSize)
		// TODO: retry forever if there is an error or n isn't the right
		// length.
		n, err := wl.outputFile.Write(frameRequest.frame.serialized)
		if err != nil {
			wl.logger.Errorf("Couldn't write pending data to disk: %w", err)
		}
		// Compute / write the frame's checksum
		checksum := computeChecksum(
			frameRequest.frame.serialized, wl.settings.ChecksumType)
		binary.Write(wl.outputFile, binary.LittleEndian, checksum)
		// Write the frame footer's (duplicate) length
		binary.Write(wl.outputFile, binary.LittleEndian, frameSize)
		curBytesWritten += int64(n) + frameMetadataSize
	}
	// Return the total byte counts, including the final segment.
	return append(bytesWritten, curBytesWritten)
}

/*
func writeAll(writer io.Writer, p []byte) (int, error) {
	var N int
	for len(p) > 0 {
		n, err := writer.Write(p)
		N, p = N+n, p[n:]
		if err != nil && isRetryErr(err) {
			return N, err
		}
	}
	return N, nil
}

func isRetryErr(err error) bool {
	return err == syscall.EINTR || err == syscall.EAGAIN
}
*/
