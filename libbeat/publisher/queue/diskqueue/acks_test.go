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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type addFramesTestStep struct {
	description      string
	input            []*readFrame
	expectedFrameID  frameID
	expectedPosition *queuePosition
	expectedSegment  *segmentID
}

type addFramesTest struct {
	frameID  frameID
	position queuePosition
	steps    []addFramesTestStep
}

func TestAddFrames(t *testing.T) {
	// If the done channel is closed, diskQueueACKs.addFrames
	// should do nothing and immediately return. Otherwise it should:
	// - add the sizes of all input frames to frameSize
	// - if any of the input frames are the first frame of their
	//   respective segment, add their segments to segmentBoundaries
	// - if the frame with id nextFrameID was among the inputs:
	//   * advance nextFrameID to the next remaining id that doesn't appear
	//     in frameSize, and nextPosition to its queuePosition (calculated
	//     based on the data in frameSize and segmentBoundaries)
	//   * write the new nextPosition to positionFile
	//   * remove any frames earlier than nextFrameID from frameSize
	//   * if we cross any segment boundaries, send the highest such segmentID
	//     to segmentACKChan (notifying the core loop that they can be
	//     deleted) and remove earlier segments from segmentBoundaries.

	testCases := map[string]addFramesTest{
		"2-segment test": {
			steps: []addFramesTestStep{
				{
					"Acknowledge first frame",
					[]*readFrame{
						rf(0, 0, true, 100),
					},
					frameID(1),
					&queuePosition{0, segmentHeaderSize + 100, 1},
					nil,
				},
				{
					"Acknowledge future frames from the second segment",
					[]*readFrame{
						rf(1, 3, true, 50),
						rf(1, 4, false, 100),
					},
					frameID(1),
					nil,
					nil,
				},
				{
					"Acknowledge second of three frames in the first segment",
					[]*readFrame{
						rf(0, 1, false, 75),
					},
					frameID(2),
					&queuePosition{0, segmentHeaderSize + 175, 2},
					nil,
				},
				{
					"Acknowledge last frame in first segment, unblocking the second",
					[]*readFrame{
						rf(0, 2, false, 100),
					},
					frameID(5),
					&queuePosition{1, segmentHeaderSize + 150, 2},
					// This time we crossed a boundary so we should get an ACK for segment
					// 0 on the notification channel.
					segmentIDRef(0),
				},
			},
		},
		"3 segments with 3 frames each": {
			steps: []addFramesTestStep{
				{
					"Acknowledge frames from segment 1 and 2",
					[]*readFrame{
						rf(1, 4, false, 100),
						rf(1, 5, false, 100),
						rf(2, 6, true, 100),
						rf(2, 8, false, 100),
					},
					frameID(0),
					&queuePosition{0, 0, 0},
					nil,
				},
				{
					"Acknowledge some of segment 0",
					[]*readFrame{
						rf(0, 1, false, 50),
						rf(0, 0, true, 100),
					},
					frameID(2),
					&queuePosition{0, segmentHeaderSize + 150, 2},
					nil,
				},
				{
					"Acknowledge the last frame of segment 0",
					[]*readFrame{
						rf(0, 2, false, 75),
					},
					frameID(3),
					&queuePosition{0, segmentHeaderSize + 225, 3},
					nil,
				},
				{
					"Acknowledge the first frame of segment 1",
					[]*readFrame{
						rf(1, 3, false, 100),
					},
					frameID(7),
					&queuePosition{2, segmentHeaderSize + 100, 1},
					segmentIDRef(1),
				},
				{
					"Acknowledge the middle frame of segment 2",
					[]*readFrame{
						rf(2, 7, false, 100),
					},
					frameID(9),
					&queuePosition{2, segmentHeaderSize + 300, 3},
					nil,
				},
			},
		},
		"ACKing multiple segments only sends the final one to the core loop": {
			steps: []addFramesTestStep{
				{
					"Add four one-frame segments",
					[]*readFrame{
						rf(3, 3, true, 100),
						rf(2, 2, true, 100),
						rf(1, 1, true, 100),
						rf(0, 0, true, 100),
					},
					frameID(4),
					&queuePosition{3, segmentHeaderSize + 100, 1},
					// We advanced from segment 0 to segment 3, so we expect
					// segmentID 2 on the ACK channel.
					segmentIDRef(2),
				},
			},
		},
		"The first frame of a segment resets nextPosition.byteIndex": {
			frameID:  35,
			position: queuePosition{9, 1000, 10},
			steps: []addFramesTestStep{
				{
					"Add the beginning of segment 10 as the next frame",
					[]*readFrame{
						rf(10, 35, true, 100),
					},
					frameID(36),
					&queuePosition{10, segmentHeaderSize + 100, 1},
					// We advanced to segment 10, so we expect segmentID 9 on
					// the ACK channel.
					segmentIDRef(9),
				},
			},
		},
		"The first frame after the queue opens doesn't overwrite nextPosition.byteIndex": {
			// Usually on the first frame of a segment, nextPosition is updated
			// to point to the beginning of the new segment. On the very first
			// frame of a new run, we don't do this, to allow the position to be
			// restored from a previous run in case we shut down partway through
			// a segment.
			frameID:  0,
			position: queuePosition{5, 1000, 10},
			steps: []addFramesTestStep{
				{
					"Add the beginning of segment 10 as the next frame",
					[]*readFrame{
						rf(5, 0, true, 100),
					},
					frameID(1),
					// The new 100-byte frame should just be added to the existing
					// byte position.
					&queuePosition{5, 1100, 11},
					nil,
				},
			},
		},
		"Segments with old schema versions have the correct positions": {
			steps: []addFramesTestStep{
				{
					"Add a frame for a schema-0 segment",
					[]*readFrame{
						{
							segment: &queueSegment{
								schemaVersion: uint32Ref(0),
							},
							bytesOnDisk: 100,
						},
					},
					frameID(1),
					// Schema-0 segments had 4-byte headers
					&queuePosition{0, 4 + 100, 1},
					nil,
				},
			},
		},
	}

	for name, test := range testCases {
		runAddFramesTest(t, name, test)
	}
}

func runAddFramesTest(t *testing.T, name string, test addFramesTest) {
	dir, err := ioutil.TempDir("", "diskqueue_acks_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "state.dat")
	stateFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer stateFile.Close()

	dqa := newDiskQueueACKs(logp.L(), test.position, stateFile)
	dqa.nextFrameID = test.frameID
	for _, step := range test.steps {
		prefix := fmt.Sprintf("[%v] %v", name, step.description)
		expectedPosition := dqa.nextPosition
		if step.expectedPosition != nil {
			expectedPosition = *step.expectedPosition
		}
		dqa.addFrames(step.input)
		if step.expectedFrameID != dqa.nextFrameID {
			t.Errorf("%v expected nextFrameID %v, got %v",
				prefix, step.expectedFrameID, dqa.nextFrameID)
			break
		}
		if expectedPosition != dqa.nextPosition {
			t.Errorf("%v expected nextPosition %v, got %v",
				prefix, step.expectedPosition, dqa.nextPosition)
			break
		}
		if step.expectedSegment == nil {
			dqa.assertNoACKedSegment(t, prefix)
		} else {
			dqa.assertACKedSegment(t, prefix, *step.expectedSegment)
		}
	}
}

func (dqa *diskQueueACKs) assertNoACKedSegment(t *testing.T, desc string) {
	select {
	case seg := <-dqa.segmentACKChan:
		t.Fatalf("%v expected no segment ACKs, got %v", desc, seg)
	default:
	}
}

func (dqa *diskQueueACKs) assertACKedSegment(
	t *testing.T, desc string, seg segmentID,
) {
	select {
	case received := <-dqa.segmentACKChan:
		if received != seg {
			t.Fatalf("%v expected ACK up to segment %v, got %v", desc, seg, received)
		}
	default:
		t.Fatalf("%v expected ACK up to segment %v, got none", desc, seg)
	}
}

func uint32Ref(v uint32) *uint32 {
	return &v
}

// rf assembles a readFrame with the given parameters and a spoofed
// queue segment, whose firstFrameID field is set to match the given frame
// if "first" is true.
func rf(seg segmentID, frame frameID, first bool, size uint64) *readFrame {
	s := &queueSegment{id: seg}
	if first {
		s.firstFrameID = frame
	}
	return &readFrame{
		segment:     s,
		id:          frame,
		bytesOnDisk: size,
	}
}
