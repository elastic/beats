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

	"github.com/elastic/beats/v8/libbeat/logp"
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
// disk depends on the header size, which can vary across schema versions.
// However, a nonzero byteIndex is always interpreted as an exact
// file position.
type queuePosition struct {
	segmentID  segmentID
	byteIndex  uint64
	frameIndex uint64
}

// diskQueueACKS stores the position of the oldest unacknowledged frame,
// synchronizing it to disk using the file handle in positionFile.
// sent to a consumer, and accepts later frames in any order, advancing
// the position as the oldest one is received. When the position changes,
// positionFile is overwritten with the new value.
//
// This is a simple way to track forward progress in the queue:
// if the application terminates before all frames have been acknowledged,
// the next session will restart at the first missing frame. This means
// some later frames may be transmitted twice in this case, but it
// guarantees that we don't drop any data.
//
// When all frames in a segment file have been acknowledged, the
// segment id is sent to diskQueueACKs.segmentACKChan (which is read
// and handled in core_loop.go) indicating that it is safe to dispose
// of that segment.
//
// diskQueueACKS detects that a segment has been completely acknowledged
// using the first frame ID of each segment as a boundary: if s is a
// queueSegment, and every frame before s.firstFrameID has been
// acknowledged, then every segment before s.id has been acknowledged.
// This means it can't detect the end of the final segment, so that case
// is handled diskQueue.handleShutdown.
type diskQueueACKs struct {
	logger *logp.Logger

	// This lock must be held to access diskQueueACKs fields (except for
	// diskQueueACKs.done, which is always safe).
	// This is needed because ACK handling happens in the same goroutine
	// as the caller (which may vary), unlike most queue logic which
	// happens in the core loop.
	lock sync.Mutex

	// The id and position of the first unacknowledged frame.
	nextFrameID  frameID
	nextPosition queuePosition

	// If a frame has been ACKed, then frameSize[frameID] contains its size on
	// disk. The size is used to track the queuePosition of the oldest
	// remaining frame, which is written to disk as ACKs are received. (We do
	// this to avoid duplicating events if the beat terminates without a clean
	// shutdown.)
	// Frames with id older than the oldest unacknowledged frame (nextFrameID)
	// are removed from the table.
	frameSize map[frameID]uint64

	// segmentBoundaries maps the first frameID of each segment to its
	// corresponding segment. We only need *queueSegment so we can
	// call queueSegment.headerSize to calculate our position on
	// disk; otherwise this could be a map from frameID to segmentID.
	segmentBoundaries map[frameID]*queueSegment

	// When a call to addFrames results in a segment being completely
	// acknowledged by a consumer, the highest segment ID that has been
	// completely acknowledged is sent to this channel, where the core loop
	// reads it and scheduled all segments up to that point for deletion.
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
		segmentBoundaries: make(map[frameID]*queueSegment),
		segmentACKChan:    make(chan segmentID, 1),
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
		if frame.id == segment.firstFrameID {
			// This is the first frame in its segment, mark it so we know when
			// we're starting a new segment.
			dqa.segmentBoundaries[frame.id] = segment
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
				if dqa.nextFrameID != 0 {
					// Special case if this is the first frame of a new session:
					// don't overwrite nextPosition, since it may contain the saved
					// position of the previous session.
					dqa.nextPosition = queuePosition{segmentID: newSegment.id}
				}
				if dqa.nextPosition.byteIndex == 0 {
					// Frame positions with byteIndex 0 are interpreted as pointing
					// to the first frame (see the definition of queuePosition).
					dqa.nextPosition.byteIndex = newSegment.headerSize()
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
