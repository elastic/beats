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
	"sync"
)

// queuePosition represents a logical position within the queue buffer.
type queuePosition struct {
	segmentID segmentID
	offset    segmentOffset
}

type diskQueueACKs struct {
	// This lock must be held to access this structure.
	lock sync.Mutex

	// The id and position of the first unacknowledged frame.
	nextFrameID  frameID
	nextPosition queuePosition

	// If a frame has been ACKed, then frameSize[frameID] contains its size on
	// disk. The size is used to track the queuePosition of the oldest
	// remaining frame, which is written to disk as ACKs are received. (We do
	// this to avoid duplicating events if the beat terminates without a clean
	// shutdown.)
	frameSize map[frameID]int64

	// segmentBoundaries maps the first frameID of each segment to its
	// corresponding segment ID.
	segmentBoundaries map[frameID]segmentID

	// When a segment has been completely acknowledged by a consumer, it sends
	// the segment ID to this channel, where it is read by the core loop and
	// scheduled for deletion.
	segmentACKChan chan segmentID
}

func (dqa *diskQueueACKs) addFrames(frames []*readFrame) {
	dqa.lock.Lock()
	defer dqa.lock.Unlock()
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
			// the beginning of the first segment.
			dqa.segmentBoundaries[frame.id] = segment.id
		}
		dqa.frameSize[frame.id] = frame.bytesOnDisk
	}
	if dqa.frameSize[dqa.nextFrameID] != 0 {
		for ; dqa.frameSize[dqa.nextFrameID] != 0; dqa.nextFrameID++ {
			newSegment, ok := dqa.segmentBoundaries[dqa.nextFrameID]
			if ok {
				// This is the start of a new segment, inform the ACK channel that
				// earlier segments are completely acknowledged.
				dqa.segmentACKChan <- newSegment - 1
				delete(dqa.segmentBoundaries, dqa.nextFrameID)

				// Update the position to the start of the new segment.
				dqa.nextPosition = queuePosition{
					segmentID: newSegment,
					offset:    0,
				}
			}
			dqa.nextPosition.offset += segmentOffset(dqa.frameSize[dqa.nextFrameID])
			delete(dqa.frameSize, dqa.nextFrameID)
		}
	}
}
