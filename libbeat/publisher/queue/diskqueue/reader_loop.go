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
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

type readerLoopRequest struct {
	segment     *queueSegment
	startOffset segmentOffset
	endOffset   segmentOffset
}

type readerLoopResponse struct {
	// The number of frames successfully read from the requested segment file.
	frameCount int64

	// The number of bytes successfully read from the requested segment file.
	byteCount int64

	// If there was an error in the segment file (i.e. inconsistent data), the
	// err field is set.
	err error
}

type readerLoop struct {
	// The settings for the queue that created this loop.
	settings *Settings

	// When there is a block available for reading, it will be sent to
	// requestChan. When the reader loop has finished processing it, it
	// sends the result to finishedReading. If there is more than one block
	// available for reading, the core loop will wait until it gets a
	// finishedReadingMessage before it
	requestChan  chan readerLoopRequest
	responseChan chan readerLoopResponse

	// Frames that have been read from disk are sent to this channel.
	// Unlike most of the queue's API channels, this one is buffered to allow
	// the reader to read ahead and cache pending frames before a consumer
	// explicitly requests them.
	output chan *readFrame

	// The helper object to deserialize binary blobs from the queue into
	// publisher.Event objects that can be returned in a readFrame.
	decoder *eventDecoder

	// The id that will be assigned to the next successfully-read frame.
	// Always starts from 0; this is just to track which frames have been
	// acknowledged, and doesn't need any consistency between runs.
	nextFrameID frameID
}

func (rl *readerLoop) run() {
	for {
		request, ok := <-rl.requestChan
		if !ok {
			// The channel has closed, we are shutting down.
			close(rl.output)
			return
		}
		response := rl.processRequest(request)
		fmt.Printf("\033[0;32mread response: read %d frames and %d bytes\033[0m\n", response.frameCount, response.byteCount)
		if response.err != nil {
			fmt.Printf("\033[0;32mresponse had err: %v\033[0m\n", response.err)
		}
		rl.responseChan <- rl.processRequest(request)
	}
}

func (rl *readerLoop) processRequest(request readerLoopRequest) readerLoopResponse {
	fmt.Printf("\033[0;32mprocessRequest(segment %d from %d to %d)\033[0m\n", request.segment.id, request.startOffset, request.endOffset)

	defer time.Sleep(time.Second)

	frameCount := int64(0)
	byteCount := int64(0)

	// Open the file and seek to the starting position.
	handle, err := request.segment.getReader(rl.settings)
	if err != nil {
		return readerLoopResponse{err: err}
	}
	defer handle.Close()
	_, err = handle.Seek(segmentHeaderSize+int64(request.startOffset), 0)
	if err != nil {
		return readerLoopResponse{err: err}
	}

	targetLength := int64(request.endOffset - request.startOffset)
	for {
		remainingLength := targetLength - byteCount

		// Try to read the next frame, clipping to the given bound.
		// If the next frame extends past this boundary, nextFrame will return
		// an error.
		frame, err := rl.nextFrame(handle, remainingLength)
		if frame != nil {
			// We've read the frame, try sending it to the output channel.
			select {
			case rl.output <- frame:
				// Success! Increment the total for this request.
				frameCount++
				byteCount += frame.bytesOnDisk
			case <-rl.requestChan:
				// Since we haven't sent a finishedReading message yet, we can only
				// reach this case when the nextReadBlock channel is closed, indicating
				// queue shutdown. In this case we immediately return.
				return readerLoopResponse{
					frameCount: frameCount,
					byteCount:  byteCount,
					err:        nil,
				}
			}
		}

		// We are done with this request if:
		// - there was an error reading the frame,
		// - there are no more frames to read, or
		// - we have reached the end of the requested region
		if err != nil || frame == nil || byteCount >= targetLength {
			return readerLoopResponse{
				frameCount: frameCount,
				byteCount:  byteCount,
				err:        err,
			}
		}

		// If the output channel's buffer is not full, the previous select
		// might not recognize when the queue is being closed, so check that
		// again separately before we move on to the next data frame.
		select {
		case <-rl.requestChan:
			return readerLoopResponse{
				frameCount: frameCount,
				byteCount:  byteCount,
				err:        nil,
			}
		default:
		}
	}
}

func (rl *readerLoop) nextFrame(
	handle *os.File, maxLength int64,
) (*readFrame, error) {
	// Ensure we are allowed to read the frame header.
	if maxLength < frameHeaderSize {
		return nil, fmt.Errorf(
			"Can't read next frame: remaining length %d is too low", maxLength)
	}
	// Wrap the handle to retry non-fatal errors and always return the full
	// requested data length if possible.
	reader := autoRetryReader{handle}
	var frameLength uint32
	err := binary.Read(reader, binary.LittleEndian, &frameLength)
	if err != nil {
		return nil, fmt.Errorf(
			"Couldn't read next frame length: %w", err)
	}

	// If the frame extends past the area we were told to read, return an error.
	// This should never happen unless the segment file is corrupted.
	if maxLength < int64(frameLength) {
		return nil, fmt.Errorf(
			"Can't read next frame: frame size is %v but remaining data is only %v",
			frameLength, maxLength)
	}
	if frameLength <= frameMetadataSize {
		// Valid enqueued data must have positive length
		return nil, fmt.Errorf(
			"Data frame with no data (length %d)", frameLength)
	}

	// Read the actual frame data
	dataLength := frameLength - frameMetadataSize
	bytes := rl.decoder.Buffer(int(dataLength))
	_, err = reader.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read data frame: %w", err)
	}

	// Read the footer (checksum + duplicate length)
	var checksum uint32
	err = binary.Read(reader, binary.LittleEndian, &checksum)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read data frame checksum: %w", err)
	}
	if rl.settings.ChecksumType != ChecksumTypeNone {
		expected := computeChecksum(bytes, rl.settings.ChecksumType)
		if checksum != expected {
			return nil, fmt.Errorf(
				"Data frame checksum mismatch (%x != %x)", checksum, expected)
		}
	}

	var duplicateLength uint32
	err = binary.Read(reader, binary.LittleEndian, &duplicateLength)
	if err != nil {
		return nil, fmt.Errorf(
			"Disk queue couldn't read trailing frame length: %w", err)
	}
	if duplicateLength != frameLength {
		return nil, fmt.Errorf(
			"Disk queue: inconsistent frame length (%d vs %d)",
			frameLength, duplicateLength)
	}

	event, err := rl.decoder.Decode()
	if err != nil {
		// TODO: Unlike errors in the segment or frame metadata, this is entirely
		// a problem in the event [de]serialization which may be isolated.
		// Rather than pass this error back to the read request, which discards
		// the rest of the segment, we should just log the error and advance to
		// the next frame, which is likely still valid.
		return nil, err
	}
	fmt.Printf("Decoded event from frame: %v\n", event)

	frame := &readFrame{
		event:       event,
		id:          rl.nextFrameID,
		bytesOnDisk: int64(frameLength),
	}
	rl.nextFrameID++

	return frame, nil
}

// A wrapper for an io.Reader that tries to read the full number of bytes
// requested, retrying on EAGAIN and EINTR, and returns an error if
// and only if the number of bytes read is less than requested.
// This is similar to io.ReadFull but with retrying.
type autoRetryReader struct {
	wrapped io.Reader
}

func (r autoRetryReader) Read(p []byte) (int, error) {
	bytesRead := 0
	reader := r.wrapped
	n, err := reader.Read(p)
	for n < len(p) {
		if err != nil && !readErrorIsRetriable(err) {
			return bytesRead + n, err
		}
		// If there is an error, it is retriable, so advance p and try again.
		bytesRead += n
		p = p[n:]
		n, err = reader.Read(p)
	}
	return bytesRead + n, nil
}

func readErrorIsRetriable(err error) bool {
	return errors.Is(err, syscall.EINTR) || errors.Is(err, syscall.EAGAIN)
}
