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

// The writer loop's job is to continuously read a data frame from the
// queue's intake channel, if there is one, and write it to disk.
func (dq *diskQueue) writerLoop() {
	// The open file handle for the segment currently being written.
	// This should be non-nil if and only if dq.segments.writing is.
	var file *os.File
	var filePosition int64
	var frameBuffer bytes.Buffer
	for {
		select {
		case frameContent := <-dq.inChan:
			frameBuffer = frameForContent(frameContent, dq.settings.ChecksumType)
		case <-dq.done:
			break
		}
		dq.segments.Lock()
		defer dq.segments.Unlock()
		// TODO: try to delete dq.segments.finished

		frameLen := uint64(frameBuffer.Len())
		if dq.segments.writing != nil &&
			dq.segments.writing.size+frameLen > dq.settings.MaxSegmentSize {
			// This segment is full. Close the file handle and move it to the
			// reading list.
			// TODO: make reasonable efforts to sync to disk.
			file.Close()
			dq.segments.reading = append(dq.segments.reading, dq.segments.writing)
			dq.segments.writing = nil
		}

		if dq.segments.writing == nil {
			// There is no active writing segment, create one.
			// TODO: (actually create one)
		}

		currentSize := dq.segments.sizeOnDiskWithLock()
		// TODO: block (releasing dq.segments) until
		// currentSize + newFrameSize <= dq.settings.MaxBufferSize

		// We now have a frame we want to write to disk, and enough free capacity
		// to write it.
		writeAll(file, frameBuffer.Bytes())
	}
}

func (segments *diskQueueSegments) sizeOnDiskWithLock() uint64 {
	total := uint64(0)
	if segments.writing != nil {
		total += segments.writing.size
	}
	for _, segment := range segments.reading {
		total += segment.size
	}
	for _, segment := range segments.waiting {
		total += segment.size
	}
	for _, segment := range segments.finished {
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
