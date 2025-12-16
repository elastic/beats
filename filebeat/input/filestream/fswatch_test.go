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
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/file"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

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

		inMemoryLog, buff := logp.NewInMemoryLocal("", logp.JSONEncoderConfig())
		fw := createWatcherWithConfig(t, inMemoryLog, paths, cfgStr)
		go fw.Run(ctx)

		basename := "created.log"
		filename := filepath.Join(dir, basename)
		err := os.WriteFile(filename, nil, 0777)
		require.NoError(t, err)

		t.Run("issues a debug message in logs", func(t *testing.T) {
			expLogMsg := fmt.Sprintf("file %q has no content yet, skipping", filename)
			require.Eventually(t, func() bool {
				return strings.Contains(buff.String(), expLogMsg)
			}, time.Second, 10*time.Millisecond, "required a debug message %q but never found", expLogMsg)
		})

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

		go fw.Run(ctx)

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
					Fingerprint: "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a",
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalFilename],
						name: normalBasename,
					}),
				},
				normalGZIPFilename: {
					Filename:    normalGZIPFilename,
					Fingerprint: "af1ee623faf25c42385da9f1bc222a3ccfd6722d6d6bcdc78538215d479b7ac7",
					Info: file.ExtendFileInfo(&testFileInfo{
						size: sizes[normalGZIPFilename],
						name: normalGZIPBasename,
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
			logger := logptest.NewTestingLogger(t, "")
			s := createScannerWithConfig(t, logger, paths, tc.cfgStr, tc.compression)
			requireEqualFiles(t, tc.expDesc, s.GetFiles())
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
		files := s.GetFiles()
		require.Empty(t, files)

		logs := parseLogs(buffer.String())
		require.NotEmpty(t, logs, "fileScanner.GetFiles must log some warnings")

		// The last log entry from s.GetFiles must be at warn level and
		// in the format 'x files are too small"
		lastEntry := logs[len(logs)-1]
		require.Equal(t, "warn", lastEntry.level, "'x files are too small' must be at level warn")
		require.Contains(t, lastEntry.message, "3 files are too small to be ingested")

		// For each file that is too small to be ingested, s.GetFiles must log
		// at debug level the filename and its size
		expectedMsgs := []string{
			fmt.Sprintf("cannot start ingesting from file %[1]q: filesize of %[1]q is 42 bytes", undersized1Filename),
			fmt.Sprintf("cannot start ingesting from file %[1]q: filesize of %[1]q is 42 bytes", undersized2Filename),
			fmt.Sprintf("cannot start ingesting from file %[1]q: filesize of %[1]q is 42 bytes", undersized3Filename),
		}

		for _, msg := range expectedMsgs {
			found := false
			for _, log := range logs {
				if strings.HasPrefix(log.message, msg) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("did not find %q in the logs", msg)
			}
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
}

func mustFingerprintIdentifier() fileIdentifier {
	fi, _ := newFingerprintIdentifier(nil, nil)

	return fi
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
		files := s.GetFiles()
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
	require.Equalf(t, len(expected), len(actual), "amount of files does not match:\n\nexpected \n%v\n\n actual \n%v\n", filenames(expected), filenames(actual))

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

func filenames(m map[string]loginp.FileDescriptor) (result string) {
	for filename := range m {
		result += filename + "\n"
	}
	return result
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
		require.Equal(b, "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a", fd.Fingerprint)
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
