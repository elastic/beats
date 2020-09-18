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

import "sync"

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

	// A map of all acked indices that are above ackedUpTo (and thus can't yet
	// be acknowledged as a continuous block).
	// TODO: do this better.
	//acked map[frameID]bool

	segments map[segmentID]segmentACKs

	// When a segment has been completely acknowledged by a consumer, it sends
	// the segment ID to this channel, where it is read by the core loop and
	// scheduled for deletion.
	segmentACKChan chan segmentID
}

// segmentACKs stores the ACKs for a single segment. If a frame has been
// ACKed, then segmentACKs[frameID] contains its size on disk. The size is
// used to track the queuePosition of the oldest remaining frame, which is
// written to disk as ACKs are received. (We do this to avoid duplicating
// events if the beat terminates without a clean shutdown.)
type segmentACKs map[frameID]int64

func (dqa *diskQueueACKs) addFrames(frames []*readFrame) {
	dqa.lock.Lock()
	defer dqa.lock.Unlock()
	for _, frame := range frames {
		segmentID := frame.segmentID
		segment := dqa.segments[segmentID]
		segment[frame.id] = frame.bytesOnDisk
		//dqa.acked[frame.id] = true
	}
	if dqa.acked[dqa.nextFrameID] {
		for ; dqa.acked[dqa.nextFrameID]; dqa.nextFrameID++ {
			delete(dqa.acked, dqa.nextFrameID)
		}
		// TODO: we now need to send the segment id, not the frame id.
		//dqa.segmentACKChan <- dqa.nextFrameID
	}
}
