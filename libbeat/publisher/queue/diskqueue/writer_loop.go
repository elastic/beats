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
// the containing segment it has been assigned to.
type segmentedFrame struct {
	// The frame to be written to disk.
	frame *writeFrame

	// The segment to which this frame should be written.
	segment *queueSegment
}

// One block for the writer loop consists of a write request and the
// segment it should be written to.
type writerLoopRequest struct {
	frames []segmentedFrame
}

// writerSegmentMetadata returns the actions taken by the writer loop
// on a single segment: the number of bytes written, and whether or
// not the segment was closed while handling the request (which signals
// to the core loop that the segment can be moved to segments.reading).
type writerSegmentMetadata struct {
	segment      *queueSegment
	bytesWritten int64
	closed       bool
}

// A writerLoopResponse reports the list of segments that have been
// completely written and can be moved to segments.reading.
// A segment is determined to have been completely written
// (and is then closed by the writer loop) when a frameWriteRequest
// targets a different segment than the previous ones.
type writerLoopResponse struct {
	segments []writerSegmentMetadata
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
		segments := wl.processRequest(block)
		wl.responseChan <- writerLoopResponse{segments: segments}
	}
}

// Write the given data to disk, returns the list of segments that were
// completed in the process.
func (wl *writerLoop) processRequest(request writerLoopRequest) []writerSegmentMetadata {
	var segments []writerSegmentMetadata
	bytesWritten := int64(0) // Bytes written to the current segment.
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
				segments = append(segments, writerSegmentMetadata{
					segment:      wl.currentSegment,
					bytesWritten: bytesWritten,
					closed:       true,
				})
				bytesWritten = 0
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
		bytesWritten += int64(n) + frameMetadataSize
	}
	if bytesWritten > 0 {
		segments = append(segments, writerSegmentMetadata{
			segment:      wl.currentSegment,
			bytesWritten: bytesWritten,
		})
	}
	return segments
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
