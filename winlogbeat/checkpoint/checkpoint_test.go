package checkpoint_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/stretchr/testify/assert"
)

// Test that a write is triggered when the maximum number of updates is reached.
func TestWriteMaxUpdates(t *testing.T) {
	dir, err := ioutil.TempDir("", "winlogbeat-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	file := filepath.Join(dir, "winlogbeat-test")
	cp, err := checkpoint.NewCheckpoint(file, 2, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Shutdown()

	assert.False(t, fileExists(file))
	cp.Persist("App", 1, time.Now())
	time.Sleep(time.Second)
	assert.False(t, fileExists(file))

	cp.Persist("App", 2, time.Now())
	time.Sleep(time.Second)
	assert.True(t, fileExists(file))
}

// Test that a write is triggered when the maximum time period since the last
// write is reached.
func TestWriteTimedFlush(t *testing.T) {
	dir, err := ioutil.TempDir("", "winlogbeat-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	file := filepath.Join(dir, "winlogbeat-test")
	cp, err := checkpoint.NewCheckpoint(file, 100, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Shutdown()

	assert.False(t, fileExists(file))
	cp.Persist("App", 1, time.Now())
	time.Sleep(1500 * time.Millisecond)
	assert.True(t, fileExists(file))
}

// fileExists returns true if the specified file exists.
func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
