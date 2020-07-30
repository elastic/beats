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
	"bytes"
	"io"
	"os"
	"syscall"
)

// frameForContent wraps the given content buffer in a
// frame header / footer and returns the resulting larger buffer.
func frameForContent(
	frameContent []byte, checksumType ChecksumType,
) bytes.Buffer {
	buf := bytes.Buffer{}
	//checksum := computeChecksum(frameContent, checksumType)
	/*buf
	frameLength := len(frameContent) + frameMetadataSize;
	frameBytes := make([]byte, frameLength)
	frameWriter :=
	binary.Write(reader.raw, binary.LittleEndian, &frameLength)*/
	return buf
}

type writerState struct {
	// The open file handle for the segment currently being written.
	// This should be non-nil if and only if diskQueue.segments.writing is.
	file         *os.File
	filePosition int64
}

func handleFrame(dq *diskQueue, state *writerState, frame bytes.Buffer) {
	dq.segments.Lock()
	defer dq.segments.Unlock()

	frameLen := uint64(frame.Len())
	// If there isn't enough space left in the current segment, close the
	// segment's file handle and move it to the reading list.
	if dq.segments.writing != nil &&
		dq.segments.writing.size+frameLen > dq.settings.MaxSegmentSize {
		// TODO: make reasonable efforts to sync to disk.
		state.file.Close()
		dq.segments.reading = append(dq.segments.reading, dq.segments.writing)
		dq.segments.writing = nil
	}

	if dq.segments.writing == nil {
		// There is no active writing segment, create one.
		// TODO: (actually create one)
	}

	// TODO: try to delete dq.segments.acked

	currentSize := dq.segments.sizeOnDiskWithLock()
	// Block (releasing the dq.segments lock) until
	// currentSize + frameLen <= dq.settings.MaxBufferSize
	for currentSize+frameLen > dq.settings.MaxBufferSize {
		// Wait for some space to be freed.
		dq.segments.segmentDeletedCond.Wait()
		if dq.closedForWrite.Load() {
			// The queue is closed, abort
		}
	}

	// We now have a frame we want to write to disk, and enough free capacity
	// to write it.
	writeAll(state.file, frame.Bytes())
}

// The writer loop's job is to continuously read a data frame from the
// queue's intake channel, if there is one, and write it to disk.
// It continues until the intake channel is empty or
func (dq *diskQueue) writerLoop() {
	defer dq.waitGroup.Done()
	//logger := dq.settings.Logger.Named("writerLoop")
	state := &writerState{}

	for {
		if dq.abort.Load() {
			// We are aborting, ignore any remaining buffered frames.
			return
		}
		select {
		case frameContent := <-dq.inChan:
			if frameContent == nil {
				// The channel has been drained, the writer loop should shut down.
				return
			}
			frameBuffer := frameForContent(frameContent, dq.settings.ChecksumType)
			handleFrame(dq, state, frameBuffer)
			if !dq.abort.Load() {
				// As long as we aren't aborting, continue processing any pending
				// frames.
				continue
			}
		case <-dq.done:
		}
		// We've processed
	}
}

// The number of bytes occupied by all the queue's segment files. Must
// be called with segments.Mutex held.
func (segments *diskQueueSegments) sizeOnDiskWithLock() uint64 {
	total := uint64(0)
	if segments.writing != nil {
		total += segments.writing.size
	}
	for _, segment := range segments.reading {
		total += segment.size
	}
	for _, segment := range segments.acking {
		total += segment.size
	}
	for _, segment := range segments.acked {
		total += segment.size
	}
	return total
}

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
