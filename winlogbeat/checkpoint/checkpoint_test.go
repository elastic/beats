// +build !integration

package checkpoint

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test that a write is triggered when the maximum number of updates is reached.
func TestWriteMaxUpdates(t *testing.T) {
	dir, err := ioutil.TempDir("", "wlb-checkpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	file := filepath.Join(dir, "some", "new", "dir", ".winlogbeat.yml")
	if !assert.False(t, fileExists(file), "%s should not exist", file) {
		return
	}

	cp, err := NewCheckpoint(file, 2, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Shutdown()

	// Send update - it's not written to disk but it's in memory.
	cp.Persist("App", 1, time.Now(), "")
	time.Sleep(500 * time.Millisecond)
	_, found := cp.States()["App"]
	assert.True(t, found)

	ps, err := cp.read()
	if err != nil {
		t.Fatal("read failed", err)
	}
	assert.Len(t, ps.States, 0)

	// Send update - it is written to disk.
	cp.Persist("App", 2, time.Now(), "")
	time.Sleep(750 * time.Millisecond)
	ps, err = cp.read()
	if err != nil {
		t.Fatal("read failed", err)
	}
	if assert.Len(t, ps.States, 1, "state not written, could be a flush timing issue, retry") {
		assert.Equal(t, "App", ps.States[0].Name)
		assert.Equal(t, uint64(2), ps.States[0].RecordNumber)
	}
}

// Test that a write is triggered when the maximum time period since the last
// write is reached.
func TestWriteTimedFlush(t *testing.T) {
	dir, err := ioutil.TempDir("", "wlb-checkpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	file := filepath.Join(dir, ".winlogbeat.yml")
	if !assert.False(t, fileExists(file), "%s should not exist", file) {
		return
	}

	cp, err := NewCheckpoint(file, 100, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Shutdown()

	// Send update then wait longer than the flush interval and it should be
	// on disk.
	cp.Persist("App", 1, time.Now(), "")
	time.Sleep(1500 * time.Millisecond)
	ps, err := cp.read()
	if err != nil {
		t.Fatal("read failed", err)
	}
	if assert.Len(t, ps.States, 1) {
		assert.Equal(t, "App", ps.States[0].Name)
		assert.Equal(t, uint64(1), ps.States[0].RecordNumber)
	}
}

// Test that createDir creates the directory with 0750 permissions.
func TestCreateDir(t *testing.T) {
	dir, err := ioutil.TempDir("", "wlb-checkpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	stateDir := filepath.Join(dir, "state", "dir", "does", "not", "exists")
	file := filepath.Join(stateDir, ".winlogbeat.yml")
	cp := &Checkpoint{file: file}

	if !assert.False(t, fileExists(file), "%s should not exist", file) {
		return
	}
	if err = cp.createDir(); err != nil {
		t.Fatal("createDir", err)
	}
	if !assert.True(t, fileExists(stateDir), "%s should exist", file) {
		return
	}

	// mkdir on Windows does not pass the POSIX mode to the CreateDirectory
	// syscall so doesn't test the mode.
	if runtime.GOOS != "windows" {
		fileInfo, err := os.Stat(stateDir)
		if assert.NoError(t, err) {
			assert.Equal(t, true, fileInfo.IsDir())
			assert.Equal(t, os.FileMode(0750), fileInfo.Mode().Perm())
		}
	}
}

// Test createDir when the directory already exists to verify that no error is
// returned.
func TestCreateDirAlreadyExists(t *testing.T) {
	dir, err := ioutil.TempDir("", "wlb-checkpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	file := filepath.Join(dir, ".winlogbeat.yml")
	cp := &Checkpoint{file: file}

	if !assert.True(t, fileExists(dir), "%s should exist", file) {
		return
	}
	assert.NoError(t, cp.createDir())
}

// fileExists returns true if the specified file exists.
func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
