package monitor

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

func TestNonRecursive(t *testing.T) {
	dir, err := ioutil.TempDir("", "monitor")
	assertNoError(t, err)
	// under macOS, temp dir has a symlink in the path (/var -> /private/var)
	// and the path returned in events has the symlink resolved
	if runtime.GOOS == "darwin" {
		if dirAlt, err := filepath.EvalSymlinks(dir); err == nil {
			dir = dirAlt
		}
	}
	defer os.RemoveAll(dir)

	watcher, err := New(false)
	assertNoError(t, err)

	assertNoError(t, watcher.Add(dir))

	assertNoError(t, watcher.Start())

	testDirOps(t, dir, watcher)

	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0750)

	ev, err := readTimeout(watcher.EventChannel())
	assertNoError(t, err)
	assert.Equal(t, subdir, ev.Name)
	assert.Equal(t, fsnotify.Create, ev.Op)

	// subdirs are not watched
	subfile := filepath.Join(subdir, "file.dat")
	assertNoError(t, ioutil.WriteFile(subfile, []byte("foo"), 0640))

	_, err = readTimeout(watcher.EventChannel())
	assert.Error(t, err)
	assert.Equal(t, errReadTimeout, err)

	assertNoError(t, watcher.Close())
}

func TestRecursive(t *testing.T) {
	dir, err := ioutil.TempDir("", "monitor")
	assertNoError(t, err)
	// under macOS, temp dir has a symlink in the path (/var -> /private/var)
	// and the path returned in events has the symlink resolved
	if runtime.GOOS == "darwin" {
		if dirAlt, err := filepath.EvalSymlinks(dir); err == nil {
			dir = dirAlt
		}
	}
	defer os.RemoveAll(dir)

	watcher, err := New(true)
	assertNoError(t, err)

	assertNoError(t, watcher.Add(dir))

	assertNoError(t, watcher.Start())

	testDirOps(t, dir, watcher)

	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0750)

	ev, err := readTimeout(watcher.EventChannel())
	assertNoError(t, err)
	assert.Equal(t, subdir, ev.Name)
	assert.Equal(t, fsnotify.Create, ev.Op)

	testDirOps(t, subdir, watcher)

	assertNoError(t, watcher.Close())
}

func TestRecursiveNoFollowSymlink(t *testing.T) {
	// fsnotify has a bug in darwin were it is following symlinks
	// see: https://github.com/fsnotify/fsnotify/issues/227
	if runtime.GOOS == "darwin" {
		t.Skip("This test fails under macOS due to a bug in fsnotify")
	}

	// Create a watched dir

	dir, err := ioutil.TempDir("", "monitor")
	assertNoError(t, err)
	// under macOS, temp dir has a symlink in the path (/var -> /private/var)
	// and the path returned in events has the symlink resolved
	if runtime.GOOS == "darwin" {
		if dirAlt, err := filepath.EvalSymlinks(dir); err == nil {
			dir = dirAlt
		}
	}
	defer os.RemoveAll(dir)

	// Create a separate dir

	linkedDir, err := ioutil.TempDir("", "linked")
	assertNoError(t, err)
	defer os.RemoveAll(linkedDir)

	// Add a symbolic link from watched dir to the other

	symLink := filepath.Join(dir, "link")
	assertNoError(t, os.Symlink(linkedDir, symLink))

	// Start the watcher

	watcher, err := New(true)
	assertNoError(t, err)

	assertNoError(t, watcher.Add(dir))
	assertNoError(t, watcher.Start())

	// Create a file in the other dir

	file := filepath.Join(linkedDir, "not.seen")
	assertNoError(t, ioutil.WriteFile(file, []byte("hello"), 0640))

	// No event is received
	ev, err := readTimeout(watcher.EventChannel())
	assert.Equal(t, errReadTimeout, err)
	if err == nil {
		t.Fatalf("Expected timeout, got event %+v", ev)
	}
	assertNoError(t, watcher.Close())
}

func testDirOps(t *testing.T, dir string, watcher Watcher) {
	fpath := filepath.Join(dir, "file.txt")
	fpath2 := filepath.Join(dir, "file2.txt")

	// Create
	assertNoError(t, ioutil.WriteFile(fpath, []byte("hello"), 0640))

	ev, err := readTimeout(watcher.EventChannel())
	assertNoError(t, err)
	assert.Equal(t, fpath, ev.Name)
	assert.Equal(t, fsnotify.Create, ev.Op)

	// Update
	// Repeat the write if no event is received. Under macOS often
	// the write fails to generate a write event for non-recursive watcher
	for i := 0; i < 3; i++ {
		f, err := os.OpenFile(fpath, os.O_RDWR|os.O_APPEND, 0640)
		assertNoError(t, err)
		f.WriteString(" world\n")
		f.Sync()
		f.Close()

		ev, err = readTimeout(watcher.EventChannel())
		if err == nil || err != errReadTimeout {
			break
		}
	}
	assertNoError(t, err)
	assert.Equal(t, fpath, ev.Name)
	assert.Equal(t, fsnotify.Write, ev.Op)

	// Move
	err = os.Rename(fpath, fpath2)
	assertNoError(t, err)

	evRename, err := readTimeout(watcher.EventChannel())
	assertNoError(t, err)
	// Sometimes a duplicate Write can be received under Linux, skip
	if evRename.Op == fsnotify.Write {
		evRename, err = readTimeout(watcher.EventChannel())
	}
	evCreate, err := readTimeout(watcher.EventChannel())
	assertNoError(t, err)

	if evRename.Op != fsnotify.Rename {
		evRename, evCreate = evCreate, evRename
	}

	assert.Equal(t, fpath, evRename.Name)
	assert.Equal(t, fsnotify.Rename, evRename.Op)

	assert.Equal(t, fpath2, evCreate.Name)
	assert.Equal(t, fsnotify.Create, evCreate.Op)

	// Delete
	err = os.Remove(fpath2)
	assertNoError(t, err)

	ev, err = readTimeout(watcher.EventChannel())
	assertNoError(t, err)

	// Windows: A write to the parent directory sneaks in
	if ev.Op == fsnotify.Write && ev.Name == dir {
		ev, err = readTimeout(watcher.EventChannel())
		assertNoError(t, err)
	}
	assert.Equal(t, fpath2, ev.Name)
	assert.Equal(t, fsnotify.Remove, ev.Op)
}

var errReadTimeout = errors.New("read timeout")

// helper to read from channel
func readTimeout(c <-chan fsnotify.Event) (fsnotify.Event, error) {
	select {
	case <-time.After(3 * time.Second):
		return fsnotify.Event{}, errReadTimeout

	case msg, ok := <-c:
		if !ok {
			return fsnotify.Event{}, errors.New("channel closed")
		}
		return msg, nil
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
