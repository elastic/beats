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
	"os"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// queuePosition represents the position of a data frame within the queue: the
// containing segment, and a byte index into that segment on disk.
// It also stores the 0-based index of the current frame within its segment
// file. (Note that this depends only on the segment file itself, and is
// unrelated to the frameID type used to identify frames in memory.)
// The frame index is logically redundant with the byte index, but
// calculating it requires a linear scan of the segment file, so we store
// both values so we can track frame counts without reading the whole segment.
// When referencing a data frame, a byteIndex of 0 / uninitialized is
// understood to mean the first frame on disk (the header offset is
// added during handling); thus, `queuePosition{segmentID: 5}` always points
// to the first frame of segment 5, even though the logical position on
// disk depends on the header size, which can vary across schema version/s.
// However, a nonzero byteIndex is always interpreted as an exact
// file position.
type queuePosition struct {
	segmentID  segmentID
	byteIndex  uint64
	frameIndex uint64
}

type diskQueueACKs struct {
	logger *logp.Logger

	// This lock must be held to access diskQueueACKs fields (except for
	// diskQueueACKs.done, which is always safe).
	lock sync.Mutex

	// The id and position of the first unacknowledged frame.
	nextFrameID  frameID
	nextPosition queuePosition

	// If a frame has been ACKed, then frameSize[frameID] contains its size on
	// disk. The size is used to track the queuePosition of the oldest
	// remaining frame, which is written to disk as ACKs are received. (We do
	// this to avoid duplicating events if the beat terminates without a clean
	// shutdown.)
	frameSize map[frameID]uint64

	// segmentBoundaries maps the first frameID of each segment to its
	// corresponding segment ID.
	segmentBoundaries map[frameID]segmentID

	// When a segment has been completely acknowledged by a consumer, it sends
	// the segment ID to this channel, where it is read by the core loop and
	// scheduled for deletion.
	segmentACKChan chan segmentID

	// An open writable file handle to the file that stores the queue position.
	// This position is advanced as we receive ACKs, confirming it is safe
	// to move forward, so the acking code is responsible for updating this
	// file.
	positionFile *os.File

	// When the queue is closed, diskQueueACKs.done is closed to signal that
	// the core loop will not accept any more acked segments and any future
	// ACKs should be ignored.
	done chan struct{}
}

func newDiskQueueACKs(
	logger *logp.Logger, position queuePosition, positionFile *os.File,
) *diskQueueACKs {
	return &diskQueueACKs{
		logger:            logger,
		nextFrameID:       0,
		nextPosition:      position,
		frameSize:         make(map[frameID]uint64),
		segmentBoundaries: make(map[frameID]segmentID),
		segmentACKChan:    make(chan segmentID),
		positionFile:      positionFile,
		done:              make(chan struct{}),
	}
}

func (dqa *diskQueueACKs) addFrames(frames []*readFrame) {
	dqa.lock.Lock()
	defer dqa.lock.Unlock()
	select {
	case <-dqa.done:
		// We are already done and should ignore any leftover ACKs we receive.
		return
	default:
	}
	for _, frame := range frames {
		segment := frame.segment
		if frame.id != 0 && frame.id == segment.firstFrameID {
			// This is the first frame in its segment, mark it so we know when
			// we're starting a new segment.
			//
			// Subtlety: we don't count the very first frame as a "boundary" even
			// though it is the first frame we read from its segment. This prevents
			// us from resetting our segment offset to zero, in case the initial
			// offset was restored from a previous session instead of starting at
			// the beginning of the first file.
			dqa.segmentBoundaries[frame.id] = segment.id
		}
		dqa.frameSize[frame.id] = frame.bytesOnDisk
	}
	oldSegmentID := dqa.nextPosition.segmentID
	if dqa.frameSize[dqa.nextFrameID] != 0 {
		for ; dqa.frameSize[dqa.nextFrameID] != 0; dqa.nextFrameID++ {
			newSegment, ok := dqa.segmentBoundaries[dqa.nextFrameID]
			if ok {
				// This is the start of a new segment. Remove this frame from the
				// segment boundary list and reset the byte index to immediately
				// after the segment header.
				delete(dqa.segmentBoundaries, dqa.nextFrameID)
				dqa.nextPosition = queuePosition{
					segmentID:  newSegment,
					byteIndex:  segmentHeaderSize,
					frameIndex: 0,
				}
			}
			dqa.nextPosition.byteIndex += dqa.frameSize[dqa.nextFrameID]
			dqa.nextPosition.frameIndex++
			delete(dqa.frameSize, dqa.nextFrameID)
		}
		// We advanced the ACK position at least somewhat, so write its
		// new value.
		err := writeQueuePositionToHandle(dqa.positionFile, dqa.nextPosition)
		if err != nil {
			// TODO: Don't spam this warning on every ACK if it's a permanent error.
			dqa.logger.Warnf("Couldn't save queue position: %v", err)
		}
	}
	if oldSegmentID != dqa.nextPosition.segmentID {
		// We crossed at least one segment boundary, inform the listener that
		// everything before the current segment has been acknowledged (but bail
		// out if our done channel has been closed, since that means there is no
		// listener on the other end.)
		select {
		case dqa.segmentACKChan <- dqa.nextPosition.segmentID - 1:
		case <-dqa.done:
		}
	}
}
