package diskqueue

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestAddFrames(t *testing.T) {
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
	dqa := newDiskQueueACKs(logp.L(), queuePosition{}, stateFile)

	segment0 := &queueSegment{
		id:           3,
		firstFrameID: 100,
	}
	segment1 := &queueSegment{
		id:           4,
		firstFrameID: 1,
	}
	frame0 := &readFrame{}

	dqa.addFrames([]*readFrame{frame0})
	t.Fatal(fmt.Errorf("hello there"))
}
