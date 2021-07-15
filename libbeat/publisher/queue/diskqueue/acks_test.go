package diskqueue

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestAddFrames(t *testing.T) {
	// If the done channel is closed, diskQueueACKs.addFrames
	// should do nothing and immediately return. Otherwise it should:
	// - add the sizes of all input frames to frameSize
	// - if any of the input frames are the first frame of their
	//   respective segment, add their segments to segmentBoundaries
	// - if the frame with id nextFrameID was among the inputs:
	//   * advance nextFrameID to the next remaining id that hasn't
	//     yet been passed into addFrames
	//   * remove any entries prior to the new nextFrameID from frameSize
	//   * advance nextPosition to the queuePosition for the new
	//     nextFrameID (calculated based on the contents of frameSize and
	//     segmentBoundaries)
	//   * write the new nextPosition to positionFile
	//   * if any segment boundaries are crossed while advancing
	//     nextFrameID, send their segmentID to segmentACKChan
	//     (notifying the core loop that these segments can be deleted)
	//     and remove them from segmentBoundaries.
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
	dqa := newDiskQueueACKs(
		logp.L(),
		queuePosition{
			segmentID:  2,
			byteIndex:  1000,
			frameIndex: 100,
		},
		stateFile)

	dqa.nextFrameID = 100
	segment3 := &queueSegment{
		id:           3,
		firstFrameID: 50,
	}
	segment4 := &queueSegment{
		id:           4,
		firstFrameID: 102,
	}
	segment5 := &queueSegment{
		id:           5,
		firstFrameID: 103,
	}
	frame100 := &readFrame{
		segment:     segment3,
		id:          100,
		bytesOnDisk: 500,
	}
	frame101 := &readFrame{
		segment:     segment4,
		id:          101,
		bytesOnDisk: 300,
	}
	frame102 := &readFrame{
		segment:     segment4,
		id:          102,
		bytesOnDisk: 100,
	}
	frame103 := &readFrame{
		segment:     segment5,
		id:          103,
		bytesOnDisk: 200,
	}

	dqa.addFrames([]*readFrame{frame101, frame102})
	if dqa.nextPosition.segmentID != 2 {
		t.Fatal("expected segment ID 2")
	}
	dqa.assertNoACKedSegment(t)

	dqa.addFrames([]*readFrame{frame100})
	if dqa.nextPosition.segmentID != 4 {
		t.Fatal("expected segment ID 4")
	}
	dqa.assertACKedSegment(t, 3)

	dqa.addFrames([]*readFrame{frame103})
	if dqa.nextPosition.segmentID != 5 {
		t.Fatalf("expected segment ID 5, got %v", dqa.nextPosition.segmentID)
	}
	dqa.assertACKedSegment(t, 4)
}

func (dqa *diskQueueACKs) assertNoACKedSegment(t *testing.T) {
	select {
	case seg := <-dqa.segmentACKChan:
		t.Fatalf("expected no segment ACKs, got %v", seg)
	default:
	}
}

func (dqa *diskQueueACKs) assertACKedSegment(t *testing.T, seg segmentID) {
	select {
	case received := <-dqa.segmentACKChan:
		if received != seg {
			t.Fatalf("expected ACK up to segment %v, got %v", seg, received)
		}
	default:
		t.Fatalf("expected ACK up to segment %v, got none", seg)
	}
}
