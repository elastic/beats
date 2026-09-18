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

package filestream

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/file"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

<<<<<<< HEAD
=======
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

func TestUnderAnyPrefix(t *testing.T) {
	set := func(paths ...string) map[string]struct{} {
		native := make([]string, len(paths))
		for i, path := range paths {
			native[i] = filepath.FromSlash(path)
		}
		return pathSet(native)
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
			assert.Equal(t, tc.want, underAnyPrefix(filepath.FromSlash(tc.path), tc.prefixes),
				"underAnyPrefix(%q, %v)", filepath.FromSlash(tc.path), tc.prefixes)
		})
	}
}

func newTestMetrics() *loginp.Metrics {
	return loginp.NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())
}

>>>>>>> 7b2abed (filestream: preserve state for paths that vanish mid-scan (#53280))
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

	go fw.Run(ctx)

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

		go fw.Run(ctx)

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
				Fingerprint: "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a",
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

		go fw.Run(ctx)

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
			fw.Run(ctx)
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

		go fw.Run(ctx)

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
				Fingerprint: "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a",
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
			fw.Run(ctx)
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

func TestFileScanner(t *testing.T) {
	dir := t.TempDir()
	dir2 := t.TempDir() // for symlink testing
	paths := []string{filepath.Join(dir, "*.log")}

	normalBasename := "normal.log"
	undersizedBasename := "undersized.log"
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
	excludedFilename := filepath.Join(dir, excludedBasename)
	excludedIncludedFilename := filepath.Join(dir, excludedIncludedBasename)
	travelerFilename := filepath.Join(dir2, travelerBasename)
	normalSymlinkFilename := filepath.Join(dir, normalSymlinkBasename)
	exclSymlinkFilename := filepath.Join(dir, exclSymlinkBasename)
	travelerSymlinkFilename := filepath.Join(dir, travelerSymlinkBasename)

	files := map[string]string{
		normalFilename:           strings.Repeat("a", 1024),
		undersizedFilename:       strings.Repeat("a", 128),
		undersized1Filename:      strings.Repeat("1", 42),
		undersized2Filename:      strings.Repeat("2", 42),
		undersized3Filename:      strings.Repeat("3", 42),
		excludedFilename:         strings.Repeat("nothing to see here", 1024),
		excludedIncludedFilename: strings.Repeat("perhaps something to see here", 1024),
		travelerFilename:         strings.Repeat("folks, I think I got lost", 1024),
	}

	sizes := make(map[string]int64, len(files))
	for filename, content := range files {
		sizes[filename] = int64(len(content))
	}
	for filename, content := range files {
		err := os.WriteFile(filename, []byte(content), 0777)
		require.NoError(t, err)
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
		name    string
		cfgStr  string
		expDesc map[string]loginp.FileDescriptor
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
				undersizedFilename: {
					Filename: undersizedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedFilename],
						name: undersizedBasename,
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
				undersizedFilename: {
					Filename: undersizedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedFilename],
						name: undersizedBasename,
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
				undersizedFilename: {
					Filename: undersizedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedFilename],
						name: undersizedBasename,
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
				undersizedFilename: {
					Filename: undersizedFilename,
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[undersizedFilename],
						name: undersizedBasename,
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
			name: "returns all files except too small to fingerprint",
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
					Fingerprint: "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a",
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalFilename],
						name: normalBasename,
					}),
				},
				excludedFilename: {
					Filename:    excludedFilename,
					Fingerprint: "bd151321c3bbdb44185414a1b56b5649a00206dd4792e7230db8904e43987336",
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedFilename],
						name: excludedBasename,
					}),
				},
				excludedIncludedFilename: {
					Filename:    excludedIncludedFilename,
					Fingerprint: "bfdb99a65297062658c26dfcea816d76065df2a2da2594bfd9b96e9e405da1c2",
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedIncludedFilename],
						name: excludedIncludedBasename,
					}),
				},
				travelerSymlinkFilename: {
					Filename:    travelerSymlinkFilename,
					Fingerprint: "c4058942bffcea08810a072d5966dfa5c06eb79b902bf0011890dd8d22e1a5f8",
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
					Fingerprint: "ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb",
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalFilename],
						name: normalBasename,
					}),
				},
				// undersizedFilename got excluded because of the matching fingerprint
				excludedFilename: {
					Filename:    excludedFilename,
					Fingerprint: "9c225a1e6a7df9c869499e923565b93937e88382bb9188145f117195cd41dcd1",
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedFilename],
						name: excludedBasename,
					}),
				},
				excludedIncludedFilename: {
					Filename:    excludedIncludedFilename,
					Fingerprint: "7985b2b9750bdd3c76903db408aff3859204d6334279eaf516ecaeb618a218d5",
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[excludedIncludedFilename],
						name: excludedIncludedBasename,
					}),
				},
				travelerSymlinkFilename: {
					Filename:    travelerSymlinkFilename,
					Fingerprint: "da437600754a8eed6c194b7241b078679551c06c7dc89685a9a71be7829ad7e5",
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
			logger := logp.NewNopLogger()
			s := createScannerWithConfig(t, logger, paths, tc.cfgStr)
			requireEqualFiles(t, tc.expDesc, s.GetFiles())
		})
	}

	t.Run("issue a single warning with the number of files that are too small and debug with filenames", func(t *testing.T) {
		cfgStr := `
scanner:
  fingerprint:
    enabled: true
    offset: 0
    length: 1024
`

		// the glob for the very small files
		paths := []string{filepath.Join(dir, undersizedGlob)}
		logger, buffer := logp.NewInMemoryLocal("test-logger", zapcore.EncoderConfig{})

		s := createScannerWithConfig(t, logger, paths, cfgStr)
		files := s.GetFiles()
		require.Empty(t, files)
		files = s.GetFiles()
		require.Empty(t, files)
		files = s.GetFiles()
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
			mustPathIdentifier(false),
			mustSourceIdentifier("foo-id"),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "fingerprint size 1 bytes cannot be smaller than 64 bytes")
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
		s, err := newFileScanner(inMemoryLog, []string{filepath.Join(dir, "*.log")}, cfg)
		require.NoError(t, err)

		files := s.GetFiles()
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
		s, err := newFileScanner(inMemoryLog, []string{filepath.Join(dir, "*.log")}, cfg)
		require.NoError(t, err)

		files := s.GetFiles()
		assert.Len(t, files, 1, "empty_link.log must be excluded")
		assert.Contains(t, files, nonEmptyLink, "nonempty_link.log should be included")
		assert.NotContains(t, buff.String(), "GetFiles") // every line has a source prefix
	})

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
	logEntries := []logEntry{}

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

<<<<<<< HEAD
	return logEntries
=======
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

	files := s.GetFiles(loginp.FileScanOptions{}).Files
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

// collectingSink records the entries a walk matches and ignores the rest.
type collectingSink struct {
	matched      []string
	unobservable []string
	vanished     []string
}

func (c *collectingSink) process(filename string, _ int) { c.matched = append(c.matched, filename) }
func (c *collectingSink) recordUnobservable(prefix string) {
	c.unobservable = append(c.unobservable, prefix)
}
func (c *collectingSink) recordVanished(prefix string) { c.vanished = append(c.vanished, prefix) }

func TestWalk(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	mkfile := func(t *testing.T, path string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o770))
		require.NoError(t, os.WriteFile(path, []byte("data"), 0o660))
	}
	collect := func(g *walkGroup) []string {
		s := &fileScanner{log: logger}
		sink := &collectingSink{}
		s.walk(g, sink)
		return sink.matched
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
		sink := &collectingSink{}
		// "app[" is a malformed pattern (unclosed character class) that
		// buildWalkGroups cannot detect upfront: matching it against "" fails on
		// the literal prefix before the parser reaches the bad token.
		sc.walk(&walkGroup{
			root:     base,
			maxDepth: 2,
			byDepth:  map[int][]string{2: {filepath.Join(base, "app[", "*.log")}},
		}, sink)

		assert.Empty(t, sink.matched, "no file can match a malformed pattern")
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

	res := s.GetFiles(loginp.FileScanOptions{})
	assert.Empty(t, res.Files, "a file used as a glob directory cannot contain matches")
	assert.Empty(t, res.Unobservable, "ENOTDIR is a real disappearance signal, not a transient scan failure")
	assert.Equal(t, int64(0), res.Metrics.ScanErrors, "ENOTDIR must not increment scan_errors")
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
	maps.Copy(truncatedPaths, paths)
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
	maps.Copy(ignoredPaths, truncatedPaths)
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
	vanished     []string
}

func (q *queuedScanner) GetFiles(loginp.FileScanOptions) loginp.ScanResults {
	if q.next >= len(q.scans) {
		return loginp.ScanResults{Files: map[string]loginp.FileDescriptor{}}
	}
	r := q.scans[q.next]
	q.next++
	return loginp.ScanResults{
		Files:        r.files,
		Metrics:      loginp.FileScanMetrics{ScanErrors: int64(len(r.unobservable))},
		Unobservable: r.unobservable,
		Vanished:     r.vanished,
	}
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

// TestFileWatcherThrottlesPostponedWarning verifies the "postponing delete
// detection" warning is throttled: consecutive scans that keep postponing files
// under an unobservable path must log it at most once per postponedWarnInterval,
// not once per scan.
func TestFileWatcherThrottlesPostponedWarning(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "a.log")
	subB := filepath.Join(base, "sub")
	b := filepath.Join(subB, "b.log")
	desc := func(path string) loginp.FileDescriptor {
		return loginp.FileDescriptor{
			Filename:    path,
			Fingerprint: completeFP("fp:" + path),
			Info:        file.ExtendFileInfo(&testFileInfo{name: filepath.Base(path), size: 5}),
		}
	}
	// One healthy scan so B becomes tracked, then three scans that cannot observe
	// sub/, so B is postponed every time. Being rapid (well within
	// postponedWarnInterval) they must produce a single warning.
	s := &queuedScanner{scans: []scanResult{
		{files: map[string]loginp.FileDescriptor{a: desc(a), b: desc(b)}},
		{files: map[string]loginp.FileDescriptor{a: desc(a)}, unobservable: []string{subB}},
		{files: map[string]loginp.FileDescriptor{a: desc(a)}, unobservable: []string{subB}},
		{files: map[string]loginp.FileDescriptor{a: desc(a)}, unobservable: []string{subB}},
	}}
	inMemoryLog, buff := logp.NewInMemoryLocal("", logp.JSONEncoderConfig())
	w := newStubWatcher(s)
	w.log = inMemoryLog
	m := newTestMetrics()
	for range s.scans {
		w.watch(t.Context(), m, 0, time.Time{})
		drainPendingFSEvents(w.events)
	}

	assert.Equalf(t, 1, strings.Count(buff.String(), "postponing their"),
		"the postponed-delete warning must be throttled to once per interval, got logs:\n%s", buff.String())
}

func TestFileWatcherDoesNotWarnForVanishedPaths(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "a.log")
	b := filepath.Join(base, "b.log")
	desc := func(path string) loginp.FileDescriptor {
		return loginp.FileDescriptor{
			Filename:    path,
			Fingerprint: completeFP("fp:" + path),
			Info:        file.ExtendFileInfo(&testFileInfo{name: filepath.Base(path), size: 5}),
		}
	}
	s := &queuedScanner{scans: []scanResult{
		{files: map[string]loginp.FileDescriptor{a: desc(a), b: desc(b)}},
		{files: map[string]loginp.FileDescriptor{a: desc(a)}, vanished: []string{b}},
		{files: map[string]loginp.FileDescriptor{a: desc(a)}, vanished: []string{b}},
	}}
	inMemoryLog, buff := logp.NewInMemoryLocal("", logp.JSONEncoderConfig())
	w := newStubWatcher(s)
	w.log = inMemoryLog
	m := newTestMetrics()
	baseline := m.ScanErrors.Get()

	w.watch(t.Context(), m, 0, time.Time{})
	drainPendingFSEvents(w.events)
	w.watch(t.Context(), m, 0, time.Time{})
	events := drainPendingFSEvents(w.events)

	assert.Empty(t, events, "a vanished path must not produce a delete event")
	assert.Contains(t, w.prev, b, "a vanished path must keep its state for the next scan")
	assert.Equal(t, baseline, m.ScanErrors.Get(), "a vanished path must not raise scan_errors")
	assert.NotContainsf(t, buff.String(), "postponing their",
		"an ordinary rename or delete must not warn, got logs:\n%s", buff.String())
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
>>>>>>> 7b2abed (filestream: preserve state for paths that vanish mid-scan (#53280))
}

func mustSourceIdentifier(inputID string) *loginp.SourceIdentifier {
	si, err := loginp.NewSourceIdentifier("filestream", inputID)
	if err != nil {
		// this will never happen
		panic(err)
	}

	return si
}

const benchmarkFileCount = 1000

func BenchmarkGetFiles(b *testing.B) {
	dir := b.TempDir()
	basenameFormat := "file-%d.log"

	for i := range benchmarkFileCount {
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

	logger := logp.NewNopLogger()
	s, err := newFileScanner(logger, paths, cfg)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		files := s.GetFiles()
		require.Len(b, files, benchmarkFileCount)
	}
}

func BenchmarkGetFilesWithFingerprint(b *testing.B) {
	dir := b.TempDir()
	basenameFormat := "file-%d.log"

	for i := range benchmarkFileCount {
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

	logger := logp.NewNopLogger()
	s, err := newFileScanner(logger, paths, cfg)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		files := s.GetFiles()
		require.Len(b, files, benchmarkFileCount)
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
		mustPathIdentifier(false),
		mustSourceIdentifier("foo-id"),
	)
	require.NoError(t, err)

	return fw
}

func createScannerWithConfig(t *testing.T, logger *logp.Logger, paths []string, cfgStr string) loginp.FSScanner {
	cfg, err := conf.NewConfigWithYAML([]byte(cfgStr), cfgStr)
	require.NoError(t, err)

	ns := &conf.Namespace{}
	err = ns.Unpack(cfg)
	require.NoError(t, err)

	config := defaultFileWatcherConfig()
	err = ns.Config().Unpack(&config)
	require.NoError(t, err)

	scanner, err := newFileScanner(logger, paths, config.Scanner)
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
	requireEqualDescriptors(t, expected.Descriptor, actual.Descriptor)
}

func requireEqualDescriptors(t *testing.T, expected, actual loginp.FileDescriptor) {
	t.Helper()
	require.Equal(t, expected.Filename, actual.Filename, "Filename")
	require.Equal(t, expected.Fingerprint, actual.Fingerprint, "Fingerprint")
	require.Equal(t, expected.Info.Name(), actual.Info.Name(), "Info.Name()")
	require.Equal(t, expected.Info.Size(), actual.Info.Size(), "Info.Size()")
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
		s, err := newFileScanner(logp.NewNopLogger(), []string{filepath.Join(dir, "*.log")}, cfg)
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
		s, err := newFileScanner(logp.NewNopLogger(), []string{filepath.Join(dir, "*.log")}, cfg)
		require.NoError(t, err)

		_, err = s.getIngestTarget(link)
		require.ErrorIs(t, err, errFileEmpty)
	})
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

	logger := logp.NewNopLogger()
	s, err := newFileScanner(logger, paths, cfg)
	require.NoError(b, err)

	it, err := s.getIngestTarget(filename)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		fd, err := s.toFileDescriptor(&it)
		require.NoError(b, err)
		require.Equal(b, "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a", fd.Fingerprint)
	}
}
