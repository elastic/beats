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

	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/file"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
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

	fw := createWatcherWithConfig(t, paths, cfgStr)

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

		fw := createWatcherWithConfig(t, paths, cfgStr)
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
		requireEqualEvents(t, expEvent, e)
	})

	t.Run("does not emit events if a file is touched and resend_on_touch is disabled", func(t *testing.T) {
		dir := t.TempDir()
		paths := []string{filepath.Join(dir, "*.log")}
		cfgStr := `
scanner:
  check_interval: 10ms
`

		ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
		defer cancel()

		fw := createWatcherWithConfig(t, paths, cfgStr)
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
  check_interval: 10ms
`

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		logp.DevelopmentSetup(logp.ToObserverOutput())

		fw := createWatcherWithConfig(t, paths, cfgStr)
		go fw.Run(ctx)

		basename := "created.log"
		filename := filepath.Join(dir, basename)
		err := os.WriteFile(filename, nil, 0777)
		require.NoError(t, err)

		t.Run("issues a debug message in logs", func(t *testing.T) {
			expLogMsg := fmt.Sprintf("file %q has no content yet, skipping", filename)
			require.Eventually(t, func() bool {
				logs := logp.ObserverLogs().FilterLevelExact(logp.DebugLevel.ZapLevel()).TakeAll()
				if len(logs) == 0 {
					return false
				}
				for _, l := range logs {
					if strings.Contains(l.Message, expLogMsg) {
						return true
					}
				}
				return false
			}, 100*time.Millisecond, 10*time.Millisecond, "required a debug message %q but never found", expLogMsg)
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

		fw := createWatcherWithConfig(t, paths, cfgStr)
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
  check_interval: 100ms
`

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		logp.DevelopmentSetup(logp.ToObserverOutput())

		fw := createWatcherWithConfig(t, paths, cfgStr)

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

		logs := logp.ObserverLogs().FilterLevelExact(logp.WarnLevel.ZapLevel()).TakeAll()
		require.Lenf(t, logs, 0, "must be no warning messages, got: %v", logs)
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

	normalFilename := filepath.Join(dir, normalBasename)
	undersizedFilename := filepath.Join(dir, undersizedBasename)
	excludedFilename := filepath.Join(dir, excludedBasename)
	excludedIncludedFilename := filepath.Join(dir, excludedIncludedBasename)
	travelerFilename := filepath.Join(dir2, travelerBasename)
	normalSymlinkFilename := filepath.Join(dir, normalSymlinkBasename)
	exclSymlinkFilename := filepath.Join(dir, exclSymlinkBasename)
	travelerSymlinkFilename := filepath.Join(dir, travelerSymlinkBasename)

	files := map[string]string{
		normalFilename:           strings.Repeat("a", 1024),
		undersizedFilename:       strings.Repeat("a", 128),
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
			s := createScannerWithConfig(t, paths, tc.cfgStr)
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
		logp.DevelopmentSetup(logp.ToObserverOutput())

		// this file is 128 bytes long
		paths := []string{filepath.Join(dir, undersizedBasename)}
		s := createScannerWithConfig(t, paths, cfgStr)
		files := s.GetFiles()
		require.Empty(t, files)
		logs := logp.ObserverLogs().FilterLevelExact(logp.WarnLevel.ZapLevel()).TakeAll()
		require.Empty(t, logs, "there must be no warning logs for files too small")
	})

	t.Run("returns error when creating scanner with a fingerprint too small", func(t *testing.T) {
		cfgStr := `
scanner:
  fingerprint:
    enabled: true
    offset: 0
    length: 1
`
		cfg, err := conf.NewConfigWithYAML([]byte(cfgStr), cfgStr)
		require.NoError(t, err)

		ns := &conf.Namespace{}
		err = ns.Unpack(cfg)
		require.NoError(t, err)

		_, err = newFileWatcher(paths, ns)
		require.Error(t, err)
		require.Contains(t, err.Error(), "fingerprint size 1 bytes cannot be smaller than 64 bytes")
	})
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
	s, err := newFileScanner(paths, cfg)
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
	s, err := newFileScanner(paths, cfg)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		files := s.GetFiles()
		require.Len(b, files, benchmarkFileCount)
	}
}

func createWatcherWithConfig(t *testing.T, paths []string, cfgStr string) loginp.FSWatcher {
	cfg, err := conf.NewConfigWithYAML([]byte(cfgStr), cfgStr)
	require.NoError(t, err)

	ns := &conf.Namespace{}
	err = ns.Unpack(cfg)
	require.NoError(t, err)

	fw, err := newFileWatcher(paths, ns)
	require.NoError(t, err)

	return fw
}

func createScannerWithConfig(t *testing.T, paths []string, cfgStr string) loginp.FSScanner {
	cfg, err := conf.NewConfigWithYAML([]byte(cfgStr), cfgStr)
	require.NoError(t, err)

	ns := &conf.Namespace{}
	err = ns.Unpack(cfg)
	require.NoError(t, err)

	config := defaultFileWatcherConfig()
	err = ns.Config().Unpack(&config)
	require.NoError(t, err)
	scanner, err := newFileScanner(paths, config.Scanner)
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
	s, err := newFileScanner(paths, cfg)
	require.NoError(b, err)

	it, err := s.getIngestTarget(filename)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		fd, err := s.toFileDescriptor(&it)
		require.NoError(b, err)
		require.Equal(b, "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a", fd.Fingerprint)
	}
}
