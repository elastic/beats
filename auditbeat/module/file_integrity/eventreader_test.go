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

package file_integrity

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func init() {
	underTest = true
}

// ErrorSharingViolation is a Windows ERROR_SHARING_VIOLATION. It means "The
// process cannot access the file because it is being used by another process."
const ErrorSharingViolation syscall.Errno = 32

func TestEventReader(t *testing.T) {
	t.Skip("Flaky test: about 1/10 of builds fails https://github.com/elastic/beats/issues/21302")
	// Make dir to monitor.
	dir, err := ioutil.TempDir("", "audit")
	if err != nil {
		t.Fatal(err)
	}
	// under macOS, temp dir has a symlink in the path (/var -> /private/var)
	// and the path returned in events has the symlink resolved
	if runtime.GOOS == "darwin" {
		if dirAlt, err := filepath.EvalSymlinks(dir); err == nil {
			dir = dirAlt
		}
	}
	defer os.RemoveAll(dir)

	// Create a new EventProducer.
	config := defaultConfig
	config.Paths = []string{dir}
	r, err := NewEventReader(config)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	defer close(done)
	events, err := r.Start(done)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new file.
	txt1 := filepath.Join(dir, "test1.txt")
	var fileMode os.FileMode = 0640
	mustRun(t, "created", func(t *testing.T) {
		if err = ioutil.WriteFile(txt1, []byte("hello"), fileMode); err != nil {
			t.Fatal(err)
		}

		event := readTimeout(t, events)
		assert.EqualValues(t, Created, event.Action&Created)
		assertSameFile(t, txt1, event.Path)
		if runtime.GOOS != "windows" {
			assert.EqualValues(t, fileMode, event.Info.Mode)
		}
	})

	// Rename the file.
	txt2 := filepath.Join(dir, "test2.txt")
	mustRun(t, "move", func(t *testing.T) {
		rename(t, txt1, txt2)

		received := readMax(t, 3, events)
		if len(received) == 0 {
			t.Fatal("no events received")
		}
		if runtime.GOOS == "darwin" {
			for _, e := range received {
				switch {
				// Destination file only gets the Moved flag
				case e.Action == Moved:
					assertSameFile(t, txt2, e.Path)
				// Source file is moved and updated
				case 0 != e.Action&Moved, 0 != e.Action&Updated:
					assertSameFile(t, txt1, e.Path)
				default:
					t.Errorf("unexpected event: %+v", e)
				}
			}
		} else {
			for _, e := range received {
				switch {
				case 0 != e.Action&Moved, 0 != e.Action&Updated:
					assert.Equal(t, txt1, e.Path)
				case 0 != e.Action&Created:
					assertSameFile(t, txt2, e.Path)
				default:
					t.Errorf("unexpected event: %+v", e)
				}
			}
		}
	})

	// Chmod the file.
	mustRun(t, "attributes modified", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip()
		}

		if err = os.Chmod(txt2, 0644); err != nil {
			t.Fatal(err)
		}

		event := readTimeout(t, events)
		assertSameFile(t, txt2, event.Path)
		assert.EqualValues(t, AttributesModified, AttributesModified&event.Action)
		assert.EqualValues(t, 0644, event.Info.Mode)
	})

	// Append data to the file.
	mustRun(t, "updated", func(t *testing.T) {
		f, err := os.OpenFile(txt2, os.O_RDWR|os.O_APPEND, fileMode)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString(" world!")
		f.Sync()
		f.Close()

		event := readTimeout(t, events)
		assertSameFile(t, txt2, event.Path)
		assert.EqualValues(t, Updated, Updated&event.Action)
		if runtime.GOOS != "windows" {
			assert.EqualValues(t, 0644, event.Info.Mode)
		}
	})

	// Change the GID of the file.
	mustRun(t, "chown", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skip chown on windows")
		}

		gid := changeGID(t, txt2)
		event := readTimeout(t, events)
		assertSameFile(t, txt2, event.Path)
		assert.EqualValues(t, AttributesModified, AttributesModified&event.Action)
		assert.EqualValues(t, gid, event.Info.GID)
	})

	mustRun(t, "deleted", func(t *testing.T) {
		if err = os.Remove(txt2); err != nil {
			t.Fatal(err)
		}

		event := readTimeout(t, events)
		assert.EqualValues(t, Deleted, Deleted&event.Action)
	})

	// Create a sub-directory.
	subDir := filepath.Join(dir, "subdir")
	mustRun(t, "dir created", func(t *testing.T) {
		if err = os.Mkdir(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		event := readTimeout(t, events)
		assertSameFile(t, subDir, event.Path)
	})

	// Test moving a file into the monitored dir from outside.
	var moveInOrig string
	moveIn := filepath.Join(dir, "test3.txt")
	mustRun(t, "move in", func(t *testing.T) {
		f, err := ioutil.TempFile("", "test3.txt")
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString("move-in")
		f.Sync()
		f.Close()
		moveInOrig = f.Name()

		rename(t, moveInOrig, moveIn)

		event := readTimeout(t, events)

		if runtime.GOOS == "darwin" {
			assert.EqualValues(t, Moved, event.Action)
		} else {
			assert.EqualValues(t, Created, event.Action)
		}
		assertSameFile(t, moveIn, event.Path)
	})

	// Test moving a file out of the monitored dir.
	mustRun(t, "move out", func(t *testing.T) {
		rename(t, moveIn, moveInOrig)
		defer os.Remove(moveInOrig)

		event := readTimeout(t, events)
		assertSameFile(t, moveIn, event.Path)
		if runtime.GOOS == "windows" {
			assert.EqualValues(t, Deleted, event.Action)
		} else {
			assert.EqualValues(t, Moved, Moved&event.Action)
		}
	})

	// Test that it does not monitor recursively.
	subFile := filepath.Join(subDir, "foo.txt")
	mustRun(t, "non-recursive", func(t *testing.T) {
		if err = ioutil.WriteFile(subFile, []byte("foo"), fileMode); err != nil {
			t.Fatal(err)
		}

		assertNoEvent(t, events)
	})
}

func TestRaces(t *testing.T) {
	t.Skip("Flaky test: about 1/20 of builds fails https://github.com/elastic/beats/issues/21303")
	const (
		fileMode os.FileMode = 0640
		N                    = 100
	)

	var dirs []string

	for i := 0; i < N; i++ {
		dir, err := ioutil.TempDir("", "audit")
		if err != nil {
			t.Fatal(err)
		}
		if dir, err = filepath.EvalSymlinks(dir); err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, dir)
	}

	defer func() {
		for _, dir := range dirs {
			os.RemoveAll(dir)
		}
	}()

	// Create a new EventProducer.
	config := defaultConfig
	config.Paths = dirs
	config.Recursive = true
	r, err := NewEventReader(config)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	defer close(done)

	// Generate a lot of events in parallel to Start() so there is a chance of
	// events arriving before all watched dirs are Add()-ed
	go func() {
		for i := 0; i < 10; i++ {
			for _, dir := range dirs {
				fname := filepath.Join(dir, fmt.Sprintf("%d.dat", i))
				ioutil.WriteFile(fname, []byte("hello"), fileMode)
			}
		}
	}()
	eventC, err := r.Start(done)
	if err != nil {
		t.Fatal(err)
	}

	const marker = "test_file"
	for _, dir := range dirs {
		fname := filepath.Join(dir, marker)
		ioutil.WriteFile(fname, []byte("hello"), fileMode)
	}

	got := 0
	for i := 0; got < N; i++ {
		ev := readTimeout(t, eventC)
		if strings.Contains(ev.Path, marker) {
			got++
		}
	}
	assert.Equal(t, N, got)
}

// readTimeout reads one event from the channel and returns it. If it does
// not receive an event after one second it will time-out and fail the test.
func readTimeout(t testing.TB, events <-chan Event) Event {
	select {
	case <-time.After(time.Second):
		t.Fatalf("%+v", errors.Errorf("timed-out waiting for event"))
	case e, ok := <-events:
		if !ok {
			t.Fatal("failed reading from event channel")
		}
		t.Logf("%+v", buildMetricbeatEvent(&e, false))
		return e
	}

	return Event{}
}

// readMax reads events from the channel over a period of one second and returns
// the events. If the max number of events is received it returns early.
func readMax(t testing.TB, max int, events <-chan Event) []Event {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	var received []Event
	for {
		select {
		case <-timer.C:
			return received
		case e, ok := <-events:
			if !ok {
				t.Fatal("failed reading from event channel")
			}
			t.Logf("%+v", buildMetricbeatEvent(&e, false))
			received = append(received, e)
			if len(received) >= max {
				return received
			}
		}
	}
}

// assertNoEvent asserts that no event is received on the channel. It waits for
// 250ms.
func assertNoEvent(t testing.TB, events <-chan Event) {
	select {
	case e := <-events:
		t.Fatal("received unexpected event", e)
	case <-time.After(250 * time.Millisecond):
	}
}

// assertSameFile asserts that two files are the same.
func assertSameFile(t testing.TB, f1, f2 string) {
	if f1 == f2 {
		return
	}

	info1, err := os.Lstat(f1)
	if err != nil {
		t.Error(err)
		return
	}

	info2, err := os.Lstat(f2)
	if err != nil {
		t.Error(err)
		return
	}

	assert.True(t, os.SameFile(info1, info2), "%v and %v are not the same file", f1, f2)
}

// changeGID changes the GID of a file using chown. It uses the second group
// that the user is a member of. If the user is only a member of one group then
// it will skip the test.
func changeGID(t testing.TB, file string) int {
	groups, err := os.Getgroups()
	if err != nil {
		t.Fatal("failed to get groups", err)
	}

	if len(groups) <= 1 {
		t.Skip("no group that we can change to")
	}

	// The second one will be a non-default group.
	gid := groups[1]
	if err = os.Chown(file, -1, gid); err != nil {
		t.Fatal(err)
	}

	return gid
}

// mustRun runs a sub-test and stops the execution of the parent if the sub-test
// fails.
func mustRun(t *testing.T, name string, f func(t *testing.T)) {
	if !t.Run(name, f) {
		t.FailNow()
	}
}

// rename renames a file or it fails the test. It retries the rename operation
// multiple times before failing.
//
// https://support.microsoft.com/en-us/help/316609/prb-error-sharing-violation-error-message-when-the-createfile-function
func rename(t *testing.T, oldPath, newPath string) {
	const maxRetries = 100

	for retries := 0; retries < maxRetries; retries++ {
		err := os.Rename(oldPath, newPath)
		if err == nil {
			if retries > 0 {
				t.Logf("rename needed %d retries", retries)
			}
			return
		}

		if linkErr, ok := err.(*os.LinkError); ok && linkErr.Err == ErrorSharingViolation {
			time.Sleep(time.Millisecond)
			continue
		}

		t.Fatal(err)
	}
}
