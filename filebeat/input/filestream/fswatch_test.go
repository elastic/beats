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

// This file was contributed to by generative AI

package filestream

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/common/match"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/testing/fs"
)

func TestIsObservationError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "missing path is a real disappearance",
			err:  &os.PathError{Op: "stat", Path: "/tmp/missing.log", Err: os.ErrNotExist},
			want: false,
		},
		{
			name: "not directory means the previous subtree is gone",
			err:  &os.PathError{Op: "readdir", Path: "/tmp/logs", Err: syscall.ENOTDIR},
			want: false,
		},
		{
			name: "fd exhaustion is transiently unobservable",
			err:  &os.PathError{Op: "open", Path: "/tmp/logs", Err: syscall.EMFILE},
			want: true,
		},
		{
			name: "permission denied is unobservable",
			err:  &os.PathError{Op: "open", Path: "/tmp/logs", Err: syscall.EACCES},
			want: true,
		},
		{
			name: "logical scanner rejection is not an observation failure",
			err:  errFileIgnored,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isObservationError(tc.err), "unexpected observation error classification")
		})
	}
}

func TestUnderAnyUnobservable(t *testing.T) {
	set := func(paths ...string) map[string]struct{} {
		m := make(map[string]struct{}, len(paths))
		for _, path := range paths {
			m[filepath.FromSlash(path)] = struct{}{}
		}
		return m
	}

	cases := []struct {
		name     string
		path     string
		prefixes map[string]struct{}
		want     bool
	}{
		{
			name:     "no prefixes never matches",
			path:     "/a/b/c",
			prefixes: set(),
			want:     false,
		},
		{
			name:     "exact directory match",
			path:     "/a/b",
			prefixes: set("/a/b"),
			want:     true,
		},
		{
			name:     "exact file match",
			path:     "/a/b/app.log",
			prefixes: set("/a/b/app.log"),
			want:     true,
		},
		{
			name:     "direct child of an unobservable directory",
			path:     "/a/b/c",
			prefixes: set("/a/b"),
			want:     true,
		},
		{
			name:     "deep descendant of an unobservable directory",
			path:     "/a/b/c/d/e.log",
			prefixes: set("/a/b"),
			want:     true,
		},
		{
			name:     "prefix is a mid-level ancestor",
			path:     "/a/b/c/d",
			prefixes: set("/a/b/c"),
			want:     true,
		},
		{
			name:     "matches one of several prefixes",
			path:     "/a/b/c",
			prefixes: set("/x/y", "/a/b", "/z"),
			want:     true,
		},
		{
			name:     "matches none of several prefixes",
			path:     "/a/b/c",
			prefixes: set("/x/y", "/z"),
			want:     false,
		},
		{
			name:     "sibling directory does not match",
			path:     "/a/c/f.log",
			prefixes: set("/a/b"),
			want:     false,
		},
		{
			name:     "separator-aware: /a/bc is not under /a/b",
			path:     "/a/bc",
			prefixes: set("/a/b"),
			want:     false,
		},
		{
			name:     "separator-aware: /foobar is not under /foo",
			path:     "/foobar/x",
			prefixes: set("/foo"),
			want:     false,
		},
		{
			name:     "path shorter than the prefix does not match",
			path:     "/a",
			prefixes: set("/a/b"),
			want:     false,
		},
		{
			name:     "unrelated path does not match",
			path:     "/x/y/z",
			prefixes: set("/a/b"),
			want:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, underAnyUnobservable(filepath.FromSlash(tc.path), tc.prefixes),
				"underAnyUnobservable(%q, %v)", filepath.FromSlash(tc.path), tc.prefixes)
		})
	}
}

func newTestMetrics() *loginp.Metrics {
	return loginp.NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())
}

func TestFileWatcher(t *testing.T) {
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, "*.log")}
	cfgStr := `
scanner:
  check_interval: 100ms
  resend_on_touch: true
  symlinks: false
  recursive_glob: true
  fingerprint:
    enabled: false
    offset: 0
    length: 1024
`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := logptest.NewFileLogger(t, filepath.Join("..", "..", "build", "integration-tests"))
	fw := createWatcherWithConfig(t, logger.Logger, paths, cfgStr)

	go fw.Run(ctx, newTestMetrics(), 0, time.Time{})

	t.Run("detects a new file", func(t *testing.T) {
		basename := "created.log"
		filename := filepath.Join(dir, basename)
		err := os.WriteFile(filename, []byte("hello"), 0777)
		require.NoError(t, err)

		e := fw.Event()
		expEvent := loginp.FSEvent{
			NewPath: filename,
			Op:      loginp.OpCreate,
			Descriptor: loginp.FileDescriptor{
				Filename: filename,
				Info:     file.ExtendFileInfo(&testFileInfo{name: basename, size: 5}), // 5 bytes written
			},
		}
		expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
		requireEqualEvents(t, expEvent, e)
	})

	t.Run("detects a file write", func(t *testing.T) {
		basename := "created.log"
		filename := filepath.Join(dir, basename)

		f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0777)
		require.NoError(t, err)
		_, err = f.WriteString("world")
		require.NoError(t, err)
		f.Close()

		e := fw.Event()
		expEvent := loginp.FSEvent{
			NewPath: filename,
			OldPath: filename,
			Op:      loginp.OpWrite,
			Descriptor: loginp.FileDescriptor{
				Filename: filename,
				Info:     file.ExtendFileInfo(&testFileInfo{name: basename, size: 10}), // +5 bytes appended
			},
		}
		expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
		requireEqualEvents(t, expEvent, e)
	})

	t.Run("detects a file rename", func(t *testing.T) {
		basename := "created.log"
		filename := filepath.Join(dir, basename)
		newBasename := "renamed.log"
		newFilename := filepath.Join(dir, newBasename)

		err := os.Rename(filename, newFilename)
		require.NoError(t, err)

		e := fw.Event()
		expEvent := loginp.FSEvent{
			NewPath: newFilename,
			OldPath: filename,
			Op:      loginp.OpRename,
			Descriptor: loginp.FileDescriptor{
				Filename: newFilename,
				Info:     file.ExtendFileInfo(&testFileInfo{name: newBasename, size: 10}),
			},
		}
		expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
		requireEqualEvents(t, expEvent, e)
	})

	t.Run("detects a file truncate", func(t *testing.T) {
		basename := "renamed.log"
		filename := filepath.Join(dir, basename)

		err := os.Truncate(filename, 2)
		require.NoError(t, err)

		e := fw.Event()
		expEvent := loginp.FSEvent{
			NewPath: filename,
			OldPath: filename,
			Op:      loginp.OpTruncate,
			Descriptor: loginp.FileDescriptor{
				Filename: filename,
				Info:     file.ExtendFileInfo(&testFileInfo{name: basename, size: 2}),
			},
		}
		expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
		requireEqualEvents(t, expEvent, e)
	})

	t.Run("emits truncate on touch when resend_on_touch is enabled", func(t *testing.T) {
		basename := "renamed.log"
		filename := filepath.Join(dir, basename)
		time := time.Now().Local().Add(time.Hour)
		err := os.Chtimes(filename, time, time)
		require.NoError(t, err)

		e := fw.Event()
		expEvent := loginp.FSEvent{
			NewPath: filename,
			OldPath: filename,
			Op:      loginp.OpTruncate,
			Descriptor: loginp.FileDescriptor{
				Filename: filename,
				Info:     file.ExtendFileInfo(&testFileInfo{name: basename, size: 2}),
			},
		}
		expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
		requireEqualEvents(t, expEvent, e)
	})

	t.Run("detects a file remove", func(t *testing.T) {
		basename := "renamed.log"
		filename := filepath.Join(dir, basename)

		err := os.Remove(filename)
		require.NoError(t, err)

		e := fw.Event()
		expEvent := loginp.FSEvent{
			OldPath: filename,
			Op:      loginp.OpDelete,
			Descriptor: loginp.FileDescriptor{
				Filename: filename,
				Info:     file.ExtendFileInfo(&testFileInfo{name: basename, size: 2}),
			},
		}
		expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
		requireEqualEvents(t, expEvent, e)
	})

	t.Run("propagates a fingerprints for a new file", func(t *testing.T) {
		dir := t.TempDir()
		paths := []string{filepath.Join(dir, "*.log")}
		cfgStr := `
scanner:
  check_interval: 100ms
  symlinks: false
  recursive_glob: true
  fingerprint:
    enabled: true
    offset: 0
    length: 1024
`

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		logger := logptest.NewFileLogger(t, filepath.Join("../", "../", "build", "integration-tests"))
		fw := createWatcherWithConfig(t, logger.Logger, paths, cfgStr)
		go fw.Run(ctx, newTestMetrics(), 0, time.Time{})

		basename := "created.log"
		filename := filepath.Join(dir, basename)
		err := os.WriteFile(filename, []byte(strings.Repeat("a", 1024)), 0777)
		require.NoError(t, err)

		e := fw.Event()
		expEvent := loginp.FSEvent{
			NewPath: filename,
			Op:      loginp.OpCreate,
			Descriptor: loginp.FileDescriptor{
				Filename:    filename,
				Fingerprint: completeFP("2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a"),
				Info:        file.ExtendFileInfo(&testFileInfo{name: basename, size: 1024}),
			},
		}
		expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
		requireEqualEvents(t, expEvent, e)
	})

	t.Run("does not emit events if a file is touched and resend_on_touch is disabled", func(t *testing.T) {
		dir := t.TempDir()
		paths := []string{filepath.Join(dir, "*.log")}
		cfgStr := `
scanner:
  fingerprint.enabled: false
  check_interval: 10ms
`

		ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
		defer cancel()

		logger := logptest.NewFileLogger(t, filepath.Join("../", "../", "build", "integration-tests"))
		fw := createWatcherWithConfig(t, logger.Logger, paths, cfgStr)
		go fw.Run(ctx, newTestMetrics(), 0, time.Time{})

		basename := "created.log"
		filename := filepath.Join(dir, basename)
		err := os.WriteFile(filename, []byte(strings.Repeat("a", 1024)), 0777)
		require.NoError(t, err)

		e := fw.Event()
		expEvent := loginp.FSEvent{
			NewPath: filename,
			Op:      loginp.OpCreate,
			Descriptor: loginp.FileDescriptor{
				Filename: filename,
				Info:     file.ExtendFileInfo(&testFileInfo{name: basename, size: 1024}),
			},
		}
		expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
		requireEqualEvents(t, expEvent, e)

		time := time.Now().Local().Add(time.Hour)
		err = os.Chtimes(filename, time, time)
		require.NoError(t, err)

		e = fw.Event()
		require.Equal(t, loginp.OpDone, e.Op)
	})

	t.Run("does not emit events for empty files", func(t *testing.T) {
		dir := t.TempDir()
		paths := []string{filepath.Join(dir, "*.log")}
		cfgStr := `
scanner:
  fingerprint.enabled: false
  check_interval: 50ms
`

		ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
		defer cancel()

		fw := createWatcherWithConfig(t, logptest.NewTestingLogger(t, ""), paths, cfgStr)
		// Wait for the watcher goroutine to exit before the subtest returns.
		// logptest.NewTestingLogger writes via t.Log, which is unsafe to call
		// after the subtest finishes and triggers a data race in
		// testing.(*common).destination. The deferred cancel above runs
		// before t.Cleanup, so this only needs to wait for Run to return.
		runDone := make(chan struct{})
		go func() {
			defer close(runDone)
			fw.Run(ctx, newTestMetrics(), 0, time.Time{})
		}()
		t.Cleanup(func() { <-runDone })

		basename := "created.log"
		filename := filepath.Join(dir, basename)
		err := os.WriteFile(filename, nil, 0777)
		require.NoError(t, err)

		t.Run("emits a create event once something is written to the empty file", func(t *testing.T) {
			err = os.WriteFile(filename, []byte("hello"), 0777)
			require.NoError(t, err)

			e := fw.Event()
			expEvent := loginp.FSEvent{
				NewPath: filename,
				Op:      loginp.OpCreate,
				Descriptor: loginp.FileDescriptor{
					Filename: filename,
					Info:     file.ExtendFileInfo(&testFileInfo{name: basename, size: 5}), // +5 bytes appended
				},
			}
			expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
			requireEqualEvents(t, expEvent, e)
		})
	})

	t.Run("does not emit an event for a fingerprint collision", func(t *testing.T) {
		dir := t.TempDir()
		paths := []string{filepath.Join(dir, "*.log")}
		cfgStr := `
scanner:
  check_interval: 10ms
  fingerprint.enabled: true
`

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		logger := logptest.NewFileLogger(t, filepath.Join("../", "../", "build", "integration-tests"))
		fw := createWatcherWithConfig(t, logger.Logger, paths, cfgStr)
		go fw.Run(ctx, newTestMetrics(), 0, time.Time{})

		basename := "created.log"
		filename := filepath.Join(dir, basename)
		err := os.WriteFile(filename, []byte(strings.Repeat("a", 1024)), 0777)
		require.NoError(t, err)

		e := fw.Event()
		expEvent := loginp.FSEvent{
			NewPath: filename,
			Op:      loginp.OpCreate,
			Descriptor: loginp.FileDescriptor{
				Filename:    filename,
				Fingerprint: completeFP("2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a"),
				Info:        file.ExtendFileInfo(&testFileInfo{name: basename, size: 1024}),
			},
		}
		expEvent.SrcID = fw.getFileIdentity(expEvent.Descriptor)
		requireEqualEvents(t, expEvent, e)

		// collisions are resolved in the alphabetical order, the first filename wins
		basename = "created_collision.log"
		filename = filepath.Join(dir, basename)
		err = os.WriteFile(filename, []byte(strings.Repeat("a", 1024)), 0777)
		require.NoError(t, err)

		e = fw.Event()
		// means no event
		require.Equal(t, loginp.OpDone, e.Op)
	})

	t.Run("does not log warnings on duplicate globs and filters out duplicates", func(t *testing.T) {
		dir := t.TempDir()
		firstBasename := "file-123.ndjson"
		secondBasename := "file-watcher-123.ndjson"
		firstFilename := filepath.Join(dir, firstBasename)
		secondFilename := filepath.Join(dir, secondBasename)
		err := os.WriteFile(firstFilename, []byte("line\n"), 0777)
		require.NoError(t, err)
		err = os.WriteFile(secondFilename, []byte("line\n"), 0777)
		require.NoError(t, err)

		paths := []string{
			// to emulate the case we have in the agent monitoring
			filepath.Join(dir, "file-*.ndjson"),
			filepath.Join(dir, "file-watcher-*.ndjson"),
		}
		cfgStr := `
scanner:
  fingerprint.enabled: false
  check_interval: 100ms
`

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		inMemoryLog, buff := logp.NewInMemoryLocal("", logp.JSONEncoderConfig())
		fw := createWatcherWithConfig(t, inMemoryLog, paths, cfgStr)

		// Wrap Run so we can wait for the watcher goroutine to exit before
		// inspecting the in-memory log buffer. The buffer returned by
		// logp.NewInMemoryLocal is goroutine safe for writes only — reading
		// it concurrently with watcher logging triggers the race detector.
		runDone := make(chan struct{})
		go func() {
			defer close(runDone)
			fw.Run(ctx, newTestMetrics(), 0, time.Time{})
		}()

		expectedEvents := []loginp.FSEvent{
			{
				NewPath: firstFilename,
				Op:      loginp.OpCreate,
				Descriptor: loginp.FileDescriptor{
					Filename: firstFilename,
					Info:     file.ExtendFileInfo(&testFileInfo{name: firstBasename, size: 5}), // "line\n"
				},
			},
			{
				NewPath: secondFilename,
				Op:      loginp.OpCreate,
				Descriptor: loginp.FileDescriptor{
					Filename: secondFilename,
					Info:     file.ExtendFileInfo(&testFileInfo{name: secondBasename, size: 5}), // "line\n"
				},
			},
		}

		// Add the SrcIDs
		for i := range expectedEvents {
			expectedEvents[i].SrcID = fw.getFileIdentity(expectedEvents[i].Descriptor)
		}
		var actualEvents []loginp.FSEvent
		actualEvents = append(actualEvents, fw.Event())
		actualEvents = append(actualEvents, fw.Event())

		// since this is coming from a map, the order is not deterministic
		// we need to sort events based on paths first
		// we expect only creation events for two different files, so it's alright.
		sort.Slice(actualEvents, func(i, j int) bool {
			return actualEvents[i].NewPath < actualEvents[j].NewPath
		})
		sort.Slice(expectedEvents, func(i, j int) bool {
			return expectedEvents[i].NewPath < expectedEvents[j].NewPath
		})

		for i, actualEvent := range actualEvents {
			requireEqualEvents(t, expectedEvents[i], actualEvent)
		}

		// Stop the watcher and wait for its goroutine to return so the buffer
		// is no longer being written to before we read from it.
		cancel()
		<-runDone

		require.NotContainsf(t, buff.String(), "WARN",
			"must be no warning messages")
	})
}

func TestFileWatcherCopyTruncateWithFingerprint(t *testing.T) {
	t.Run("copy truncate happens at once", func(t *testing.T) {
		w, activePath, rotatedPath := newFileWatcherForCopyTruncateTests(t)
		ctx := context.Background()

		// 1. A single file exists
		initialContent := strings.Repeat("a", 96)
		require.NoError(t, os.WriteFile(activePath, []byte(initialContent), 0o600), "failed to write initial active file")
		w.watch(ctx, newTestMetrics(), 0, time.Time{})

		initialEvents := drainPendingFSEvents(w.events)
		requireEventSignatures(t, initialEvents, []loginp.FSEvent{
			{Op: loginp.OpCreate, NewPath: activePath},
		})
		initialCreateEvt := findEvent(initialEvents, loginp.FSEvent{Op: loginp.OpCreate, NewPath: activePath})
		initialFingerprint := initialCreateEvt.Descriptor.Fingerprint
		require.NotEmpty(t, initialFingerprint, "initial active file fingerprint must be present")

		// 2. Copy+truncate:
		//   - copy foo.log -> foo.log.1
		//   - truncate foo.log and add data (less than previously)
		copyFile(t, activePath, rotatedPath)
		require.NoError(t, os.WriteFile(activePath, []byte(strings.Repeat("b", 64)), 0o600), "failed to rewrite active file after rotation")
		w.watch(ctx, newTestMetrics(), 0, time.Time{})

		events := drainPendingFSEvents(w.events)
		requireEventSignatures(t, events, []loginp.FSEvent{
			{Op: loginp.OpRename, OldPath: activePath, NewPath: rotatedPath},
			{Op: loginp.OpCreate, NewPath: activePath},
		})

		renamedEvt := findEvent(events, loginp.FSEvent{Op: loginp.OpRename, OldPath: activePath, NewPath: rotatedPath})
		createdActiveEvt := findEvent(events, loginp.FSEvent{Op: loginp.OpCreate, NewPath: activePath})
		require.Equal(t, initialFingerprint, renamedEvt.Descriptor.Fingerprint, "rotated file should keep initial fingerprint")
		require.NotEqual(t, initialFingerprint, createdActiveEvt.Descriptor.Fingerprint, "rewritten active file should get a new fingerprint")
	})

	t.Run("copy truncate happens in two steps", func(t *testing.T) {
		w, activePath, rotatedPath := newFileWatcherForCopyTruncateTests(t)
		ctx := context.Background()

		// 1. A single file exists
		initialContent := strings.Repeat("c", 96)
		require.NoError(t, os.WriteFile(activePath, []byte(initialContent), 0o600), "failed to write initial active file")
		w.watch(ctx, newTestMetrics(), 0, time.Time{})

		initialEvents := drainPendingFSEvents(w.events)
		requireEventSignatures(t, initialEvents, []loginp.FSEvent{
			{Op: loginp.OpCreate, NewPath: activePath},
		})
		initialCreateEvt := findEvent(initialEvents, loginp.FSEvent{Op: loginp.OpCreate, NewPath: activePath})
		initialFingerprint := initialCreateEvt.Descriptor.Fingerprint
		require.NotEmpty(t, initialFingerprint, "initial active file fingerprint must be present")

		// 2. The file is copied: foo.log -> foo.log.1
		copyFile(t, activePath, rotatedPath)
		w.watch(ctx, newTestMetrics(), 0, time.Time{})

		// Expectation: no file events, because both files are considered the same
		copyStepEvents := drainPendingFSEvents(w.events)
		require.Empty(t, copyStepEvents, "no file events when a file is copied (same fingerprint)")
		requireEventSignatures(t, copyStepEvents, []loginp.FSEvent{})

		// 3. foo.log is truncated & written to (less data than before).
		require.NoError(t, os.WriteFile(activePath, []byte(strings.Repeat("d", 64)), 0o600), "failed to truncate and rewrite active file")
		w.watch(ctx, newTestMetrics(), 0, time.Time{})

		// Expectation: 'foo.log' is considered new and 'foo.log.1' is considered a rename
		truncateStepEvents := drainPendingFSEvents(w.events)
		requireEventSignatures(t, truncateStepEvents, []loginp.FSEvent{
			{Op: loginp.OpCreate, NewPath: activePath},
			{Op: loginp.OpRename, OldPath: activePath, NewPath: rotatedPath},
		})
	})

	t.Run("copy truncate happens in three steps", func(t *testing.T) {
		w, activePath, rotatedPath := newFileWatcherForCopyTruncateTests(t)
		ctx := context.Background()

		// 1. A single file exists
		initialContent := strings.Repeat("e", 96)
		require.NoError(t, os.WriteFile(activePath, []byte(initialContent), 0o600), "failed to write initial active file")
		w.watch(ctx, newTestMetrics(), 0, time.Time{})

		initialEvents := drainPendingFSEvents(w.events)
		requireEventSignatures(t, initialEvents, []loginp.FSEvent{
			{Op: loginp.OpCreate, NewPath: activePath},
		})
		initialCreateEvt := findEvent(initialEvents, loginp.FSEvent{Op: loginp.OpCreate, NewPath: activePath})
		initialFingerprint := initialCreateEvt.Descriptor.Fingerprint
		require.NotEmpty(t, initialFingerprint, "initial active file fingerprint must be present")

		// 2. The file is copied: foo.log -> foo.log.1
		copyFile(t, activePath, rotatedPath)
		w.watch(ctx, newTestMetrics(), 0, time.Time{})

		// Expectation: no file events, because both files are considered the same
		copyStepEvents := drainPendingFSEvents(w.events)
		require.Empty(t, copyStepEvents, "no file events when a file is copied (same fingerprint)")
		requireEventSignatures(t, copyStepEvents, []loginp.FSEvent{})

		// 3. foo.log is truncated (0 bytes)
		require.NoError(t, os.WriteFile(activePath, nil, 0o600), "failed to truncate active file to empty")
		w.watch(ctx, newTestMetrics(), 0, time.Time{})

		// Expectation: foo.log is considered renamed: foo.log -> foo.log.1
		// the empty file foo.log is ignored because it is empty
		emptyStepEvents := drainPendingFSEvents(w.events)
		requireEventSignatures(t, emptyStepEvents, []loginp.FSEvent{
			{Op: loginp.OpRename, OldPath: activePath, NewPath: rotatedPath},
		})

		// 4. data is added to foo.log
		require.NoError(t, os.WriteFile(activePath, []byte(strings.Repeat("f", 64)), 0o600), "failed to add new data to active file")
		w.watch(ctx, newTestMetrics(), 0, time.Time{})

		// Expectation: foo.log is discovered as a new file
		newDataStepEvents := drainPendingFSEvents(w.events)
		requireEventSignatures(t, newDataStepEvents, []loginp.FSEvent{
			{Op: loginp.OpCreate, NewPath: activePath},
		})
		newActiveEvt := findEvent(newDataStepEvents, loginp.FSEvent{Op: loginp.OpCreate, NewPath: activePath})
		require.NotEqual(t, initialFingerprint, newActiveEvt.Descriptor.Fingerprint, "newly recreated active file should have a different fingerprint")
	})
}

// newFileWatcherForCopyTruncateTests returns a file watcher configured to
// harvest rotated files and two file paths used for rotation.
func newFileWatcherForCopyTruncateTests(t *testing.T) (watcher *fileWatcher, activePath string, rotatedPath string) {
	dir := fs.TempDir(t, "..", "..", "build")
	activePath = filepath.Join(dir, "foo.log")
	rotatedPath = filepath.Join(dir, "foo.log.1")
	paths := []string{filepath.Join(dir, "foo.log*")}
	cfgStr := `
scanner:
  check_interval: 10ms
  fingerprint:
    length: 64
`

	logger := logptest.NewFileLogger(t, dir)
	w := createWatcherWithConfig(t, logger.Logger, paths, cfgStr)
	w.events = make(chan loginp.FSEvent, 16)
	return w, activePath, rotatedPath
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()

	content, err := os.ReadFile(from)
	require.NoError(t, err, "failed to read source file %q", from)
	//nolint:gosec // All paths are controlled by the test code. It's safe
	require.NoError(t, os.WriteFile(to, content, 0o600), "failed to write destination file %q", to)
}

// fsEventToString returns a stable string representation that includes
// only the fields: Op, OldPath and NewPath. The returned string is
// human-friendly.
func fsEventToString(e loginp.FSEvent) string {
	return fmt.Sprintf("Op: '%s'|OldPath: '%s'|NewPath: '%s'", e.Op, e.OldPath, e.NewPath)
}

func requireEventSignatures(t *testing.T, events, expected []loginp.FSEvent) {
	t.Helper()

	actualKeys := make([]string, 0, len(events))
	for _, e := range events {
		actualKeys = append(actualKeys, fsEventToString(e))
	}

	expectedKeys := make([]string, 0, len(expected))
	for _, e := range expected {
		expectedKeys = append(expectedKeys, fsEventToString(e))
	}

	require.ElementsMatch(t, expectedKeys, actualKeys, "unexpected file watcher events (order ignored)")
}

// findEvent finds expected in events by comparing Op, OldPath and NewPath.
func findEvent(events []loginp.FSEvent, expected loginp.FSEvent) loginp.FSEvent {
	for _, e := range events {
		if e.Op == expected.Op && e.OldPath == expected.OldPath && e.NewPath == expected.NewPath {
			return e
		}
	}
	return loginp.FSEvent{}
}

func drainPendingFSEvents(events <-chan loginp.FSEvent) []loginp.FSEvent {
	drained := make([]loginp.FSEvent, 0)

	for {
		select {
		case e := <-events:
			drained = append(drained, e)
		default:
			return drained
		}
	}
}

func TestFileScanner(t *testing.T) {
	dir := t.TempDir()
	dir2 := t.TempDir() // for symlink testing
	paths := []string{filepath.Join(dir, "*.log")}

	normalBasename := "normal.log"
	undersizedBasename := "undersized.log"
	normalGZIPBasename := "normal.gz.log"
	undersizedGZIPBasename := "undersized.gz.log"
	excludedBasename := "excluded.log"
	excludedIncludedBasename := "excluded_included.log"
	travelerBasename := "traveler.log"
	normalSymlinkBasename := "normal_symlink.log"
	exclSymlinkBasename := "excl_symlink.log"
	travelerSymlinkBasename := "portal.log"
	undersizedGlob := "undersized-*.txt"

	normalFilename := filepath.Join(dir, normalBasename)
	undersizedFilename := filepath.Join(dir, undersizedBasename)
	undersized1Filename := filepath.Join(dir, "undersized-1.txt")
	undersized2Filename := filepath.Join(dir, "undersized-2.txt")
	undersized3Filename := filepath.Join(dir, "undersized-3.txt")
	normalGZIPFilename := filepath.Join(dir, normalGZIPBasename)
	undersizedGZIPFilename := filepath.Join(dir, undersizedGZIPBasename)
	excludedFilename := filepath.Join(dir, excludedBasename)
	excludedIncludedFilename := filepath.Join(dir, excludedIncludedBasename)
	travelerFilename := filepath.Join(dir2, travelerBasename)
	normalSymlinkFilename := filepath.Join(dir, normalSymlinkBasename)
	exclSymlinkFilename := filepath.Join(dir, exclSymlinkBasename)
	travelerSymlinkFilename := filepath.Join(dir, travelerSymlinkBasename)

	normalRepeat := 1024
	undersizedRepeat := 128
	files := map[string]string{
		normalFilename:           strings.Repeat("a", normalRepeat),
		undersizedFilename:       strings.Repeat("a", undersizedRepeat),
		excludedFilename:         strings.Repeat("nothing to see here", normalRepeat),
		undersized1Filename:      strings.Repeat("1", 42),
		undersized2Filename:      strings.Repeat("2", 42),
		undersized3Filename:      strings.Repeat("3", 42),
		excludedIncludedFilename: strings.Repeat("perhaps something to see here", normalRepeat),
		travelerFilename:         strings.Repeat("folks, I think I got lost", normalRepeat),
	}
	// GZIP files should behave just like plain-text files. Thus using the same
	// content length, but different data so the fingerprint won't be the same
	gzFiles := map[string]string{
		normalGZIPFilename:     strings.Repeat("g", normalRepeat),
		undersizedGZIPFilename: strings.Repeat("g", undersizedRepeat),
	}

	sizes := make(map[string]int64, len(files)+len(gzFiles))
	for filename, content := range files {
		sizes[filename] = int64(len(content))
	}
	for filename, content := range files {
		err := os.WriteFile(filename, []byte(content), 0777)
		require.NoError(t, err)
	}

	for basename, content := range gzFiles {
		f, err := os.OpenFile(basename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		require.NoError(t, err, "could not create gzip file")

		w := gzip.NewWriter(f)
		_, err = w.Write([]byte(content))
		require.NoError(t, err, "could not write to gzip file")
		require.NoError(t, w.Close(), "could not close gzip writer")

		fi, err := f.Stat()
		require.NoError(t, err, "could not stat gzip file to get its size")

		sizes[basename] = fi.Size()
		require.NoError(t, err)
		require.NoError(t, f.Close(), "could not close gzip file")
	}

	// this is to test that a symlink for a known file does not add the file twice
	err := os.Symlink(normalFilename, normalSymlinkFilename)
	require.NoError(t, err)

	// this is to test that a symlink for an unknown file is added once
	err = os.Symlink(travelerFilename, travelerSymlinkFilename)
	require.NoError(t, err)

	// this is to test that a symlink to an excluded file is not added
	err = os.Symlink(exclSymlinkFilename, exclSymlinkFilename)
	require.NoError(t, err)

	// this is to test that directories are handled and excluded
	err = os.Mkdir(filepath.Join(dir, "dir"), 0777)
	require.NoError(t, err)

	cases := []struct {
		name        string
		cfgStr      string
		compression string
		expDesc     map[string]loginp.FileDescriptor
	}{
		{
			name: "returns all files when no limits, not including the repeated symlink",
			cfgStr: `
scanner:
  symlinks: true
  recursive_glob: true
  fingerprint:
    enabled: false
    offset: 0
    length: 1024
`,
			expDesc: map[string]loginp.FileDescriptor{
				normalFilename: {
					Filename: normalFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalFilename],
						name: normalBasename,
					}),
				},
				normalGZIPFilename: {
					Filename: normalGZIPFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalGZIPFilename],
						name: normalGZIPBasename,
					}),
				},
				undersizedFilename: {
					Filename: undersizedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedFilename],
						name: undersizedBasename,
					}),
				},
				undersizedGZIPFilename: {
					Filename: undersizedGZIPFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedGZIPFilename],
						name: undersizedGZIPBasename,
					}),
				},
				excludedFilename: {
					Filename: excludedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedFilename],
						name: excludedBasename,
					}),
				},
				excludedIncludedFilename: {
					Filename: excludedIncludedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedIncludedFilename],
						name: excludedIncludedBasename,
					}),
				},
				travelerSymlinkFilename: {
					Filename: travelerSymlinkFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[travelerFilename],
						name: travelerSymlinkBasename,
					}),
				},
			},
		},
		{
			name: "returns filtered files, excluding symlinks",
			cfgStr: `
scanner:
  symlinks: false # symlinks are disabled
  recursive_glob: false
  fingerprint:
    enabled: false
    offset: 0
    length: 1024
`,
			expDesc: map[string]loginp.FileDescriptor{
				normalFilename: {
					Filename: normalFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalFilename],
						name: normalBasename,
					}),
				},
				normalGZIPFilename: {
					Filename: normalGZIPFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalGZIPFilename],
						name: normalGZIPBasename,
					}),
				},
				undersizedFilename: {
					Filename: undersizedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedFilename],
						name: undersizedBasename,
					}),
				},
				undersizedGZIPFilename: {
					Filename: undersizedGZIPFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedGZIPFilename],
						name: undersizedGZIPBasename,
					}),
				},
				excludedFilename: {
					Filename: excludedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedFilename],
						name: excludedBasename,
					}),
				},
				excludedIncludedFilename: {
					Filename: excludedIncludedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedIncludedFilename],
						name: excludedIncludedBasename,
					}),
				},
			},
		},
		{
			name: "returns files according to excluded list",
			cfgStr: `
scanner:
  exclude_files: ['.*exclude.*']
  symlinks: true
  recursive_glob: true
  fingerprint:
    enabled: false
    offset: 0
    length: 1024
`,
			expDesc: map[string]loginp.FileDescriptor{
				normalFilename: {
					Filename: normalFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalFilename],
						name: normalBasename,
					}),
				},
				normalGZIPFilename: {
					Filename: normalGZIPFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalGZIPFilename],
						name: normalGZIPBasename,
					}),
				},
				undersizedFilename: {
					Filename: undersizedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedFilename],
						name: undersizedBasename,
					}),
				},
				normalGZIPFilename: {
					Filename: normalGZIPFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalGZIPFilename],
						name: normalGZIPBasename,
					}),
				},
				undersizedFilename: {
					Filename: undersizedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedFilename],
						name: undersizedBasename,
					}),
				},
				undersizedGZIPFilename: {
					Filename: undersizedGZIPFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedGZIPFilename],
						name: undersizedGZIPBasename,
					}),
				},
				travelerSymlinkFilename: {
					Filename: travelerSymlinkFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[travelerFilename],
						name: travelerSymlinkBasename,
					}),
				},
			},
		},
		{
			name: "returns no symlink if the original file is excluded",
			cfgStr: `
scanner:
  fingerprint.enabled: false
  exclude_files: ['.*exclude.*', '.*traveler.*']
  symlinks: true
`,
			expDesc: map[string]loginp.FileDescriptor{
				normalFilename: {
					Filename: normalFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalFilename],
						name: normalBasename,
					}),
				},
				normalGZIPFilename: {
					Filename: normalGZIPFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalGZIPFilename],
						name: normalGZIPBasename,
					}),
				},
				undersizedFilename: {
					Filename: undersizedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedFilename],
						name: undersizedBasename,
					}),
				},
				undersizedGZIPFilename: {
					Filename: undersizedGZIPFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedGZIPFilename],
						name: undersizedGZIPBasename,
					}),
				},
			},
		},
		{
			name: "returns files according to included list",
			cfgStr: `
scanner:
  include_files: ['.*include.*']
  symlinks: true
  recursive_glob: true
  fingerprint:
    enabled: false
    offset: 0
    length: 1024
`,
			expDesc: map[string]loginp.FileDescriptor{
				excludedIncludedFilename: {
					Filename: excludedIncludedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedIncludedFilename],
						name: excludedIncludedBasename,
					}),
				},
			},
		},
		{
			name: "returns no included symlink if the original file is not included",
			cfgStr: `
scanner:
  fingerprint.enabled: false
  include_files: ['.*include.*', '.*portal.*']
  symlinks: true
`,
			expDesc: map[string]loginp.FileDescriptor{
				excludedIncludedFilename: {
					Filename: excludedIncludedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedIncludedFilename],
						name: excludedIncludedBasename,
					}),
				},
			},
		},
		{
			name: "returns an included symlink if the original file is included",
			cfgStr: `
scanner:
  fingerprint.enabled: false
  include_files: ['.*include.*', '.*portal.*', '.*traveler.*']
  symlinks: true
`,
			expDesc: map[string]loginp.FileDescriptor{
				excludedIncludedFilename: {
					Filename: excludedIncludedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedIncludedFilename],
						name: excludedIncludedBasename,
					}),
				},
				travelerSymlinkFilename: {
					Filename: travelerSymlinkFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[travelerFilename],
						name: travelerSymlinkBasename,
					}),
				},
			},
		},
		{
			name:        "returns all files except too small to fingerprint",
			compression: CompressionAuto,
			cfgStr: `
scanner:
  symlinks: true
  recursive_glob: true
  fingerprint:
    enabled: true
    offset: 0
    length: 1024
`,
			expDesc: map[string]loginp.FileDescriptor{
				normalFilename: {
					Filename:    normalFilename,
					Fingerprint: completeFP("2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a"),
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalFilename],
						name: normalBasename,
					}),
				},
				normalGZIPFilename: {
					Filename:    normalGZIPFilename,
					Fingerprint: completeFP("af1ee623faf25c42385da9f1bc222a3ccfd6722d6d6bcdc78538215d479b7ac7"),
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalGZIPFilename],
						name: normalGZIPBasename,
					}),
				},
				excludedFilename: {
					Filename:    excludedFilename,
					Fingerprint: completeFP("bd151321c3bbdb44185414a1b56b5649a00206dd4792e7230db8904e43987336"),
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedFilename],
						name: excludedBasename,
					}),
				},
				excludedIncludedFilename: {
					Filename:    excludedIncludedFilename,
					Fingerprint: completeFP("bfdb99a65297062658c26dfcea816d76065df2a2da2594bfd9b96e9e405da1c2"),
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedIncludedFilename],
						name: excludedIncludedBasename,
					}),
				},
				travelerSymlinkFilename: {
					Filename:    travelerSymlinkFilename,
					Fingerprint: completeFP("c4058942bffcea08810a072d5966dfa5c06eb79b902bf0011890dd8d22e1a5f8"),
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[travelerFilename],
						name: travelerSymlinkBasename,
					}),
				},
			},
		},
		{
			name: "returns all files that match a non-standard fingerprint window",
			cfgStr: `
scanner:
  symlinks: true
  recursive_glob: true
  fingerprint:
    enabled: true
    offset: 2
    length: 64
`,
			expDesc: map[string]loginp.FileDescriptor{
				normalFilename: {
					Filename:    normalFilename,
					Fingerprint: completeFP("ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb"),
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalFilename],
						name: normalBasename,
					}),
				},
				// undersizedFilename got excluded because of the matching fingerprint
				excludedFilename: {
					Filename:    excludedFilename,
					Fingerprint: completeFP("9c225a1e6a7df9c869499e923565b93937e88382bb9188145f117195cd41dcd1"),
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedFilename],
						name: excludedBasename,
					}),
				},
				excludedIncludedFilename: {
					Filename:    excludedIncludedFilename,
					Fingerprint: completeFP("7985b2b9750bdd3c76903db408aff3859204d6334279eaf516ecaeb618a218d5"),
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedIncludedFilename],
						name: excludedIncludedBasename,
					}),
				},
				travelerSymlinkFilename: {
					Filename:    travelerSymlinkFilename,
					Fingerprint: completeFP("da437600754a8eed6c194b7241b078679551c06c7dc89685a9a71be7829ad7e5"),
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[travelerFilename],
						name: travelerSymlinkBasename,
					}),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger := logptest.NewTestingLogger(t, "")
			s := createScannerWithConfig(t, logger, paths, tc.cfgStr, tc.compression)
			files, _, _ := s.GetFiles(loginp.FileScanOptions{})
			requireEqualFiles(t, tc.expDesc, files)
		})
	}

	t.Run("does not issue warnings when file is too small", func(t *testing.T) {
		cfgStr := `
scanner:
  fingerprint:
    enabled: true
    offset: 0
    length: 1024
`
		logger, buffer := logp.NewInMemoryLocal("test-logger", zapcore.EncoderConfig{})

		// the glob for the very small files
		paths := []string{filepath.Join(dir, undersizedGlob)}
		s := createScannerWithConfig(t, logger, paths, cfgStr, CompressionNone)
		files, _, _ := s.GetFiles(loginp.FileScanOptions{})
		require.Empty(t, files)
		files, _, _ = s.GetFiles(loginp.FileScanOptions{})
		require.Empty(t, files)
		files, _, _ = s.GetFiles(loginp.FileScanOptions{})
		require.Empty(t, files)

		logs := parseLogs(buffer.String())
		require.NotEmpty(t, logs, "fileScanner.GetFiles must log messages")

		// For each file that is too small to be ingested, s.GetFiles must log
		// a summary warning (only once) and then an individual debug message per file
		singleFileFormat := "cannot start ingesting from file %[1]q: filesize of %[1]q is 42 bytes"
		expectedLogs := []struct {
			level string
			msg   string
			count int
		}{
			{"warn", "ingestion from some files will be delayed", 1},
			{"debug", fmt.Sprintf(singleFileFormat, undersized1Filename), 3},
			{"debug", fmt.Sprintf(singleFileFormat, undersized2Filename), 3},
			{"debug", fmt.Sprintf(singleFileFormat, undersized3Filename), 3},
		}

		for _, el := range expectedLogs {
			found := 0
			for _, log := range logs[1:] {
				if !strings.HasPrefix(log.message, el.msg) {
					continue
				}
				found++
				assert.Equalf(t, el.level, log.level, "log level for %q does not match", el.msg)
			}

			assert.Equalf(t, el.count, found, "the amount of log lines %q does not match", el.msg)
		}
	})

	t.Run("returns error when creating scanner with a fingerprint too small", func(t *testing.T) {
		cfg := fileWatcherConfig{
			Scanner: fileScannerConfig{
				Fingerprint: fingerprintConfig{
					Enabled: true,
					Offset:  0,
					Length:  1,
				},
			}}
		_, err = newFileWatcher(
			logptest.NewTestingLogger(t, ""),
			paths,
			cfg,
			CompressionNone,
			false,
			mustPathIdentifier(false),
			mustSourceIdentifier("foo-id"),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "fingerprint size 1 bytes cannot be smaller than 64 bytes")
	})

	t.Run("caps a fingerprint larger than the maximum and warns", func(t *testing.T) {
		cfg := fileScannerConfig{
			Fingerprint: fingerprintConfig{
				Enabled: true,
				Offset:  0,
				Length:  MaxFingerprintSize + 1,
			},
		}
		inMemoryLog, buff := logp.NewInMemoryLocal("", logp.JSONEncoderConfig())
		s, err := newFileScanner(inMemoryLog, paths, cfg, CompressionNone)
		require.NoError(t, err, "an oversized fingerprint length must be capped, not rejected")
		assert.Equal(t, MaxFingerprintSize, s.cfg.Fingerprint.Length,
			"the fingerprint length must be capped to the maximum")
		assert.Len(t, s.readBuffer, int(MaxFingerprintSize),
			"the read buffer must be allocated at the capped length, not the configured one")
		assert.Contains(t, buff.String(), "exceeds the maximum",
			"capping an oversized fingerprint length must log a warning")
	})

	t.Run("empty regular files are silently excluded", func(t *testing.T) {
		dir := t.TempDir()
		empty := filepath.Join(dir, "empty.log")
		err := os.WriteFile(empty, nil, 0644)
		require.NoError(t, err)

		nonEmpty := filepath.Join(dir, "nonempty.log")
		err = os.WriteFile(nonEmpty, []byte("hello"), 0644)
		require.NoError(t, err)

		cfg := fileScannerConfig{
			Symlinks:    false,
			Fingerprint: fingerprintConfig{Enabled: false},
		}
		inMemoryLog, buff := logp.NewInMemoryLocal("", logp.JSONEncoderConfig())
		s, err := newFileScanner(inMemoryLog, []string{filepath.Join(dir, "*.log")}, cfg, CompressionNone)
		require.NoError(t, err)

		files, _, _ := s.GetFiles(loginp.FileScanOptions{})
		assert.Len(t, files, 1, "empty.log must be excluded")
		assert.Contains(t, files, nonEmpty, "nonempty.log should be included")
		assert.NotContains(t, buff.String(), "GetFiles") // every line has a source prefix
	})

	t.Run("symlinks to empty files are silently excluded", func(t *testing.T) {
		dir := t.TempDir()
		emptyTarget := filepath.Join(dir, "empty_target.txt")
		err := os.WriteFile(emptyTarget, nil, 0644)
		require.NoError(t, err)

		emptyLink := filepath.Join(dir, "empty_link.log")
		err = os.Symlink(emptyTarget, emptyLink)
		require.NoError(t, err)

		nonEmptyTarget := filepath.Join(dir, "nonempty_target.txt")
		err = os.WriteFile(nonEmptyTarget, []byte("content"), 0644)
		require.NoError(t, err)

		nonEmptyLink := filepath.Join(dir, "nonempty_link.log")
		err = os.Symlink(nonEmptyTarget, nonEmptyLink)
		require.NoError(t, err)

		cfg := fileScannerConfig{
			Symlinks:    true,
			Fingerprint: fingerprintConfig{Enabled: false},
		}
		inMemoryLog, buff := logp.NewInMemoryLocal("", logp.JSONEncoderConfig())
		s, err := newFileScanner(inMemoryLog, []string{filepath.Join(dir, "*.log")}, cfg, CompressionNone)
		require.NoError(t, err)

		files, _, _ := s.GetFiles(loginp.FileScanOptions{})
		assert.Len(t, files, 1, "empty_link.log must be excluded")
		assert.Contains(t, files, nonEmptyLink, "nonempty_link.log should be included")
		assert.NotContains(t, buff.String(), "GetFiles") // every line has a source prefix
	})
}

func TestFileScannerScanMetrics(t *testing.T) {
	dir := t.TempDir()
	keepLog := filepath.Join(dir, "keep.log")
	excludedLog := filepath.Join(dir, "excluded.log")
	emptyLog := filepath.Join(dir, "empty.log")
	smallLog := filepath.Join(dir, "small.log")
	dirLog := filepath.Join(dir, "directory.log")
	linkLog := filepath.Join(dir, "link.log")
	oldLog := filepath.Join(dir, "old.log")

	now := time.Now()
	require.NoError(t, os.WriteFile(keepLog, []byte(strings.Repeat("k", 128)), 0644), "failed to write keep log")
	require.NoError(t, os.WriteFile(excludedLog, []byte(strings.Repeat("e", 128)), 0644), "failed to write excluded log")
	require.NoError(t, os.WriteFile(emptyLog, nil, 0644), "failed to write empty log")
	require.NoError(t, os.WriteFile(smallLog, []byte("small"), 0644), "failed to write small log")
	require.NoError(t, os.WriteFile(oldLog, []byte(strings.Repeat("o", 128)), 0644), "failed to write old log")
	require.NoError(t, os.Mkdir(dirLog, 0755), "failed to create directory")
	require.NoError(t, os.Symlink(keepLog, linkLog), "failed to create symlink")
	require.NoError(t, os.Chtimes(oldLog, now.Add(-2*time.Hour), now.Add(-2*time.Hour)), "failed to age old log")

	paths := []string{
		filepath.Join(dir, "*.log"),
	}
	cfgStr := `
scanner:
  exclude_files: ['.*excluded.*']
  symlinks: false
  recursive_glob: false
  fingerprint:
    enabled: true
    offset: 0
    length: 64
`

	scanner := createScannerWithConfig(t, logp.NewNopLogger(), paths, cfgStr, CompressionNone)
	files, scanMetrics, _ := scanner.GetFiles(loginp.FileScanOptions{
		CurrentTime: now,
		IgnoreOlder: time.Hour,
	})
	require.Contains(t, files, keepLog, "keep log must be ingestible")
	require.Contains(t, files, oldLog, "old log must still be returned")
	require.Len(t, files, 2, "keep and old logs should be ingestible scan targets")

	assert.Equal(t, loginp.FileScanMetrics{
		FilesIgnored:        2,
		FilesMatched:        7,
		FilesNoIngestTarget: 3,
		FilesEmpty:          1,
		FilesUnique:         2,
		ScanErrors:          0,
	}, scanMetrics, "unexpected scan metrics")
}

func TestFileWatcherScanMetricsCountsIgnoredFiles(t *testing.T) {
	dir := t.TempDir()
	oldLog := filepath.Join(dir, "old.log")
	newLog := filepath.Join(dir, "new.log")

	require.NoError(t, os.WriteFile(oldLog, []byte("old\n"), 0644), "failed to write old log")
	require.NoError(t, os.WriteFile(newLog, []byte("new\n"), 0644), "failed to write new log")
	oldModTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(oldLog, oldModTime, oldModTime), "failed to age old log")

	fw := createWatcherWithConfig(t, logp.NewNopLogger(), []string{filepath.Join(dir, "*.log")}, `
scanner:
  fingerprint.enabled: false
`)
	metrics := newTestMetrics()
	baseline := loginp.FileScanMetrics{
		FilesMatched:        metrics.FilesMatched.Get(),
		FilesUnique:         metrics.FilesUnique.Get(),
		FilesNoIngestTarget: metrics.FilesNoIngestTarget.Get(),
		FilesIgnored:        metrics.FilesIgnored.Get(),
		FilesEmpty:          metrics.FilesEmpty.Get(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fw.watch(ctx, metrics, time.Hour, time.Time{})

	assert.Equal(t, baseline.FilesMatched+2, metrics.FilesMatched.Get(), "files_matched")
	assert.Equal(t, baseline.FilesUnique+2, metrics.FilesUnique.Get(), "files_unique")
	assert.Equal(t, baseline.FilesNoIngestTarget, metrics.FilesNoIngestTarget.Get(), "files_no_ingest_target")
	assert.Equal(t, baseline.FilesIgnored+1, metrics.FilesIgnored.Get(), "files_ignored")
	assert.Equal(t, baseline.FilesEmpty, metrics.FilesEmpty.Get(), "files_empty")
}

// getFilesViaGlob is the verbatim pre-#48686 filepath.Glob based GetFiles (copied
// from main, with the receiver passed as an argument). It is the oracle the
// single-pass walker must match.
//
// It deliberately calls the same production getIngestTarget/toFileDescriptor
// helpers the walker uses, so the parity comparison isolates exactly the changed
// logic — path enumeration, matching and dedup — and not the per-file ingest-target
// resolution the two share. A behaviour change in those shared helpers moves both
// sides together and would not be caught here; that is intended, they are covered
// by their own tests (e.g. TestGetIngestTarget).
func getFilesViaGlob(s *fileScanner) map[string]loginp.FileDescriptor {
	fdByName := map[string]loginp.FileDescriptor{}
	// used to determine if a symlink resolves in a already known target
	uniqueIDs := map[string]string{}
	// used to filter out duplicate matches
	uniqueFiles := map[string]struct{}{}

	for _, path := range s.paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			s.log.Errorf("glob(%s) failed: %v", path, err)
			continue
		}

		for _, filename := range matches {
			// in case multiple globs match on the same file we filter out duplicates
			if _, knownFile := uniqueFiles[filename]; knownFile {
				continue
			}
			uniqueFiles[filename] = struct{}{}

			it, err := s.getIngestTarget(filename)
			if err != nil {
				if !errors.Is(err, errFileEmpty) {
					s.log.Debugf("cannot create an ingest target for file %q: %s", filename, err)
				}
				continue
			}

			fd, err := s.toFileDescriptor(&it)
			if errors.Is(err, errFileTooSmall) {
				if s.smallFilesWarned.CompareAndSwap(false, true) {
					s.log.Warnf("ingestion from some files will be delayed, files need to be at "+
						"least %d in size for ingestion to start. To change this "+
						"behaviour set 'prospector.scanner.fingerprint.length' and "+
						"'prospector.scanner.fingerprint.offset'. "+
						"Enable debug logging to see all file names of delayed files.",
						s.cfg.Fingerprint.Offset+s.cfg.Fingerprint.Length)
				}
				s.log.Debugf("cannot start ingesting from file %q: %s", filename, err)
				continue
			}
			if err != nil {
				s.log.Warnf("cannot create a file descriptor for an ingest target %q: %s", filename, err)
				continue
			}

			fileID := fd.FileID()
			if knownFilename, exists := uniqueIDs[fileID]; exists {
				s.log.Warnf("%q points to an already known ingest target %q [%s==%s]. Skipping", fd.Filename, knownFilename, fileID, fileID)
				continue
			}
			uniqueIDs[fileID] = fd.Filename
			fdByName[filename] = fd
		}
	}

	return fdByName
}

// TestScannerWalkMatchesGlob asserts the single-pass walker returns exactly the
// same files as the previous filepath.Glob implementation across a range of trees,
// including the symlink-following and depth-cap behaviour that must be preserved.
func TestScannerWalkMatchesGlob(t *testing.T) {
	mkfile := func(t *testing.T, path string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o770))
		require.NoError(t, os.WriteFile(path, []byte("data"), 0o660))
	}
	noFingerprint := fingerprintConfig{Enabled: false}

	tests := []struct {
		name  string
		setup func(t *testing.T, dir string) ([]string, fileScannerConfig)
		extra func(t *testing.T, dir string, files map[string]loginp.FileDescriptor)
	}{
		{
			name: "flat glob",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "a.json"))
				mkfile(t, filepath.Join(dir, "b.json"))
				mkfile(t, filepath.Join(dir, "c.log"))
				return []string{filepath.Join(dir, "*.json")},
					fileScannerConfig{Fingerprint: noFingerprint}
			},
		},
		{
			name: "recursive depths incl. cap boundary",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "d1.json"))                     // depth 1
				mkfile(t, filepath.Join(dir, "a/b/c.json"))                  // depth 3
				mkfile(t, filepath.Join(dir, "a/a/a/a/a/a/a/a/d8.json"))     // depth 9 (8 dirs): in cap
				mkfile(t, filepath.Join(dir, "a/a/a/a/a/a/a/a/a/deep.json")) // depth 10 (9 dirs): beyond cap
				return []string{filepath.Join(dir, "**", "*.json")},
					fileScannerConfig{RecursiveGlob: true, Fingerprint: noFingerprint}
			},
			extra: func(t *testing.T, dir string, files map[string]loginp.FileDescriptor) {
				assert.Contains(t, files, filepath.Join(dir, "a/a/a/a/a/a/a/a/d8.json"),
					"file at the depth cap must be included")
				assert.NotContains(t, files, filepath.Join(dir, "a/a/a/a/a/a/a/a/a/deep.json"),
					"file beyond RecursiveGlobDepth must be excluded")
			},
		},
		{
			name: "exclude_files",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "keep.json"))
				mkfile(t, filepath.Join(dir, "sub/skip-me.json"))
				return []string{filepath.Join(dir, "**", "*.json")},
					fileScannerConfig{
						RecursiveGlob: true,
						ExcludedFiles: []match.Matcher{match.MustCompile("skip-")},
						Fingerprint:   noFingerprint,
					}
			},
		},
		{
			name: "include_files",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "wanted.json"))
				mkfile(t, filepath.Join(dir, "other.json"))
				return []string{filepath.Join(dir, "*.json")},
					fileScannerConfig{
						IncludedFiles: []match.Matcher{match.MustCompile("wanted")},
						Fingerprint:   noFingerprint,
					}
			},
		},
		{
			name: "symlinked intermediate directory",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				real := filepath.Join(dir, "real")
				mkfile(t, filepath.Join(real, "x.json"))
				mkfile(t, filepath.Join(real, "nested/y.json"))
				require.NoError(t, os.Symlink(real, filepath.Join(dir, "link")))
				return []string{filepath.Join(dir, "**", "*.json")},
					fileScannerConfig{RecursiveGlob: true, Fingerprint: noFingerprint}
			},
			extra: func(t *testing.T, dir string, files map[string]loginp.FileDescriptor) {
				assert.Contains(t, files, filepath.Join(dir, "link", "x.json"),
					"file reachable only via a symlinked dir must be found")
			},
		},
		{
			name: "symlink cycle terminates",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "a.json"))
				require.NoError(t, os.Symlink(dir, filepath.Join(dir, "loop")))
				return []string{filepath.Join(dir, "**", "*.json")},
					fileScannerConfig{RecursiveGlob: true, Fingerprint: noFingerprint}
			},
		},
		{
			name: "recursive_glob disabled",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "a.json"))
				mkfile(t, filepath.Join(dir, "sub/b.json"))
				return []string{filepath.Join(dir, "**", "*.json")},
					fileScannerConfig{RecursiveGlob: false, Fingerprint: noFingerprint}
			},
		},
		{
			name: "missing base directory",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				return []string{filepath.Join(dir, "does-not-exist", "**", "*.json")},
					fileScannerConfig{RecursiveGlob: true, Fingerprint: noFingerprint}
			},
		},
		{
			name: "many exclude patterns",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "a/doc-1.json"))
				mkfile(t, filepath.Join(dir, "a/b/doc-2.ndjson"))
				mkfile(t, filepath.Join(dir, "skip/secret.json")) // excluded by /skip/.*
				return []string{
						filepath.Join(dir, "**", "*.json"),
						filepath.Join(dir, "**", "*.ndjson"),
					},
					fileScannerConfig{
						RecursiveGlob: true,
						ExcludedFiles: benchExcludePatterns(),
						Fingerprint:   noFingerprint,
					}
			},
		},
		{
			// The same file is reachable at depth 3 (real path) and depth 2
			// (through the symlink, in a lexically earlier sibling). Both the old
			// glob and the new walker must keep the shallower path so the "path"
			// file identity is stable. This case diverges under a naive
			// depth-first "first wins" dedup.
			name: "symlink alias keeps shallowest path",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "areal/deep/x.json"))
				require.NoError(t, os.Symlink(
					filepath.Join(dir, "areal/deep"), filepath.Join(dir, "zlink")))
				return []string{filepath.Join(dir, "**", "*.json")},
					fileScannerConfig{RecursiveGlob: true, Fingerprint: noFingerprint}
			},
			extra: func(t *testing.T, dir string, files map[string]loginp.FileDescriptor) {
				assert.Len(t, files, 1)
				assert.Contains(t, files, filepath.Join(dir, "zlink", "x.json"),
					"the shallower aliased path must be kept")
				assert.NotContains(t, files, filepath.Join(dir, "areal", "deep", "x.json"),
					"the deeper aliased path must be deduplicated away")
			},
		},
		{
			// The same file is reachable through two configured globs with
			// different bases: directly under "a" (depth 1) and through a symlink
			// under "b" (depth 2). "b" is configured first, so its (deeper) path
			// must win — matching main's config-path-order tie-break, not a
			// shallowest-path heuristic.
			name: "config-path order wins across bases",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "a", "x.log"))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "b"), 0o770))
				require.NoError(t, os.Symlink(
					filepath.Join(dir, "a"), filepath.Join(dir, "b", "link")))
				return []string{
						filepath.Join(dir, "b", "**", "*.log"),
						filepath.Join(dir, "a", "**", "*.log"),
					},
					fileScannerConfig{RecursiveGlob: true, Fingerprint: noFingerprint}
			},
			extra: func(t *testing.T, dir string, files map[string]loginp.FileDescriptor) {
				assert.Len(t, files, 1)
				assert.Contains(t, files, filepath.Join(dir, "b", "link", "x.log"),
					"the path under the first configured glob must win")
				assert.NotContains(t, files, filepath.Join(dir, "a", "x.log"))
			},
		},
		{
			// Two sibling dirs where one name is a byte-prefix of the other and
			// the next byte sorts before '/' ('-' is 0x2d, '/' is 0x2f): full-path
			// byte order puts d-x/a.log first, but filepath.Glob sorts per
			// directory, so main kept d/z.log. The files share a fingerprint, so
			// they collide on FileID and the tie-break (same pattern, equal scan
			// order index) decides which path survives.
			name: "same fingerprint tie-break keeps glob order",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				content := []byte(strings.Repeat("same-", 20)) // identical fingerprints
				for _, p := range []string{
					filepath.Join(dir, "d", "z.log"),
					filepath.Join(dir, "d-x", "a.log"),
				} {
					require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o770))
					require.NoError(t, os.WriteFile(p, content, 0o660))
				}
				return []string{filepath.Join(dir, "*", "*.log")},
					fileScannerConfig{Fingerprint: fingerprintConfig{Enabled: true, Length: 64}}
			},
			extra: func(t *testing.T, dir string, files map[string]loginp.FileDescriptor) {
				assert.Len(t, files, 1, "same-fingerprint files must dedup to one")
				assert.Contains(t, files, filepath.Join(dir, "d", "z.log"),
					"the tie-break must keep the path filepath.Glob returned first (per-directory sort)")
			},
		},
		{
			// filepath.Glob returns any entry whose name matches, including a
			// symlink resolving to a directory; with symlinks enabled and
			// fingerprinting off main kept it in the result map. The walker must
			// yield it too and leave type filtering to getIngestTarget.
			name: "symlink to directory at leaf",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "f.log"))
				require.NoError(t, os.Mkdir(filepath.Join(dir, "targetdir"), 0o770))
				require.NoError(t, os.Symlink(
					filepath.Join(dir, "targetdir"), filepath.Join(dir, "linkdir")))
				return []string{filepath.Join(dir, "*")},
					fileScannerConfig{Symlinks: true, Fingerprint: noFingerprint}
			},
			extra: func(t *testing.T, dir string, files map[string]loginp.FileDescriptor) {
				// The walker matching the filepath.Glob oracle is asserted above and
				// holds on every OS. Whether the symlink-to-directory survives
				// getIngestTarget is platform specific: it stats the resolved target
				// and drops it when Size()==0, which a directory reports on Windows
				// but not on Unix. So on Unix the symlink is kept like a file; on
				// Windows it is filtered out like the plain directory it points to.
				if runtime.GOOS != "windows" {
					assert.Contains(t, files, filepath.Join(dir, "linkdir"),
						"a symlink resolving to a directory is matched like a file by filepath.Glob")
				}
				assert.NotContains(t, files, filepath.Join(dir, "targetdir"),
					"a plain directory is rejected by getIngestTarget")
			},
		},
		{
			// A literal component after the first wildcard: only the "app" child
			// of each first-level dir can match, and the walker must not collect
			// (nor descend into) sibling subtrees.
			name: "literal component after wildcard",
			setup: func(t *testing.T, dir string) ([]string, fileScannerConfig) {
				mkfile(t, filepath.Join(dir, "x", "app", "f.log"))
				mkfile(t, filepath.Join(dir, "x", "other", "g.log"))
				mkfile(t, filepath.Join(dir, "y", "app", "h.log"))
				return []string{filepath.Join(dir, "*", "app", "*.log")},
					fileScannerConfig{Fingerprint: noFingerprint}
			},
			extra: func(t *testing.T, dir string, files map[string]loginp.FileDescriptor) {
				assert.Len(t, files, 2, "only files under app/ match the pattern")
				assert.Contains(t, files, filepath.Join(dir, "x", "app", "f.log"))
				assert.Contains(t, files, filepath.Join(dir, "y", "app", "h.log"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			paths, cfg := tc.setup(t, dir)
			s, err := newFileScanner(logptest.NewTestingLogger(t, ""), paths, cfg, CompressionNone)
			require.NoError(t, err)

			got, _, _ := s.GetFiles(loginp.FileScanOptions{})
			requireEqualFiles(t, getFilesViaGlob(s), got)

			if tc.extra != nil {
				tc.extra(t, dir, got)
			}
		})
	}
}

// TestScannerStablePathWithDuplicateFingerprint verifies that when two files share
// a fingerprint (so they are deduplicated by FileID), GetFiles keeps the shallowest
// path, so the "path" file identity stays stable and the file is not re-ingested.
// See https://github.com/elastic/beats/issues/48686.
func TestScannerStablePathWithDuplicateFingerprint(t *testing.T) {
	dir := t.TempDir()
	content := []byte(strings.Repeat("identical-header-", 8)) // same fingerprint for both
	for _, p := range []string{
		filepath.Join(dir, "deep", "sub", "b.json"), // depth 3
		filepath.Join(dir, "a.json"),                // depth 1
	} {
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o770))
		require.NoError(t, os.WriteFile(p, content, 0o660))
	}

	cfg := fileScannerConfig{
		RecursiveGlob: true,
		Fingerprint:   fingerprintConfig{Enabled: true, Length: 64},
	}
	s, err := newFileScanner(logptest.NewTestingLogger(t, ""),
		[]string{filepath.Join(dir, "**", "*.json")}, cfg, CompressionNone)
	require.NoError(t, err)

	files, _, _ := s.GetFiles(loginp.FileScanOptions{})
	require.Len(t, files, 1, "files with the same fingerprint must dedup to one")
	assert.Contains(t, files, filepath.Join(dir, "a.json"), "must keep the shallowest path")
}

func TestGlobRoot(t *testing.T) {
	base := t.TempDir()

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "literal path returns itself",
			pattern: filepath.Join(base, "var", "log", "syslog"),
			want:    filepath.Join(base, "var", "log", "syslog"),
		},
		{
			name:    "wildcard in basename returns its directory",
			pattern: filepath.Join(base, "logs", "*.log"),
			want:    filepath.Join(base, "logs"),
		},
		{
			name:    "recursive glob returns the dir before **",
			pattern: filepath.Join(base, "logs", "**", "*.json"),
			want:    filepath.Join(base, "logs"),
		},
		{
			name:    "wildcard mid-path returns the leading literal dir",
			pattern: filepath.Join(base, "*", "app", "*.log"),
			want:    base,
		},
		{
			name:    "multiple wildcards return the leading literal dir",
			pattern: filepath.Join(base, "*", "*", "*.json"),
			want:    base,
		},
		{
			name:    "character class counts as a metacharacter",
			pattern: filepath.Join(base, "logs", "app[0-9]", "out.log"),
			want:    filepath.Join(base, "logs"),
		},
		{
			name:    "question mark counts as a metacharacter",
			pattern: filepath.Join(base, "logs", "?.log"),
			want:    filepath.Join(base, "logs"),
		},
		{
			name:    "relative pattern collapses to the current dir",
			pattern: "*.json",
			want:    ".",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, globRoot(tc.pattern), "globRoot(%q)", tc.pattern)
		})
	}
}

func TestDepthBelow(t *testing.T) {
	base := t.TempDir()

	tests := []struct {
		name    string
		root    string
		pattern string
		want    int
	}{
		{"pattern equals root", base, base, 0},
		{"one level", base, filepath.Join(base, "a.json"), 1},
		{"two levels", base, filepath.Join(base, "x", "y.json"), 2},
		{"three levels", base, filepath.Join(base, "x", "y", "z.json"), 3},
		{"wildcards count as segments", base, filepath.Join(base, "*", "*.json"), 2},
		{"nested root", filepath.Join(base, "a"), filepath.Join(base, "a", "b", "c.log"), 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, depthBelow(tc.root, tc.pattern),
				"depthBelow(%q, %q)", tc.root, tc.pattern)
		})
	}
}

func TestBuildWalkGroups(t *testing.T) {
	base := t.TempDir()
	newScanner := func(paths ...string) *fileScanner {
		return &fileScanner{paths: paths, log: logptest.NewTestingLogger(t, "")}
	}

	t.Run("literal paths go to literals, not groups", func(t *testing.T) {
		lit := filepath.Join(base, "var", "log", "syslog")
		s := newScanner(lit)
		s.buildWalkGroups()

		assert.Equal(t, []string{lit}, s.literals)
		assert.Empty(t, s.walkGroups)
	})

	t.Run("expanded recursive set groups under one root", func(t *testing.T) {
		root := filepath.Join(base, "a")
		p1 := filepath.Join(root, "*.json")
		p2 := filepath.Join(root, "*", "*.json")
		p3 := filepath.Join(root, "*", "*", "*.json")
		s := newScanner(p1, p2, p3)
		s.buildWalkGroups()

		assert.Empty(t, s.literals)
		require.Contains(t, s.walkGroups, root)
		g := s.walkGroups[root]
		assert.Equal(t, root, g.root)
		assert.Equal(t, 3, g.maxDepth)
		assert.Equal(t, map[int][]string{1: {p1}, 2: {p2}, 3: {p3}}, g.byDepth)
	})

	t.Run("patterns sharing a root and depth are grouped together", func(t *testing.T) {
		root := filepath.Join(base, "a")
		pj := filepath.Join(root, "*.json")
		pn := filepath.Join(root, "*.ndjson")
		s := newScanner(pj, pn)
		s.buildWalkGroups()

		require.Contains(t, s.walkGroups, root)
		g := s.walkGroups[root]
		assert.Equal(t, 1, g.maxDepth)
		assert.Equal(t, map[int][]string{1: {pj, pn}}, g.byDepth)
	})

	t.Run("distinct roots produce distinct groups", func(t *testing.T) {
		ra, rb := filepath.Join(base, "a"), filepath.Join(base, "b")
		pa := filepath.Join(ra, "*.json")
		pb := filepath.Join(rb, "*.json")
		s := newScanner(pa, pb)
		s.buildWalkGroups()

		assert.Len(t, s.walkGroups, 2)
		require.Contains(t, s.walkGroups, ra)
		require.Contains(t, s.walkGroups, rb)
		assert.Equal(t, map[int][]string{1: {pa}}, s.walkGroups[ra].byDepth)
		assert.Equal(t, map[int][]string{1: {pb}}, s.walkGroups[rb].byDepth)
	})

	t.Run("mixes literals and globs", func(t *testing.T) {
		lit := filepath.Join(base, "exact.log")
		glob := filepath.Join(base, "a", "*.json")
		root := filepath.Join(base, "a")
		s := newScanner(lit, glob)
		s.buildWalkGroups()

		assert.Equal(t, []string{lit}, s.literals)
		require.Contains(t, s.walkGroups, root)
		assert.Equal(t, map[int][]string{1: {glob}}, s.walkGroups[root].byDepth)
	})

	t.Run("invalid pattern is skipped", func(t *testing.T) {
		bad := filepath.Join(base, "a", "[.json") // unterminated character class
		s := newScanner(bad)
		s.buildWalkGroups()

		assert.Empty(t, s.literals)
		assert.Empty(t, s.walkGroups)
	})
}

func TestWalk(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	mkfile := func(t *testing.T, path string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o770))
		require.NoError(t, os.WriteFile(path, []byte("data"), 0o660))
	}
	collect := func(g *walkGroup) []string {
		s := &fileScanner{log: logger}
		var got []string
		s.walk(g, func(f string, _ int) { got = append(got, f) }, func(string) {})
		return got
	}

	t.Run("matches by depth and bounds recursion", func(t *testing.T) {
		base := t.TempDir()
		mkfile(t, filepath.Join(base, "a.log"))                // depth 1
		mkfile(t, filepath.Join(base, "sub", "b.log"))         // depth 2
		mkfile(t, filepath.Join(base, "sub", "deep", "c.log")) // depth 3: beyond maxDepth

		got := collect(&walkGroup{
			root:     base,
			maxDepth: 2,
			byDepth: map[int][]string{
				1: {filepath.Join(base, "*.log")},
				2: {filepath.Join(base, "*", "*.log")},
			},
		})
		assert.ElementsMatch(t, []string{
			filepath.Join(base, "a.log"),
			filepath.Join(base, "sub", "b.log"),
		}, got)
	})

	t.Run("follows symlinked directories", func(t *testing.T) {
		base := t.TempDir()
		mkfile(t, filepath.Join(base, "real", "x.log"))
		require.NoError(t, os.Symlink(filepath.Join(base, "real"), filepath.Join(base, "link")))

		got := collect(&walkGroup{
			root:     base,
			maxDepth: 2,
			byDepth:  map[int][]string{2: {filepath.Join(base, "*", "*.log")}},
		})
		assert.ElementsMatch(t, []string{
			filepath.Join(base, "real", "x.log"),
			filepath.Join(base, "link", "x.log"),
		}, got)
	})

	t.Run("yields broken symlinks like glob", func(t *testing.T) {
		base := t.TempDir()
		mkfile(t, filepath.Join(base, "a.log"))
		require.NoError(t, os.Symlink(filepath.Join(base, "missing"), filepath.Join(base, "broken.log")))

		got := collect(&walkGroup{
			root:     base,
			maxDepth: 1,
			byDepth:  map[int][]string{1: {filepath.Join(base, "*.log")}},
		})
		// filepath.Glob does not stat entries at the last pattern component, so a
		// broken symlink is returned and later rejected by getIngestTarget.
		assert.ElementsMatch(t, []string{
			filepath.Join(base, "a.log"),
			filepath.Join(base, "broken.log"),
		}, got, "broken symlinks must be yielded like filepath.Glob and filtered later")
	})

	t.Run("yields dirs and symlinked dirs matching a leaf pattern", func(t *testing.T) {
		base := t.TempDir()
		mkfile(t, filepath.Join(base, "f.log"))
		require.NoError(t, os.Mkdir(filepath.Join(base, "targetdir"), 0o770))
		require.NoError(t, os.Symlink(filepath.Join(base, "targetdir"), filepath.Join(base, "linkdir")))

		got := collect(&walkGroup{
			root:     base,
			maxDepth: 1,
			byDepth:  map[int][]string{1: {filepath.Join(base, "*")}},
		})
		assert.ElementsMatch(t, []string{
			filepath.Join(base, "f.log"),
			filepath.Join(base, "linkdir"),
			filepath.Join(base, "targetdir"),
		}, got, "entries matching the pattern must be yielded regardless of type, like filepath.Glob")
	})

	t.Run("logs a malformed pattern once per walk", func(t *testing.T) {
		base := t.TempDir()
		mkfile(t, filepath.Join(base, "appx", "f1.log"))
		mkfile(t, filepath.Join(base, "appx", "f2.log"))

		inMemoryLog, buff := logp.NewInMemoryLocal("", logp.JSONEncoderConfig())
		sc := &fileScanner{log: inMemoryLog}
		var got []string
		// "app[" is a malformed pattern (unclosed character class) that
		// buildWalkGroups cannot detect upfront: matching it against "" fails on
		// the literal prefix before the parser reaches the bad token.
		sc.walk(&walkGroup{
			root:     base,
			maxDepth: 2,
			byDepth:  map[int][]string{2: {filepath.Join(base, "app[", "*.log")}},
		}, func(f string, _ int) { got = append(got, f) }, func(string) {})

		assert.Empty(t, got, "no file can match a malformed pattern")
		assert.Equalf(t, 1, strings.Count(buff.String(), "glob match("),
			"a malformed pattern must be logged once per walk, not once per file, got logs:\n%s", buff.String())
	})

	t.Run("prunes subtrees that cannot match", func(t *testing.T) {
		base := t.TempDir()
		mkfile(t, filepath.Join(base, "x", "app", "f.log"))
		mkfile(t, filepath.Join(base, "x", "other", "g.log"))
		mkfile(t, filepath.Join(base, "y", "app", "h.log"))

		got := collect(&walkGroup{
			root:     base,
			maxDepth: 3,
			byDepth:  map[int][]string{3: {filepath.Join(base, "*", "app", "*.log")}},
		})
		// Only files under the literal "app" component can match; subtrees such
		// as x/other must not contribute matches (and are not descended into).
		assert.ElementsMatch(t, []string{
			filepath.Join(base, "x", "app", "f.log"),
			filepath.Join(base, "y", "app", "h.log"),
		}, got, "only entries under directories matching the pattern components can match")
	})

	t.Run("missing root yields nothing", func(t *testing.T) {
		base := t.TempDir()
		got := collect(&walkGroup{
			root:     filepath.Join(base, "does-not-exist"),
			maxDepth: 1,
			byDepth:  map[int][]string{1: {filepath.Join(base, "does-not-exist", "*.log")}},
		})
		assert.Empty(t, got)
	})
}

func TestFileScannerDoesNotReportNotDirectoryAsUnobservable(t *testing.T) {
	root := t.TempDir()
	notDir := filepath.Join(root, "not-a-dir")
	require.NoError(t, os.WriteFile(notDir, []byte("hello\n"), 0o640))

	cfg := fileScannerConfig{Fingerprint: fingerprintConfig{Enabled: false}}
	s, err := newFileScanner(
		logptest.NewTestingLogger(t, ""),
		[]string{filepath.Join(notDir, "*.log")},
		cfg,
		CompressionNone,
	)
	require.NoError(t, err)

	files, metrics, unobservable := s.GetFiles(loginp.FileScanOptions{})
	assert.Empty(t, files, "a file used as a glob directory cannot contain matches")
	assert.Empty(t, unobservable, "ENOTDIR is a real disappearance signal, not a transient scan failure")
	assert.Equal(t, int64(0), metrics.ScanErrors, "ENOTDIR must not increment scan_errors")
}

func TestFileWatcherHarvesterMetrics(t *testing.T) {
	identifier, err := newFingerprintIdentifier(nil, logp.NewNopLogger())
	require.NoError(t, err, "failed to create fingerprint identifier")
	fw := &fileWatcher{
		fileIdentifier:   identifier,
		sourceIdentifier: mustSourceIdentifier("foo-id"),
		log:              logp.NewNopLogger(),
		events:           make(chan loginp.FSEvent, 10),
	}

	now := time.Now()
	oldModTime := now.Add(-2 * time.Hour)
	descriptor := func(name string, size int64, modTime time.Time, gzip bool) loginp.FileDescriptor {
		return loginp.FileDescriptor{
			Filename:    name,
			Fingerprint: loginp.FingerprintID{Sum: name},
			GZIP:        gzip,
			Info:        file.ExtendFileInfo(&testFileInfo{name: name, size: size, time: modTime}),
		}
	}
	paths := map[string]loginp.FileDescriptor{
		"complete":  descriptor("complete", 100, now, false),
		"near":      descriptor("near", 100, now, false),
		"lagging":   descriptor("lagging", 100, now, false),
		"no-active": descriptor("no-active", 100, now, false),
		"gzip":      descriptor("gzip", 100, now, true),
		"ignored":   descriptor("ignored", 100, oldModTime, false),
	}
	fw.prev = map[string]loginp.FileDescriptor{
		"complete":  descriptor("complete", 100, now, false),
		"near":      descriptor("near", 100, now, false),
		"lagging":   descriptor("lagging", 100, now, false),
		"no-active": descriptor("no-active", 100, now, false),
		"gzip":      descriptor("gzip", 100, now, true),
		"ignored":   descriptor("ignored", 100, oldModTime, false),
	}
	fw.scanner = &testFileScanner{files: paths}

	metrics := loginp.NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())
	// Register some files/offsets, like a harvester would do.
	completeOffset, cleanupCompleteOffset := metrics.RegisterHarvesterOffset(fw.getFileIdentity(paths["complete"]), 10)
	nearOffset, _ := metrics.RegisterHarvesterOffset(fw.getFileIdentity(paths["near"]), 5)
	laggingOffset, _ := metrics.RegisterHarvesterOffset(fw.getFileIdentity(paths["lagging"]), 4)
	gzipOffset, _ := metrics.RegisterHarvesterOffset(fw.getFileIdentity(paths["gzip"]), 10)
	ignoredOffset, _ := metrics.RegisterHarvesterOffset(fw.getFileIdentity(paths["ignored"]), 10)

	// Make sure the test uses the same atomic update path as harvesters.
	// Update to the actually expected values, like a harvester would do.
	completeOffset.Store(100)
	nearOffset.Store(95)
	laggingOffset.Store(94)
	gzipOffset.Store(100)
	ignoredOffset.Store(100)

	fw.watch(t.Context(), metrics, time.Hour, time.Time{})

	assert.EqualValues(t, 1, metrics.FilesIngestedPercent100.Get(), "files_ingested_percent_100")
	assert.EqualValues(t, 1, metrics.FilesIngestedPercent95To99.Get(), "files_ingested_percent_95_99")
	assert.EqualValues(t, 1, metrics.FilesIngestedPercentLt95.Get(), "files_ingested_percent_lt_95")

	// Copy paths and 'truncate' one file
	truncatedPaths := map[string]loginp.FileDescriptor{}
	for path, fd := range paths {
		truncatedPaths[path] = fd
	}
	truncatedPaths["complete"] = descriptor("complete", 50, now, false)
	fw.scanner = &testFileScanner{files: truncatedPaths}
	fw.watch(t.Context(), metrics, time.Hour, time.Time{})

	assert.EqualValues(t, 0, metrics.FilesIngestedPercent100.Get(), "files_ingested_percent_100 after truncation")
	assert.EqualValues(t, 1, metrics.FilesIngestedPercent95To99.Get(), "files_ingested_percent_95_99 after truncation")
	assert.EqualValues(t, 1, metrics.FilesIngestedPercentLt95.Get(), "files_ingested_percent_lt_95 after truncation")

	// Simulate the harvester restart caused by truncation.
	cleanupCompleteOffset()
	_, _ = metrics.RegisterHarvesterOffset(fw.getFileIdentity(truncatedPaths["complete"]), 0)

	// Copy truncatedPaths and make one file older
	ignoredPaths := map[string]loginp.FileDescriptor{}
	for path, fd := range truncatedPaths {
		ignoredPaths[path] = fd
	}
	ignoredPaths["near"] = descriptor("near", 100, oldModTime, false)
	fw.scanner = &testFileScanner{files: ignoredPaths}
	fw.watch(t.Context(), metrics, time.Hour, time.Time{})

	// The truncated file from the previous step is now a 'normal file at 50%'
	// The 'near' file (95% ingested) is ignored because of ignore_older
	assert.EqualValues(t, 0, metrics.FilesIngestedPercent100.Get(), "files_ingested_percent_100 after ignored")
	assert.EqualValues(t, 0, metrics.FilesIngestedPercent95To99.Get(), "files_ingested_percent_95_99 after ignored")
	assert.EqualValues(t, 2, metrics.FilesIngestedPercentLt95.Get(), "files_ingested_percent_lt_95 after ignored")

	// An update with no paths slice effectively removes all files from the
	// last update from the metrics
	fw.scanner = &testFileScanner{}
	fw.watch(t.Context(), metrics, time.Hour, time.Time{})

	assert.EqualValues(t, 0, metrics.FilesIngestedPercent100.Get(), "files_ingested_percent_100 after reset")
	assert.EqualValues(t, 0, metrics.FilesIngestedPercent95To99.Get(), "files_ingested_percent_95_99 after reset")
	assert.EqualValues(t, 0, metrics.FilesIngestedPercentLt95.Get(), "files_ingested_percent_lt_95 after reset")
}

func TestFileWatcherRunCleansHarvesterMetricsOnShutdown(t *testing.T) {
	identifier, err := newFingerprintIdentifier(nil, logp.NewNopLogger())
	require.NoError(t, err, "failed to create fingerprint identifier")

	now := time.Now()
	fd := loginp.FileDescriptor{
		Filename:    "complete",
		Fingerprint: loginp.FingerprintID{Sum: "complete"},
		Info:        file.ExtendFileInfo(&testFileInfo{name: "complete", size: 100, time: now}),
	}
	paths := map[string]loginp.FileDescriptor{
		"complete": fd,
	}

	fw := &fileWatcher{
		cfg:              fileWatcherConfig{Interval: time.Hour},
		prev:             map[string]loginp.FileDescriptor{"complete": fd},
		scanner:          &testFileScanner{files: paths},
		log:              logp.NewNopLogger(),
		events:           make(chan loginp.FSEvent, 1),
		notifyChan:       make(chan loginp.HarvesterStatus, 1),
		closedHarvesters: map[string]int64{},
		fileIdentifier:   identifier,
		sourceIdentifier: mustSourceIdentifier("foo-id"),
	}

	metrics := loginp.NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())
	sourceID := fw.getFileIdentity(fd)
	_, _ = metrics.RegisterHarvesterOffset(sourceID, 100)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	fw.Run(ctx, metrics, time.Hour, time.Time{})

	assert.EqualValues(t, 0, metrics.FilesIngestedPercent100.Get(), "files_ingested_percent_100 after watcher shutdown")
	assert.EqualValues(t, 0, metrics.FilesIngestedPercent95To99.Get(), "files_ingested_percent_95_99 after watcher shutdown")
	assert.EqualValues(t, 0, metrics.FilesIngestedPercentLt95.Get(), "files_ingested_percent_lt_95 after watcher shutdown")
}

// queuedScanner is an FSScanner test double that returns pre-programmed results,
// one per GetFiles call, so watcher behaviour can be driven scan by scan.
type queuedScanner struct {
	scans []scanResult
	next  int
}

type scanResult struct {
	files        map[string]loginp.FileDescriptor
	unobservable []string
}

func (q *queuedScanner) GetFiles(loginp.FileScanOptions) (map[string]loginp.FileDescriptor, loginp.FileScanMetrics, []string) {
	if q.next >= len(q.scans) {
		return map[string]loginp.FileDescriptor{}, loginp.FileScanMetrics{}, nil
	}
	r := q.scans[q.next]
	q.next++
	return r.files, loginp.FileScanMetrics{ScanErrors: int64(len(r.unobservable))}, r.unobservable
}

func newStubWatcher(scanner loginp.FSScanner) *fileWatcher {
	return &fileWatcher{
		log:              logp.NewNopLogger(),
		prev:             map[string]loginp.FileDescriptor{},
		scanner:          scanner,
		events:           make(chan loginp.FSEvent, 128),
		closedHarvesters: map[string]int64{},
		notifyChan:       make(chan loginp.HarvesterStatus, 5),
		fileIdentifier:   mustPathIdentifier(false),
		sourceIdentifier: mustSourceIdentifier("test-id"),
	}
}

// TestFileWatcherPostponesDeletesUnderUnobservablePaths is the watcher half of the
// fd-exhaustion fix: a previously seen file under a path the scan
// could not observe must not be reported deleted, otherwise its registry state is
// wiped and it is re-ingested once the resource frees up.
func TestFileWatcherPostponesDeletesUnderUnobservablePaths(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "a.log")
	subB := filepath.Join(base, "sub")
	b := filepath.Join(subB, "b.log")
	c := filepath.Join(base, "c.log")

	desc := func(path string, size int64) loginp.FileDescriptor {
		return loginp.FileDescriptor{
			Filename:    path,
			Fingerprint: completeFP("fp:" + path), // stable and unique per path so FileID is well-defined
			Info:        file.ExtendFileInfo(&testFileInfo{name: filepath.Base(path), size: size}),
		}
	}
	run := func(w *fileWatcher, m *loginp.Metrics) []loginp.FSEvent {
		w.watch(context.Background(), m, 0, time.Time{})
		return drainPendingFSEvents(w.events)
	}
	has := func(events []loginp.FSEvent, op loginp.Operation, oldPath, newPath string) bool {
		for _, e := range events {
			if e.Op == op && e.OldPath == oldPath && e.NewPath == newPath {
				return true
			}
		}
		return false
	}

	t.Run("postpone, carry forward, resume without re-create", func(t *testing.T) {
		s := &queuedScanner{scans: []scanResult{
			{files: map[string]loginp.FileDescriptor{a: desc(a, 5), b: desc(b, 5)}},                // healthy
			{files: map[string]loginp.FileDescriptor{a: desc(a, 5)}, unobservable: []string{subB}}, // B's dir unobservable
			{files: map[string]loginp.FileDescriptor{a: desc(a, 5), b: desc(b, 5)}},                // healthy again
		}}
		w := newStubWatcher(s)
		m := newTestMetrics()
		// scan_errors is a shared gauge, so compare against a baseline.
		base := m.ScanErrors.Get()

		ev1 := run(w, m)
		require.True(t, has(ev1, loginp.OpCreate, "", a), "scan1 should create A")
		require.True(t, has(ev1, loginp.OpCreate, "", b), "scan1 should create B")
		assert.Equal(t, base, m.ScanErrors.Get(), "healthy scan1 must not raise scan_errors")

		ev2 := run(w, m)
		assert.False(t, has(ev2, loginp.OpDelete, b, ""), "scan2 must NOT delete B: its directory was unobservable")
		_, tracked := w.prev[b]
		assert.True(t, tracked, "scan2 must carry B forward in prev")
		assert.Equal(t, base+1, m.ScanErrors.Get(), "scan2 must raise scan_errors for the unobservable dir")

		ev3 := run(w, m)
		assert.False(t, has(ev3, loginp.OpCreate, "", b), "scan3 must NOT re-create B: it stayed tracked")
		assert.False(t, has(ev3, loginp.OpDelete, b, ""), "scan3 must NOT delete B")
		assert.Equal(t, base, m.ScanErrors.Get(), "scan3 healthy again must decrement scan_errors back")
	})

	t.Run("rename out of a now-unobservable directory is a rename, not a re-create", func(t *testing.T) {
		oldDir := filepath.Join(base, "old")
		oldPath := filepath.Join(oldDir, "f.log")
		newPath := filepath.Join(base, "renamed.log")

		// Identical fingerprint at both paths => identical FileID => exact-FileID
		// rename, regardless of path.
		descFP := func(path string) loginp.FileDescriptor {
			return loginp.FileDescriptor{
				Filename:    path,
				Fingerprint: completeFP("same-content"),
				Info:        file.ExtendFileInfo(&testFileInfo{name: filepath.Base(path), size: 5}),
			}
		}

		s := &queuedScanner{scans: []scanResult{
			{files: map[string]loginp.FileDescriptor{oldPath: descFP(oldPath)}},                                 // scan1: track old
			{files: map[string]loginp.FileDescriptor{newPath: descFP(newPath)}, unobservable: []string{oldDir}}, // scan2: renamed; old dir unobservable
		}}
		w := newStubWatcher(s)
		m := newTestMetrics()

		run(w, m) // scan1: create old
		ev2 := run(w, m)

		assert.True(t, has(ev2, loginp.OpRename, oldPath, newPath),
			"renamed file must be a rename even though its old dir was unobservable")
		assert.False(t, has(ev2, loginp.OpCreate, "", newPath),
			"renamed file must NOT be re-created from offset 0")
		_, oldTracked := w.prev[oldPath]
		assert.False(t, oldTracked, "old path must not linger in prev after the rename")
		_, newTracked := w.prev[newPath]
		assert.True(t, newTracked, "new path must be tracked after the rename")
	})

	t.Run("growing rename+grow out of a now-unobservable directory is a rename, not a re-create", func(t *testing.T) {
		oldDir := filepath.Join(base, "growing-old")
		oldPath := filepath.Join(oldDir, "f.log")
		newPath := filepath.Join(base, "grown.log")

		// Growing mode: scan1 sees a sub-threshold (incomplete) file whose raw
		// fingerprint is a strict prefix of the completed fingerprint seen at the
		// new path in scan2 — the same file renamed AND grown across the threshold
		// in one scan. The exact-FileID pass cannot match it (the identity changes
		// on crossing the threshold), so it exercises the prefix-match pass.
		growing := loginp.FileDescriptor{
			Filename:    oldPath,
			Fingerprint: loginp.FingerprintID{Raw: "aabb"},
			Info:        file.ExtendFileInfo(&testFileInfo{name: filepath.Base(oldPath), size: 4}),
		}
		grown := loginp.FileDescriptor{
			Filename:    newPath,
			Fingerprint: loginp.FingerprintID{Raw: "aabbccdd", Sum: "sum-aabbccdd"},
			Info:        file.ExtendFileInfo(&testFileInfo{name: filepath.Base(newPath), size: 8}),
		}

		s := &queuedScanner{scans: []scanResult{
			{files: map[string]loginp.FileDescriptor{oldPath: growing}},
			{files: map[string]loginp.FileDescriptor{newPath: grown}, unobservable: []string{oldDir}},
		}}
		w := newStubWatcher(s)
		w.growingFingerprint = true
		m := newTestMetrics()

		run(w, m) // scan1: create old (still growing)
		ev2 := run(w, m)

		assert.True(t, has(ev2, loginp.OpRename, oldPath, newPath),
			"grown+renamed file must be a rename even though its old dir was unobservable")
		assert.False(t, has(ev2, loginp.OpCreate, "", newPath),
			"grown+renamed file must NOT be re-created from offset 0")
		_, oldTracked := w.prev[oldPath]
		assert.False(t, oldTracked, "old path must not linger in prev after the growing rename")
		_, newTracked := w.prev[newPath]
		assert.True(t, newTracked, "new path must be tracked after the growing rename")
	})

	t.Run("control: genuine disappearance still deletes", func(t *testing.T) {
		s := &queuedScanner{scans: []scanResult{
			{files: map[string]loginp.FileDescriptor{a: desc(a, 5), b: desc(b, 5)}},
			{files: map[string]loginp.FileDescriptor{a: desc(a, 5)}}, // B gone, nothing unobservable
		}}
		w := newStubWatcher(s)
		m := newTestMetrics()
		run(w, m)
		ev2 := run(w, m)
		assert.True(t, has(ev2, loginp.OpDelete, b, ""), "B must be deleted when nothing is unobservable")
	})

	t.Run("scoping: unobservable subtree does not mask a real delete elsewhere", func(t *testing.T) {
		s := &queuedScanner{scans: []scanResult{
			{files: map[string]loginp.FileDescriptor{a: desc(a, 5), b: desc(b, 5), c: desc(c, 5)}},
			{files: map[string]loginp.FileDescriptor{a: desc(a, 5)}, unobservable: []string{subB}}, // B unobservable, C really gone
		}}
		w := newStubWatcher(s)
		m := newTestMetrics()
		run(w, m)
		ev2 := run(w, m)
		assert.False(t, has(ev2, loginp.OpDelete, b, ""), "B under an unobservable dir must not be deleted")
		assert.True(t, has(ev2, loginp.OpDelete, c, ""), "C genuinely gone must still be deleted")
		_, tracked := w.prev[b]
		assert.True(t, tracked, "B must be carried forward")
	})

	t.Run("unobservable prefix with nothing tracked under it is a no-op", func(t *testing.T) {
		// B was never observed (e.g. its directory has been unreadable since the
		// first scan). The prefix is reported and counted, but there is no prev
		// entry to protect, so the watcher must neither invent one nor emit any
		// event for it — and the gauge must still decrement once it clears.
		s := &queuedScanner{scans: []scanResult{
			{files: map[string]loginp.FileDescriptor{a: desc(a, 5)}, unobservable: []string{subB}},
			{files: map[string]loginp.FileDescriptor{a: desc(a, 5)}}, // dir readable again, still nothing under it
		}}
		w := newStubWatcher(s)
		m := newTestMetrics()
		base := m.ScanErrors.Get()

		ev1 := run(w, m)
		assert.True(t, has(ev1, loginp.OpCreate, "", a), "A is new")
		assert.False(t, has(ev1, loginp.OpDelete, b, ""), "nothing tracked under the prefix, so nothing to delete")
		_, tracked := w.prev[b]
		assert.False(t, tracked, "a never-seen path must not be conjured into prev")
		assert.Equal(t, base+1, m.ScanErrors.Get(), "scan_errors still counts a never-observable path")

		run(w, m)
		assert.Equal(t, base, m.ScanErrors.Get(), "scan_errors must decrement once the path is observable again")
	})
}

func mustSourceIdentifier(inputID string) *loginp.SourceIdentifier {
	si, err := loginp.NewSourceIdentifier("filestream", inputID)
	if err != nil {
		// this will never happen
		panic(err)
	}

	return si
}

type testFileScanner struct {
	files map[string]loginp.FileDescriptor
}

// GetFiles returns s.files and empty metrics.
func (s *testFileScanner) GetFiles(loginp.FileScanOptions) (map[string]loginp.FileDescriptor, loginp.FileScanMetrics, []string) {
	return s.files, loginp.FileScanMetrics{}, nil
}

const benchmarkFileCount = 1000

func BenchmarkGetFiles(b *testing.B) {
	dir := b.TempDir()
	basenameFormat := "file-%d.log"

	for i := 0; i < benchmarkFileCount; i++ {
		filename := filepath.Join(dir, fmt.Sprintf(basenameFormat, i))
		content := fmt.Sprintf("content-%d\n", i)
		err := os.WriteFile(filename, []byte(strings.Repeat(content, 1024)), 0777)
		require.NoError(b, err)
	}
	paths := []string{filepath.Join(dir, "*.log")}
	cfg := fileScannerConfig{
		Fingerprint: fingerprintConfig{
			Enabled: false,
		},
	}
	s, err := newFileScanner(logp.NewNopLogger(), paths, cfg, CompressionNone)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		files, _, _ := s.GetFiles(loginp.FileScanOptions{})
		require.Len(b, files, benchmarkFileCount)
	}
}

func BenchmarkGetFilesWithFingerprint(b *testing.B) {
	dir := b.TempDir()
	basenameFormat := "file-%d.log"

	for i := 0; i < benchmarkFileCount; i++ {
		filename := filepath.Join(dir, fmt.Sprintf(basenameFormat, i))
		content := fmt.Sprintf("content-%d\n", i)
		err := os.WriteFile(filename, []byte(strings.Repeat(content, 1024)), 0777)
		require.NoError(b, err)
	}
	paths := []string{filepath.Join(dir, "*.log")}
	cfg := fileScannerConfig{
		Fingerprint: fingerprintConfig{
			Enabled: true,
			Offset:  0,
			Length:  1024,
		},
	}

	s, err := newFileScanner(logp.NewNopLogger(), paths, cfg, CompressionNone)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		files, _, _ := s.GetFiles(loginp.FileScanOptions{})
		require.Len(b, files, benchmarkFileCount)
	}
}

// BenchmarkGetFilesWithFingerprintGrowing measures repeated scans of stable,
// above-threshold files. Each GetFiles call is one scan, so b.N iterations model
// files that have grown past the fingerprint threshold and then sit unchanged
// across many scans — the scenario the sub-threshold growing benchmarks in
// BenchmarkFilestream do not exercise.
//
// The "static" and "growing" sub-benchmarks fingerprint the identical
// above-threshold files; their only difference is growing mode. The delta is the
// per-scan raw-header re-encode that growing mode performs on every completed
// file to bridge a possible threshold crossing. It calls GetFiles directly
// without advancing the completedFingerprints set, so it models the UNSUPPRESSED
// cost — exactly the work the watch loop elides on stable files by tracking
// completed paths. It therefore quantifies the per-scan cost that suppression
// removes after a file's first crossing scan.
func BenchmarkGetFilesWithFingerprintGrowing(b *testing.B) {
	cases := []struct {
		name    string
		growing bool
	}{
		{"static", false},
		{"growing", true},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			dir := b.TempDir()
			for i := range benchmarkFileCount {
				filename := filepath.Join(dir, fmt.Sprintf("file-%d.log", i))
				content := fmt.Sprintf("content-%d\n", i)
				// ~10KB per file: well above the 1024-byte threshold, so every
				// file is fingerprinted with a final SHA-256 on every scan.
				err := os.WriteFile(filename, []byte(strings.Repeat(content, 1024)), 0777)
				require.NoError(b, err)
			}
			paths := []string{filepath.Join(dir, "*.log")}
			cfg := fileScannerConfig{
				Fingerprint: fingerprintConfig{
					Enabled: true,
					Offset:  0,
					Length:  1024,
					Growing: tc.growing,
				},
			}
			s, err := newFileScanner(logp.NewNopLogger(), paths, cfg, CompressionNone)
			require.NoError(b, err)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				files, _, _ := s.GetFiles(loginp.FileScanOptions{})
				require.Len(b, files, benchmarkFileCount)
			}
		})
	}
}

func createWatcherWithConfig(t *testing.T, logger *logp.Logger, paths []string, cfgStr string) *fileWatcher {
	tmpCfg := struct {
		Scaner fileWatcherConfig `config:"scanner"`
	}{
		Scaner: defaultFileWatcherConfig(),
	}
	cfg, err := conf.NewConfigWithYAML([]byte(cfgStr), cfgStr)
	require.NoError(t, err)

	err = cfg.Unpack(&tmpCfg)
	require.NoError(t, err, "cannot unpack file watcher config")

	fw, err := newFileWatcher(
		logger,
		paths,
		tmpCfg.Scaner,
		CompressionNone,
		false,
		mustPathIdentifier(false),
		mustSourceIdentifier("foo-id"),
	)
	require.NoError(t, err)

	return fw
}

func createScannerWithConfig(t *testing.T, logger *logp.Logger, paths []string, cfgStr string, compression string) loginp.FSScanner {
	cfg, err := conf.NewConfigWithYAML([]byte(cfgStr), cfgStr)
	require.NoError(t, err)

	ns := &conf.Namespace{}
	err = ns.Unpack(cfg)
	require.NoError(t, err)

	config := defaultFileWatcherConfig()
	err = ns.Config().Unpack(&config)
	require.NoError(t, err)
	scanner, err := newFileScanner(logger, paths, config.Scanner, compression)
	require.NoError(t, err)

	return scanner
}

func requireEqualFiles(t *testing.T, expected, actual map[string]loginp.FileDescriptor) {
	t.Helper()
	require.Lenf(t, actual, len(expected), "amount of files does not match:\n\nexpected \n%v\n\n actual \n%v\n", filenames(expected), filenames(actual))

	for expFilename, expFD := range expected {
		actFD, exists := actual[expFilename]
		require.Truef(t, exists, "the actual file list is missing expected filename %s", expFilename)
		requireEqualDescriptors(t, expFD, actFD)
	}
}

func requireEqualEvents(t *testing.T, expected, actual loginp.FSEvent) {
	t.Helper()
	require.Equal(t, expected.NewPath, actual.NewPath, "NewPath")
	require.Equal(t, expected.OldPath, actual.OldPath, "OldPath")
	require.Equal(t, expected.Op, actual.Op, "Op")
	require.Equal(t, expected.SrcID, actual.SrcID, "SrcID")
	requireEqualDescriptors(t, expected.Descriptor, actual.Descriptor)
}

func requireEqualDescriptors(t *testing.T, expected, actual loginp.FileDescriptor) {
	t.Helper()
	require.Equal(t, expected.Filename, actual.Filename, "Filename")
	require.Equal(t, expected.Fingerprint, actual.Fingerprint, "Fingerprint")
	require.Equal(t, expected.Info.Name(), actual.Info.Name(), "Info.Name()")
	require.Equal(t, expected.Info.Size(), actual.Info.Size(), "Info.Size()")
}

// completeFP builds the FingerprintID a static (non-growing) fingerprint scan
// produces: a completed SHA-256 with no retained raw material.
func completeFP(sum string) loginp.FingerprintID {
	return loginp.FingerprintID{Sum: sum}
}

func filenames(m map[string]loginp.FileDescriptor) (result string) {
	for filename := range m {
		result += filename + "\n"
	}
	return result
}

func TestGetIngestTarget(t *testing.T) {
	t.Run("empty regular file", func(t *testing.T) {
		dir := t.TempDir()

		filename := filepath.Join(dir, "empty.log")
		err := os.WriteFile(filename, nil, 0644)
		require.NoError(t, err)

		cfg := fileScannerConfig{
			Symlinks:    false,
			Fingerprint: fingerprintConfig{Enabled: false},
		}
		s, err := newFileScanner(logp.NewNopLogger(), []string{filepath.Join(dir, "*.log")}, cfg, CompressionNone)
		require.NoError(t, err)

		_, err = s.getIngestTarget(filename)
		require.ErrorIs(t, err, errFileEmpty)
	})

	t.Run("symlink to an empty file", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "empty_target.txt")
		err := os.WriteFile(target, nil, 0644)
		require.NoError(t, err)

		link := filepath.Join(dir, "link.log")
		err = os.Symlink(target, link)
		require.NoError(t, err)

		cfg := fileScannerConfig{
			Symlinks:    true,
			Fingerprint: fingerprintConfig{Enabled: false},
		}
		s, err := newFileScanner(logp.NewNopLogger(), []string{filepath.Join(dir, "*.log")}, cfg, CompressionNone)
		require.NoError(t, err)

		_, err = s.getIngestTarget(link)
		require.ErrorIs(t, err, errFileEmpty)
	})
}

func TestToFileDescriptor_TooSmallFile_NoFileOpen(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "small.log")

	fingerprintLength := int64(1024)

	err := os.WriteFile(filename, []byte("a small file"), 0644)
	require.NoError(t, err, "failed to create test file")

	cfg := fileScannerConfig{
		Fingerprint: fingerprintConfig{
			Enabled: true,
			Offset:  0,
			Length:  fingerprintLength,
		},
	}

	s, err := newFileScanner(logp.NewNopLogger(), []string{filename}, cfg, CompressionNone)
	require.NoError(t, err, "failed to create scanner")
	it, err := s.getIngestTarget(filename)
	require.NoError(t, err, "getIngestTarget should succeed")

	// Remove read permissions - if file is opened, we'll get permission denied
	err = os.Chmod(filename, 0000)
	require.NoError(t, err, "failed to chmod test file")

	_, err = s.toFileDescriptor(&it)
	require.ErrorIs(t, err, errFileTooSmall,
		"expected errFileTooSmall, it probably tried to open the file")
}

// TestToFileDescriptor_GrowingLifecycle tests the Enhanced Fingerprint
// lifecycle through the scanner:
//
//  1. small file (below offset+length) under growing mode → raw-hex
//     FingerprintID.Raw, Complete=false, no Sum.
//  2. file grows past threshold → SHA-256 Sum matching the static fingerprint
//     output for the same bytes, Complete=true, and the raw header carried in
//     Raw (so the crossing can be prefix-matched against the predecessor).
//  3. second scan still at threshold via toFileDescriptor → identical
//     FingerprintID. The per-scan raw-header suppression lives in GetFiles
//     (exercised by TestGetFiles_GrowingRawSuppression), so a direct
//     toFileDescriptor call always recomputes Raw.
//  4. file truncated back below threshold → raw-hex Raw, Complete=false.
func TestToFileDescriptor_GrowingLifecycle(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "growing.log")

	const length int64 = 1024

	cfg := fileScannerConfig{
		Fingerprint: fingerprintConfig{
			Enabled: true,
			Offset:  0,
			Length:  length,
			Growing: true,
		},
	}
	s, err := newFileScanner(
		logp.NewNopLogger(), []string{filename}, cfg, CompressionNone)
	require.NoError(t, err, "could not create file scanner")

	writeFile := func(t *testing.T, n int) {
		t.Helper()
		// Use a non-trivial repeating pattern so prefix relationships are
		// preserved as the file grows.
		require.NoError(t, os.WriteFile(
			filename, []byte(strings.Repeat("abcd", n/4+1)[:n]), 0o644))
	}

	// --- Step 1: small file (200 bytes < 1024) ---
	writeFile(t, 200)
	it, err := s.getIngestTarget(filename)
	require.NoError(t, err, "getIngestTarget failed")

	fd1, err := s.toFileDescriptor(&it)
	require.NoError(t, err, "toFileDescriptor failed")

	assert.False(t, fd1.Fingerprint.Complete(), "step 1: sub-threshold file must not be Complete")
	assert.Equal(t,
		hex.EncodeToString([]byte(strings.Repeat("abcd", 51)[:200])),
		fd1.Fingerprint.Raw,
		"step 1: Raw should be raw hex of bytes[0:200]")
	assert.Empty(t, fd1.Fingerprint.Sum, "step 1: no SHA-256 Sum while below threshold")

	// --- Step 2: file grows past threshold (1500 bytes >= 1024) ---
	writeFile(t, 1500)
	it, err = s.getIngestTarget(filename)
	require.NoError(t, err, "getIngestTarget failed")

	fd2, err := s.toFileDescriptor(&it)
	require.NoError(t, err, "toFileDescriptor failed")

	assert.True(t, fd2.Fingerprint.Complete(),
		"step 2: descriptor must be Complete at/above threshold")

	expectedRawHex := hex.EncodeToString([]byte(strings.Repeat("abcd", 256)[:length]))
	expectedSHA := sha256.Sum256([]byte(strings.Repeat("abcd", 256)[:length]))
	assert.Equal(t, hex.EncodeToString(expectedSHA[:]), fd2.Fingerprint.Sum,
		"step 2: Sum must be SHA-256 of bytes[0:length]")
	assert.Equal(t, expectedRawHex, fd2.Fingerprint.Raw,
		"step 2: Raw must be raw hex of the same bytes for bridging the transition")

	// Verify the threshold-transition prefix relationship: the previous raw-hex
	// (200 bytes) is a prefix of the current completed Raw (1024 bytes). This is
	// the exact relation SameFile/rename detection rely on via Continues.
	assert.True(t, fd1.Fingerprint.Continues(fd2.Fingerprint),
		"step 2: prev raw-hex must be a prefix of the completed Raw")

	// --- Step 3: same file, second scan still at threshold ---
	it, err = s.getIngestTarget(filename)
	require.NoError(t, err, "getIngestTarget failed")

	fd3, err := s.toFileDescriptor(&it)
	require.NoError(t, err, "toFileDescriptor failed")

	assert.Equal(t, fd2.Fingerprint, fd3.Fingerprint,
		"step 3: direct toFileDescriptor recomputes the same FingerprintID "+
			"(GetFiles-level Raw suppression does not apply here)")

	// --- Step 4: file truncated back below threshold ---
	writeFile(t, 100)
	it, err = s.getIngestTarget(filename)
	require.NoError(t, err, "getIngestTarget failed")

	fd4, err := s.toFileDescriptor(&it)
	require.NoError(t, err, "toFileDescriptor failed")

	assert.False(t, fd4.Fingerprint.Complete(),
		"step 4: not Complete after truncation back below threshold")
	assert.NotEmpty(t, fd4.Fingerprint.Raw,
		"step 4: Raw set to the sub-threshold raw hex")
	assert.Empty(t, fd4.Fingerprint.Sum,
		"step 4: no SHA-256 Sum while below threshold")
}

// TestGetFiles_GrowingRawSuppression verifies the optimization that avoids
// recomputing the bridging raw header for files that are already complete: the
// header is emitted on the scan a file crosses the threshold (so the transition
// can still be prefix-matched) and dropped on subsequent scans of the now-stable
// file. If the file is truncated back below the threshold the suppression is
// lifted, and a fresh crossing re-emits the header.
//
// GetFiles is pure with respect to the completedFingerprints set; the watch
// loop is what advances it. The scan helper below reproduces that contract:
// run GetFiles, then hand the scanner the paths that are now complete, exactly
// as fileWatcher.watch does. The enumeration scans in the prospector's
// Init/TakeOver phases deliberately skip that second step, which is why they
// never suppress the header.
func TestGetFiles_GrowingRawSuppression(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "growing.log")
	const length int64 = 1024

	cfg := fileScannerConfig{
		Fingerprint: fingerprintConfig{
			Enabled: true,
			Offset:  0,
			Length:  length,
			Growing: true,
		},
	}
	s, err := newFileScanner(logp.NewNopLogger(), []string{filename}, cfg, CompressionNone)
	require.NoError(t, err, "could not create file scanner")

	writeFile := func(n int) {
		require.NoError(t, os.WriteFile(
			filename, []byte(strings.Repeat("abcd", n/4+1)[:n]), 0o644))
	}
	scan := func() loginp.FingerprintID {
		files, _, _ := s.GetFiles(loginp.FileScanOptions{})
		require.Contains(t, files, filename, "file must be scanned")
		fp := files[filename].Fingerprint
		// Mirror the watch loop: tell the scanner which paths are now complete
		// so the next scan can skip recomputing their bridging raw header.
		completed := map[string]struct{}{}
		for p, fd := range files {
			if fd.Fingerprint.Complete() {
				completed[p] = struct{}{}
			}
		}
		s.completedFingerprints = completed
		return fp
	}

	// Sub-threshold: growing, raw-hex carried as the identity.
	writeFile(200)
	fp := scan()
	assert.False(t, fp.Complete(), "sub-threshold file must not be Complete")
	assert.NotEmpty(t, fp.Raw, "sub-threshold file is identified by its raw header")

	// Crossing scan: Complete, and the bridging raw header is still present so a
	// growing predecessor can be prefix-matched.
	writeFile(1500)
	fp = scan()
	require.True(t, fp.Complete(), "file must be Complete at/above threshold")
	assert.NotEmpty(t, fp.Raw, "crossing scan must carry the bridging raw header")

	// Stable scans: still Complete, but the now-redundant raw header is dropped.
	for range 3 {
		fp = scan()
		require.True(t, fp.Complete(), "stable file stays Complete")
		assert.Empty(t, fp.Raw,
			"raw header must be suppressed once the file is already complete")
	}

	// Truncated back below threshold: suppression lifted, raw identity returns.
	writeFile(200)
	fp = scan()
	assert.False(t, fp.Complete(), "truncated file is growing again")
	assert.NotEmpty(t, fp.Raw, "sub-threshold file carries its raw header again")

	// A fresh crossing re-emits the bridging header (not suppressed by a stale
	// completed-set entry).
	writeFile(1500)
	fp = scan()
	require.True(t, fp.Complete(), "file is Complete after re-crossing")
	assert.NotEmpty(t, fp.Raw, "re-crossing must carry the bridging raw header again")
}

func BenchmarkToFileDescriptor(b *testing.B) {
	dir := b.TempDir()
	basename := "created.log"
	filename := filepath.Join(dir, basename)
	err := os.WriteFile(filename, []byte(strings.Repeat("a", 1024)), 0777)
	require.NoError(b, err)

	paths := []string{filename}
	cfg := fileScannerConfig{
		Fingerprint: fingerprintConfig{
			Enabled: true,
			Offset:  0,
			Length:  1024,
		},
	}

	s, err := newFileScanner(logp.NewNopLogger(), paths, cfg, CompressionNone)
	require.NoError(b, err)

	it, err := s.getIngestTarget(filename)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		fd, err := s.toFileDescriptor(&it)
		require.NoError(b, err)
		require.Equal(b, "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a", fd.Fingerprint.Sum)
	}
}

type logEntry struct {
	timestamp string
	level     string
	message   string
}

// parseLogs parsers the logs in buff and returns them as a slice of logEntry.
// It is meant to be used with `logp.NewInMemoryLocal` where buff is the
// contents of the buffer returned by `logp.NewInMemoryLocal`.
// Log entries are expected to be separated by a new line and each log entry
// is expected to have 3 fields separated by a tab "\t": timestamp, level
// and message.
func parseLogs(buff string) []logEntry {
	var logEntries []logEntry

	for l := range strings.SplitSeq(buff, "\n") {
		if l == "" {
			continue
		}

		split := strings.Split(l, "\t")
		if len(split) != 3 {
			continue
		}
		logEntries = append(logEntries, logEntry{
			timestamp: split[0],
			level:     split[1],
			message:   split[2],
		})
	}

	return logEntries
}
