package diskqueue

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
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

	wrapper := wrapACKChan(dqa.segmentACKChan)
	defer wrapper.close()

	dqa.nextFrameID = 100
	segment0 := &queueSegment{
		id:           3,
		firstFrameID: 100,
	}
	segment1 := &queueSegment{
		id:           4,
		firstFrameID: 125,
	}
	frame0 := &readFrame{
		segment:     segment0,
		id:          100,
		bytesOnDisk: 500,
	}
	frame1 := &readFrame{
		segment:     segment1,
		id:          101,
		bytesOnDisk: 300,
	}
	if dqa.nextPosition.segmentID != 2 {
		t.Fatal("expected segment ID 2")
	}
	dqa.addFrames([]*readFrame{frame1})
	dqa.addFrames([]*readFrame{frame0})
	if dqa.nextPosition.segmentID != 3 {
		t.Fatal("expected segment ID 3")
	}
	if !wrapper.matchesSegments([]segmentID{3}) {
		t.Fatalf("")
	}

	//t.Fatal(fmt.Errorf("hello there"))
}

// A wrapper that listens to diskQueueACKs.segmentACKChan on an auxiliary
// goroutine so that addFrames can be tested without blocking.
// Callers should call ackChanWrapper.close() during to terminate the
// background goroutine and synchronize the data before checking test
// results.
type ackChanWrapper struct {
	ch        chan segmentID
	seen      map[segmentID]bool
	seenCount int
	wg        sync.WaitGroup
}

func wrapACKChan(ch chan segmentID) *ackChanWrapper {
	wrapper := &ackChanWrapper{ch: ch, seen: make(map[segmentID]bool)}
	wrapper.wg.Add(1)
	go wrapper.run()
	return wrapper
}

func (w *ackChanWrapper) run() {
	defer w.wg.Done()
	for seg := range w.ch {
		fmt.Printf("saw segment %v\n", seg)
		w.seen[seg] = true
		w.seenCount++
	}
}

func (w *ackChanWrapper) close() {
	close(w.ch)
	w.wg.Wait()
}

func (w *ackChanWrapper) matchesSegments(segments []segmentID) bool {
	if w.seenCount != len(segments) {
		return false
	}
	for _, seg := range segments {
		if !w.seen[seg] {
			return false
		}
	}
	return true
}

/*func (w *ackChanWrapper) seenSegments() []segmentID {
	return []segmentID{}
}*/
