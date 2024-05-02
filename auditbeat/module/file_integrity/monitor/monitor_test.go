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

//go:build !integration

package monitor

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

func alwaysInclude(path string) bool {
	return false
}

func TestNonRecursive(t *testing.T) {
	dir := t.TempDir()

	watcher, err := New(false, alwaysInclude)
	assertNoError(t, err)
	assertNoError(t, watcher.Add(dir))
	assertNoError(t, watcher.Start())

	testDirOps(t, dir, watcher)

	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0o750)

	ev, err := readTimeout(t, watcher)
	assertNoError(t, err)
	assert.Equal(t, subdir, ev.Name)
	assert.Equal(t, fsnotify.Create, ev.Op)

	// subdirs are not watched
	subfile := filepath.Join(subdir, "file.dat")
	assertNoError(t, os.WriteFile(subfile, []byte("foo"), 0o640))

	_, err = readTimeout(t, watcher)
	assert.Error(t, err)
	assert.Equal(t, errReadTimeout, err)

	assertNoError(t, watcher.Close())
}

func TestRecursive(t *testing.T) {
	if runtime.GOOS == "darwin" {
		// This test races on Darwin because internal races in the kqueue
		// implementation of fsnotify when a watch is added in response to
		// a subdirectory created inside a watched directory.
		// This race doesn't affect auditbeat because the file_integrity module
		// under Darwin uses fsevents instead of kqueue.
		t.Skip("Disabled on Darwin")
	}
	dir := t.TempDir()

	watcher, err := New(true, alwaysInclude)
	assertNoError(t, err)

	assertNoError(t, watcher.Add(dir))

	assertNoError(t, watcher.Start())

	testDirOps(t, dir, watcher)

	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0o750)

	ev, err := readTimeout(t, watcher)
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

	dir := t.TempDir()

	// Create a separate dir

	linkedDir := t.TempDir()

	// Add a symbolic link from watched dir to the other

	symLink := filepath.Join(dir, "link")
	assertNoError(t, os.Symlink(linkedDir, symLink))

	// Start the watcher

	watcher, err := New(true, alwaysInclude)
	assertNoError(t, err)

	assertNoError(t, watcher.Add(dir))
	assertNoError(t, watcher.Start())

	// Create a file in the other dir

	file := filepath.Join(linkedDir, "not.seen")
	assertNoError(t, os.WriteFile(file, []byte("hello"), 0o640))

	// No event is received
	ev, err := readTimeout(t, watcher)
	assert.Equal(t, errReadTimeout, err)
	if err == nil {
		t.Fatalf("Expected timeout, got event %+v", ev)
	}
	assertNoError(t, watcher.Close())
}

func TestRecursiveSubdirPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permissions test on Windows")
	}

	if os.Getuid() == 0 {
		t.Skip("skipping as root can access every file and thus this unittest will fail")
		return
	}

	// Create dir to be watched

	dir, err := os.MkdirTemp("", "monitor")
	assertNoError(t, err)
	if runtime.GOOS == "darwin" {
		if dirAlt, err := filepath.EvalSymlinks(dir); err == nil {
			dir = dirAlt
		}
	}
	defer os.RemoveAll(dir)

	// Create not watched dir

	outDir := t.TempDir()

	// Populate not watched subdir

	for _, name := range []string{"a", "b", "c"} {
		path := filepath.Join(outDir, name)
		assertNoError(t, os.Mkdir(path, 0o755))
		assertNoError(t, os.WriteFile(filepath.Join(path, name), []byte("Hello"), 0o644))
	}

	// Make a subdir not accessible

	assertNoError(t, os.Chmod(filepath.Join(outDir, "b"), 0))

	// Setup watches on watched dir

	watcher, err := New(true, alwaysInclude)
	assertNoError(t, err)

	assertNoError(t, watcher.Start())
	assertNoError(t, watcher.Add(dir))

	defer func() {
		assertNoError(t, watcher.Close())
	}()

	// No event is received

	ev, err := readTimeout(t, watcher)
	assert.Equal(t, errReadTimeout, err)
	if err != errReadTimeout {
		t.Fatalf("Expected timeout, got event %+v", ev)
	}

	// Move the outside directory into the watched

	dest := filepath.Join(dir, "subdir")
	assertNoError(t, os.Rename(outDir, dest))

	// Receive all events

	var evs []fsnotify.Event
	for {
		// No event is received
		ev, err := readTimeout(t, watcher)
		if errors.Is(err, errReadTimeout) {
			break
		}
		assertNoError(t, err)
		evs = append(evs, ev)
	}

	// Verify that events for all accessible files are received
	// File "b/b" is missing because a watch to b couldn't be installed

	expected := map[string]fsnotify.Op{
		dest:                       fsnotify.Create,
		filepath.Join(dest, "a"):   fsnotify.Create,
		filepath.Join(dest, "a/a"): fsnotify.Create,
		filepath.Join(dest, "b"):   fsnotify.Create,
		filepath.Join(dest, "c"):   fsnotify.Create,
		filepath.Join(dest, "c/c"): fsnotify.Create,
	}
	assert.Len(t, evs, len(expected))
	for _, ev := range evs {
		op, found := expected[ev.Name]
		assert.True(t, found, ev.Name)
		assert.Equal(t, op, ev.Op)
	}
}

func TestRecursiveExcludedPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permissions test on Windows")
	}

	// Create dir to be watched

	dir := t.TempDir()

	// Create not watched dir

	outDir := t.TempDir()

	// Populate not watched subdir

	for _, name := range []string{"a", "b", "c"} {
		path := filepath.Join(outDir, name)
		assertNoError(t, os.Mkdir(path, 0o755))
		assertNoError(t, os.WriteFile(filepath.Join(path, name), []byte("Hello"), 0o644))
	}

	// excludes file/dir named "b"
	selectiveExclude := func(path string) bool {
		r := filepath.Base(path) == "b"
		t.Logf("path: %v, excluded: %v\n", path, r)
		return r
	}

	// Setup watches on watched dir

	watcher, err := New(true, selectiveExclude)
	assertNoError(t, err)

	assertNoError(t, watcher.Start())
	assertNoError(t, watcher.Add(dir))

	defer func() {
		assertNoError(t, watcher.Close())
	}()

	// No event is received

	ev, err := readTimeout(t, watcher)
	assert.Equal(t, errReadTimeout, err)
	if err != errReadTimeout {
		t.Fatalf("Expected timeout, got event %+v", ev)
	}

	// Move the outside directory into the watched

	dest := filepath.Join(dir, "subdir")
	assertNoError(t, os.Rename(outDir, dest))

	// Receive all events

	var evs []fsnotify.Event
	for {
		// No event is received
		ev, err := readTimeout(t, watcher)
		if err == errReadTimeout {
			break
		}
		assertNoError(t, err)
		evs = append(evs, ev)
	}

	// Verify that events for all accessible files are received
	// "b" and "b/b" are missing as they are excluded

	expected := map[string]fsnotify.Op{
		dest:                       fsnotify.Create,
		filepath.Join(dest, "a"):   fsnotify.Create,
		filepath.Join(dest, "a/a"): fsnotify.Create,
		filepath.Join(dest, "c"):   fsnotify.Create,
		filepath.Join(dest, "c/c"): fsnotify.Create,
	}
	assert.Len(t, evs, len(expected))
	for _, ev := range evs {
		op, found := expected[ev.Name]
		assert.True(t, found, ev.Name)
		assert.Equal(t, op, ev.Op)
	}
}

func testDirOps(t *testing.T, dir string, watcher Watcher) {
	fpath := filepath.Join(dir, "file.txt")
	fpath2 := filepath.Join(dir, "file2.txt")

	// Create
	assertNoError(t, os.WriteFile(fpath, []byte("hello"), 0o640))

	ev, err := readTimeout(t, watcher)
	assertNoError(t, err)
	assert.Equal(t, fpath, ev.Name)
	assert.Equal(t, fsnotify.Create, ev.Op)

	// Update
	// Repeat the write if no event is received. Under macOS often
	// the write fails to generate a write event for non-recursive watcher
	for i := 0; i < 3; i++ {
		f, err := os.OpenFile(fpath, os.O_RDWR|os.O_APPEND, 0o640)
		assertNoError(t, err)
		f.WriteString(" world\n")
		f.Sync()
		f.Close()

		ev, err = readTimeout(t, watcher)
		if err == nil || err != errReadTimeout {
			break
		}
	}
	assertNoError(t, err)
	assert.Equal(t, fpath, ev.Name)
	assert.Equal(t, fsnotify.Write, ev.Op)

	// Consume all leftover writes to fpath
	for err == nil && ev.Name == fpath && ev.Op == fsnotify.Write {
		ev, err = readTimeout(t, watcher)
	}

	// Helper to read events ignoring writes. These have been observed
	// under Windows in two cases:
	// - Writes to the parent dir (metadata updates after update loop above?)
	// - Delayed writes to "fpath" file, not discarded by above consumer loop.
	readIgnoreWrites := func(t *testing.T, w Watcher) (fsnotify.Event, error) {
		for {
			ev, err := readTimeout(t, w)
			if err != nil || ev.Op != fsnotify.Write {
				return ev, err
			}
		}
	}

	// Move
	err = os.Rename(fpath, fpath2)
	assertNoError(t, err)

	evRename, err := readIgnoreWrites(t, watcher)
	assertNoError(t, err)

	evCreate, err := readIgnoreWrites(t, watcher)
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

	ev, err = readIgnoreWrites(t, watcher)
	assertNoError(t, err)

	assert.Equal(t, fpath2, ev.Name)
	assert.Equal(t, fsnotify.Remove, ev.Op)
}

var errReadTimeout = errors.New("read timeout")

// helper to read from channel
func readTimeout(tb testing.TB, watcher Watcher) (fsnotify.Event, error) {
	tb.Helper()

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fsnotify.Event{}, errReadTimeout

		case msg, ok := <-watcher.EventChannel():
			if !ok {
				return fsnotify.Event{}, errors.New("channel closed")
			}
			return msg, nil

		case err := <-watcher.ErrorChannel():
			tb.Log("readTimeout got error:", err)
		}
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
