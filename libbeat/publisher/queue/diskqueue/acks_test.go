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

	dqa.addFrames([]*readFrame{})
	t.Fatal(fmt.Errorf("hello there"))
}
