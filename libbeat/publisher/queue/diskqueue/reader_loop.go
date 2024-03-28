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
	"fmt"
	"io"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// startPosition and endPosition are absolute byte offsets into the segment
// file on disk, and must point to frame boundaries.
type readerLoopRequest struct {
	segment       *queueSegment
	startPosition uint64
	startFrameID  frameID
	endPosition   uint64
}

type readerLoopResponse struct {
	// The number of frames successfully read from the requested segment file.
	frameCount uint64

	// The number of bytes successfully read from the requested segment file.
	// If this is less than (endOffset - startOffset) from the original request,
	// then err is guaranteed to be non-nil.
	byteCount uint64

	// If there was an error in the segment file (i.e. inconsistent data), the
	// err field is set.
	err error
}

type readerLoop struct {
	// The settings for the queue that created this loop.
	settings Settings

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

	// If set, this encoding helper is called on events after loading
	// them from disk, to convert them to their final output serialization
	// format.
	outputEncoder queue.Encoder
}

func newReaderLoop(settings Settings, outputEncoder queue.Encoder) *readerLoop {
	return &readerLoop{
		settings: settings,

		requestChan:   make(chan readerLoopRequest, 1),
		responseChan:  make(chan readerLoopResponse),
		output:        make(chan *readFrame, settings.ReadAheadLimit),
		decoder:       newEventDecoder(),
		outputEncoder: outputEncoder,
	}
}

func (rl *readerLoop) run() {
	for {
		request, ok := <-rl.requestChan
		if !ok {
			// The channel is closed, we are shutting down.
			close(rl.output)
			return
		}
		response := rl.processRequest(request)
		rl.responseChan <- response
	}
}

func (rl *readerLoop) processRequest(request readerLoopRequest) readerLoopResponse {
	frameCount := uint64(0)
	byteCount := uint64(0)
	nextFrameID := request.startFrameID

	// Open the file and seek to the starting position.
	handle, err := request.segment.getReader(rl.settings)
	rl.decoder.serializationFormat = handle.serializationFormat
	if err != nil {
		return readerLoopResponse{err: err}
	}
	defer handle.Close()

	_, err = handle.Seek(int64(request.startPosition), io.SeekStart)
	if err != nil {
		return readerLoopResponse{err: err}
	}

	targetLength := request.endPosition - request.startPosition
	for {
		remainingLength := targetLength - byteCount

		// Try to read the next frame, clipping to the given bound.
		// If the next frame extends past this boundary, nextFrame will return
		// an error.
		frame, err := rl.nextFrame(handle, remainingLength)
		if frame != nil {
			// Add the segment / frame ID, which nextFrame leaves blank.
			frame.segment = request.segment
			frame.id = nextFrameID
			nextFrameID++
			// If an output encoder is configured, apply it now
			if rl.outputEncoder != nil {
				frame.event, _ = rl.outputEncoder.EncodeEntry(frame.event)
			}
			// We've read the frame, try sending it to the output channel.
			select {
			case rl.output <- frame:
				// Successfully sent! Increment the total for this request.
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
		// - there was an error reading the frame
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

// nextFrame reads and decodes one frame from the given file handle, as long
// it does not exceed the given length bound. The returned frame leaves the
// segment and frame IDs unset.
// The returned error will be set if and only if the returned frame is nil.
func (rl *readerLoop) nextFrame(handle *segmentReader, maxLength uint64) (*readFrame, error) {
	// Ensure we are allowed to read the frame header.
	if maxLength < frameHeaderSize {
		return nil, fmt.Errorf(
			"can't read next frame: remaining length %d is too low", maxLength)
	}
	// Wrap the handle to retry non-fatal errors and always return the full
	// requested data length if possible.
	reader := autoRetryReader{handle}
	var frameLength uint32
	err := binary.Read(reader, binary.LittleEndian, &frameLength)
	if err != nil {
		return nil, fmt.Errorf("couldn't read data frame header: %w", err)
	}

	// If the frame extends past the area we were told to read, return an error.
	// This should never happen unless the segment file is corrupted.
	if maxLength < uint64(frameLength) {
		return nil, fmt.Errorf(
			"can't read next frame: frame size is %d but remaining data is only %d",
			frameLength, maxLength)
	}
	if frameLength <= frameMetadataSize {
		// Valid enqueued data must have positive length
		return nil, fmt.Errorf(
			"data frame with no data (length %d)", frameLength)
	}

	// Read the actual frame data
	dataLength := frameLength - frameMetadataSize
	bytes := rl.decoder.Buffer(int(dataLength))
	_, err = reader.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("couldn't read data frame content: %w", err)
	}

	// Read the footer (checksum + duplicate length)
	var checksum uint32
	err = binary.Read(reader, binary.LittleEndian, &checksum)
	if err != nil {
		return nil, fmt.Errorf("couldn't read data frame checksum: %w", err)
	}
	expected := computeChecksum(bytes)
	if checksum != expected {
		return nil, fmt.Errorf(
			"data frame checksum mismatch (%x != %x)", checksum, expected)
	}

	var duplicateLength uint32
	err = binary.Read(reader, binary.LittleEndian, &duplicateLength)
	if err != nil {
		return nil, fmt.Errorf("couldn't read data frame footer: %w", err)
	}
	if duplicateLength != frameLength {
		return nil, fmt.Errorf(
			"inconsistent data frame length (%d vs %d)",
			frameLength, duplicateLength)
	}

	event, err := rl.decoder.Decode()
	if err != nil {
		// Unlike errors in the segment or frame metadata, this is entirely
		// a problem in the event [de]serialization which may be isolated (i.e.
		// may not indicate data corruption in the segment).
		// TODO: Rather than pass this error back to the read request, which
		// discards the rest of the segment, we should just log the error and
		// advance to the next frame, which is likely still valid.
		return nil, fmt.Errorf("couldn't decode data frame: %w", err)
	}

	frame := &readFrame{
		event:       event,
		bytesOnDisk: uint64(frameLength),
	}

	return frame, nil
}
