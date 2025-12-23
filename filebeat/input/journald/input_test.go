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

//go:build linux

package journald

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalctl"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestInputCanReadAllBoots(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "multiple-boots.journal.gz"))

	env := newInputTestingEnvironment(t)
	cfg := mapstr.M{
		"paths": []string{out},
	}
	inp := env.mustCreateInput(cfg)

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)

	env.startInput(ctx, inp)
	env.waitUntilEventCount(6)
}

func TestInputFieldsTranslation(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "input-multiline-parser.journal.gz"))
	// A few random keys to verify
	keysToCheck := map[string]string{
		"systemd.user_unit": "log-service.service",
		"process.pid":       "2084785",
		"systemd.transport": "stdout",
		"host.hostname":     "x-wing",
	}

	testCases := map[string]struct {
		saveRemoteHostname bool
	}{
		"Save hostname enabled":  {saveRemoteHostname: true},
		"Save hostname disabled": {saveRemoteHostname: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)

			inp := env.mustCreateInput(mapstr.M{
				"paths":                 []string{out},
				"include_matches.match": []string{"_SYSTEMD_USER_UNIT=log-service.service"},
				"save_remote_hostname":  tc.saveRemoteHostname,
			})

			ctx, cancelInput := context.WithCancel(context.Background())
			env.startInput(ctx, inp)
			env.waitUntilEventCount(6)

			for eventIdx, event := range env.pipeline.clients[0].GetEvents() {
				for k, v := range keysToCheck {
					got, err := event.Fields.GetValue(k)
					if err == nil {
						if got, want := fmt.Sprint(got), v; got != want {
							t.Errorf("expecting key %q to have value '%#v', but got '%#v' instead", k, want, got)
						}
					} else {
						t.Errorf("key %q not found on event %d", k, eventIdx)
					}
				}
				if tc.saveRemoteHostname {
					v, err := event.Fields.GetValue("log.source.address")
					if err != nil {
						t.Errorf("key 'log.source.address' not found on evet %d", eventIdx)
					}

					if got, want := fmt.Sprint(v), "x-wing"; got != want {
						t.Errorf("expecting key 'log.source.address' to have value '%#v', but got '%#v' instead", want, got)
					}
				}
			}
			cancelInput()
		})
	}
}

// TestCompareGoSystemdWithJournalctl ensures the new implementation produces
// events in the same format as the original one. We use the events from the
// already existing journal file 'input-multiline-parser.journal'
//
// Generating golden file: to generate the golden file you need to copy
// and run this test on a older version that still uses go-systemd,
// like 8.16.0, so the input run on this older version, call
// `env.pipeline.GetAllEvents()`, get the events, marshal them as
// JSON with "  " as the indent argument and write it to the file.
//
// The following fields are not currently tested:
// __CURSOR - it is added to the registry and there are other tests for it
// __MONOTONIC_TIMESTAMP - it is part of the cursor
func TestCompareGoSystemdWithJournalctl(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "input-multiline-parser.journal.gz"))
	env := newInputTestingEnvironment(t)
	inp := env.mustCreateInput(mapstr.M{
		"paths": []string{out},
		"seek":  "head",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	defer cancelInput()

	env.startInput(ctx, inp)
	env.waitUntilEventCount(8)

	rawEvents := env.pipeline.GetAllEvents()
	events := []beat.Event{}
	for _, evt := range rawEvents {
		_ = evt.Delete("event.created")
		// Fields that the go-systemd version did not add
		_ = evt.Delete("journald.custom.seqnum")
		_ = evt.Delete("journald.custom.seqnum_id")
		_ = evt.Delete("journald.custom.realtime_timestamp")
		// Marshal and Unmarshal because of type changes
		// We ignore errors as those types can always marshal and unmarshal
		data, _ := json.Marshal(evt)
		newEvt := beat.Event{}
		json.Unmarshal(data, &newEvt) //nolint: errcheck // this will never fail
		if newEvt.Meta == nil {
			// the golden file has it as an empty map
			newEvt.Meta = mapstr.M{}
		}
		events = append(events, newEvt)
	}

	// Read JSON events
	goldenEvents := []beat.Event{}
	data, err := os.ReadFile(filepath.Join("testdata", "input-multiline-parser-events.json"))
	if err != nil {
		t.Fatalf("cannot read golden file: %s", err)
	}

	if err := json.Unmarshal(data, &goldenEvents); err != nil {
		t.Fatalf("cannot unmarshal golden events: %s", err)
	}

	if len(events) != len(goldenEvents) {
		t.Fatalf("expecting %d events, got %d", len(goldenEvents), len(events))
	}

	// The timestamps can have different locations set, but still be equal,
	// this causes the require.EqualValues to fail, so we compare them manually
	// and set them all to the same time.
	for i, goldEvent := range goldenEvents {
		// We have compared the length already, both slices have
		// have the same number of elements
		evt := events[i]
		if !goldEvent.Timestamp.Equal(evt.Timestamp) {
			t.Errorf(
				"event %d timestamp is different than expected. Expecting %s, got %s",
				i, goldEvent.Timestamp.String(), evt.Timestamp.String())
		}

		events[i].Timestamp = goldEvent.Timestamp
	}

	require.EqualValues(t, goldenEvents, events, "events do not match reference")
}

func TestMatchers(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "matchers.journal.gz"))
	// If this test fails, uncomment the following line to see the debug logs
	// logp.DevelopmentSetup()
	testCases := []struct {
		name           string
		matchers       map[string]any
		confiFields    map[string]any
		expectedEvents int
	}{
		{ // FOO=foo
			name: "single marcher",
			matchers: map[string]any{
				"match": []string{
					"FOO=foo",
				},
			},
			expectedEvents: 2,
		},
		{ // FOO=foo AND BAR=bar
			name: "different keys work as AND",
			matchers: map[string]any{
				"match": []string{
					"FOO=foo",
					"BAR=bar",
				},
			},
			expectedEvents: 1,
		},
		{ // FOO_BAR=foo OR FOO_BAR=bar
			name: "same keys work as OR",
			matchers: map[string]any{
				"match": []string{
					"FOO_BAR=foo",
					"FOO_BAR=bar",
				},
			},
			expectedEvents: 2,
		},
		{ // (FOO_BAR=foo OR FOO_BAR=bar) AND message="message 4"
			name: "same keys work as OR, AND the odd one, one match",
			matchers: map[string]any{
				"match": []string{
					"FOO_BAR=foo",
					"FOO_BAR=bar",
					"MESSAGE=message 4",
				},
			},
			expectedEvents: 1,
		},
		{ // (FOO_BAR=foo OR FOO_BAR=bar) AND message="message 1"
			name: "same keys work as OR, AND the odd one. No matches",
			matchers: map[string]any{
				"match": []string{
					"FOO_BAR=foo",
					"FOO_BAR=bar",
					"MESSAGE=message 1",
				},
			},
			expectedEvents: 0,
		},
		{
			name:     "transport: journal",
			matchers: map[string]any{},
			confiFields: map[string]any{
				"transports": []string{"journal"},
			},
			expectedEvents: 6,
		},
		{
			name:     "syslog identifier: sudo",
			matchers: map[string]any{},
			confiFields: map[string]any{
				"syslog_identifiers": []string{"sudo"},
			},
			expectedEvents: 1,
		},
		{
			name:     "unit",
			matchers: map[string]any{},
			confiFields: map[string]any{
				"units": []string{"session-39.scope"},
			},
			expectedEvents: 7,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)
			cfg := mapstr.M{
				"paths":           []string{out},
				"include_matches": tc.matchers,
			}
			cfg.Update(mapstr.M(tc.confiFields))
			inp := env.mustCreateInput(cfg)

			ctx, cancelInput := context.WithCancel(context.Background())
			defer cancelInput()

			env.startInput(ctx, inp)
			env.waitUntilEventCount(tc.expectedEvents)
		})
	}
}

//go:embed pkg/journalctl/testdata/corner-cases.json
var msgByteArrayJSON []byte

func TestReaderAdapterCanHandleNonStringFields(t *testing.T) {
	testCases := []map[string]any{}
	if err := json.Unmarshal(msgByteArrayJSON, &testCases); err != nil {
		t.Fatalf("could not unmarshal the contents from 'testdata/message-byte-array.json' into map[string]any: %s", err)
	}

	for idx, event := range testCases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			mock := journalReaderMock{
				NextFunc: func(cancel v2.Canceler) (journalctl.JournalEntry, error) {
					return journalctl.JournalEntry{
						Fields: event,
					}, nil
				}}
			ra := readerAdapter{
				r:         &mock,
				converter: journalfield.NewConverter(logp.L(), nil),
				canceler:  context.Background(),
			}

			evt, err := ra.Next()
			if err != nil {
				t.Fatalf("readerAdapter.Next must succeed, got an error: %s", err)
			}
			if len(evt.Content) == 0 {
				t.Fatal("event.Content must be populated")
			}
		})
	}
}

func TestInputCanReportStatus(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "multiple-boots.journal.gz"))

	env := newInputTestingEnvironment(t)
	cfg := mapstr.M{
		"paths": []string{out},
	}
	inp := env.mustCreateInput(cfg)

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)

	env.startInput(ctx, inp)
	env.waitUntilEventCount(6)

	env.RequireStatuses([]statusUpdate{
		{
			state: status.Starting,
			msg:   "Starting",
		},
		{
			state: status.Running,
			msg:   "Running",
		},
	})
}

var expectedBinaryMessges = [][]byte{
	{
		0, 2, 4, 8, 10, 12, 14, 16, 18,
	},
	{
		0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100,
	},
	{
		0xED, 0xA0, 0xBC, 0xED, 0xBF, 0xA0, 0xED, 0xA0, 0xBD, 0xED, 0xB1, 0x81,
		0xEF, 0xB8, 0x8F, 0xED, 0xA0, 0xBE, 0xED, 0xBA, 0xB5, 0xED, 0xA0, 0xBE,
		0xED, 0xBA, 0xB5, 0xED, 0xA0, 0xBD, 0xED, 0xBF, 0xA0, 0xE2, 0xA0, 0x80,
		0xED, 0xA0, 0xBC, 0xED, 0xBC, 0x8A, 0xED, 0xA0, 0xBD, 0xED, 0xBF, 0xA0,
		0xED, 0xA0, 0xBC, 0xED, 0xBE, 0x80, 0xED, 0xA0, 0xBE, 0xED, 0xBA, 0xB5,
		0xED, 0xA0, 0xBD, 0xED, 0xB2, 0xA7, 0xE2, 0x9D, 0x97,
	},
	[]byte(`FOO\nBAR\nFOO`),
	{
		240, 159, 143, 160, 240, 159, 145, 129, 239, 184, 143, 240, 159, 170,
		181, 240, 159, 170, 181, 240, 159, 159, 160, 226, 160, 128, 240, 159,
		140, 138, 240, 159, 159, 160, 240, 159, 142, 128, 240, 159, 170, 181,
		240, 159, 146, 167, 226, 157, 151,
	},
	{
		27, 91, 63, 50, 48, 48, 52, 104, 114, 111, 111, 116, 64, 55, 97, 97,
		56, 48, 97, 98, 54, 101, 97, 99, 52, 58, 47, 35, 32, 101, 99, 104, 111,
		32, 102, 111, 111, 32, 98, 97, 114, 13,
	},
	{
		27, 91, 63, 50, 48, 48, 52, 108, 13, 102, 111, 111, 32, 98, 97, 114, 13,
	},
	{
		27, 91, 63, 50, 48, 48, 52, 104, 114, 111, 111, 116, 64, 55, 97, 97, 56,
		48, 97, 98, 54, 101, 97, 99, 52, 58, 47, 35, 32, 101, 120, 105, 116, 13,
	},
	{
		27, 91, 63, 50, 48, 48, 52, 108, 13, 101, 120, 105, 116, 13,
	},
}

func TestBinaryDataIsCorrectlyHandled(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "binary.journal.gz"))

	env := newInputTestingEnvironment(t)
	cfg := mapstr.M{
		"paths": []string{out},
	}
	inp := env.mustCreateInput(cfg)

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)

	env.startInput(ctx, inp)
	env.waitUntilEventCount(len(expectedBinaryMessges))
	events := env.pipeline.GetAllEvents()
	for i, evt := range events {
		msg := []byte(evt.Fields["message"].(string)) //nolint:errcheck // we know it's a string.
		if !bytes.Equal(expectedBinaryMessges[i], msg) {
			t.Errorf("expecting entry %d to be:\n%#v\ngot:\n%#v", i, expectedBinaryMessges[i], msg)
		}
	}
}

// TestPathIsFolder ensures the Journald input works when a folder is passed
// in paths. The desired behaviour is that the input will ingest all entries
// from existing files and new files that might appear in the future.
//
// This is implemented by using `--directory` when a directory is in the paths
// setting. Because the way journalctl works, by default, it only reads entries
// from a single journal, so if manually testing or modifying the files used by
// this test, ensure all journal files belong to the same journal and new files
// have entries that are ahead in time from old files.
func TestPathIsFolder(t *testing.T) {
	srcDir := decompressAll(t, filepath.Join("testdata", "journal*.journal.gz"))
	dstDir := t.TempDir()

	srcFiles := []string{}
	dstFiles := []string{}
	for i := range 3 {
		fName := fmt.Sprintf("journal%d.journal", i+1)
		srcFiles = append(srcFiles, filepath.Join(srcDir, fName))
		dstFiles = append(dstFiles, filepath.Join(dstDir, fName))
	}

	env := newInputTestingEnvironment(t)
	cfg := mapstr.M{
		"paths": []string{dstDir},
	}
	inp := env.mustCreateInput(cfg)

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)

	env.startInput(ctx, inp)

	for i := range 3 {
		if err := os.Rename(srcFiles[i], dstFiles[i]); err != nil {
			t.Fatalf("cannot move file: %s", err)
		}

		env.waitUntilEventCount(10 + i*10)
	}
}

func TestDoubleStarCanBeUsed(t *testing.T) {
	srcDir := decompressAll(t, filepath.Join("testdata", "journal*.journal.gz"))
	dstDir := t.TempDir()

	srcFiles := []string{}
	dstFiles := []string{}
	for i := range 3 {
		fName := fmt.Sprintf("journal%d.journal", i+1)
		srcFiles = append(srcFiles, filepath.Join(srcDir, fName))
		dstFiles = append(dstFiles, filepath.Join(t.TempDir(), fName))
	}

	// We want to test a glob in the format:
	// /tmp/TestFoo/*/*
	// To match files like
	//   - /tmp/TestFoo/001/journal1.journal
	//   - /tmp/TestFoo/001/journal2.journal
	// So we construct the glob from dstDir

	split := strings.Split(dstDir, "/")
	split = split[:len(split)-1]
	split = append(split, "*", "*")
	path := filepath.Join(split...)
	path = string(filepath.Separator) + path // Add the leading separator

	env := newInputTestingEnvironment(t)
	cfg := mapstr.M{
		"paths": []string{path},
	}

	inp := env.mustCreateInput(cfg)
	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)

	for i := range len(srcFiles) {
		if err := os.Rename(srcFiles[i], dstFiles[i]); err != nil {
			t.Fatalf("cannot move file: %s", err)
		}
	}

	env.startInput(ctx, inp)
	env.waitUntilEventCount(len(srcFiles) * 10)
}

func TestConfigureJournalctlPath(t *testing.T) {
	t.Run("default path without chroot", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(mapstr.M{})
		_, inp, err := Configure(cfg, logp.NewNopLogger())
		require.NoError(t, err, "Configure should succeed without chroot")
		jd, ok := inp.(*journald)
		require.True(t, ok, "input should be of type *journald")
		assert.Equal(t, defaultJournalCtlPath, jd.JournalctlPath, "should use default journalctl path when chroot is not set")
	})

	t.Run("chroot with default path auto-switches to chroot default", func(t *testing.T) {
		chrootDir := t.TempDir()
		cfg := conf.MustNewConfigFrom(mapstr.M{
			"chroot": chrootDir,
		})
		_, _, err := Configure(cfg, logp.NewNopLogger())
		require.Error(t, err, "Configure should fail when journalctl binary doesn't exist in chroot")
		assert.Contains(t, err.Error(), "cannot stat journalctl binary in chroot", "error should indicate journalctl binary is missing")
	})

	t.Run("chroot with relative path returns error", func(t *testing.T) {
		chrootDir := t.TempDir()
		cfg := conf.MustNewConfigFrom(mapstr.M{
			"chroot":          chrootDir,
			"journalctl_path": "relative/path/journalctl",
		})
		_, _, err := Configure(cfg, logp.NewNopLogger())
		require.Error(t, err, "Configure should fail with relative path when chroot is set")
		assert.Contains(t, err.Error(), "journalctl_path must be an absolute path when chroot is set", "error should indicate path must be absolute")
	})

	t.Run("chroot with absolute path", func(t *testing.T) {
		chrootDir := t.TempDir()
		journalctlPath := "/usr/bin/journalctl"
		cfg := conf.MustNewConfigFrom(mapstr.M{
			"chroot":          chrootDir,
			"journalctl_path": journalctlPath,
		})
		_, inp, err := Configure(cfg, logp.NewNopLogger())
		require.Error(t, err, "Configure should fail when journalctl binary doesn't exist in chroot")
		assert.Contains(t, err.Error(), "cannot stat journalctl binary in chroot", "error should indicate journalctl binary is missing")
		jd, ok := inp.(*journald)
		if ok {
			assert.Equal(t, journalctlPath, jd.JournalctlPath, "journalctl path should be set correctly even when binary doesn't exist")
		}
	})

	t.Run("chroot auto-switches default to chroot default", func(t *testing.T) {
		chrootDir := t.TempDir()
		cfg := conf.MustNewConfigFrom(mapstr.M{
			"chroot": chrootDir,
		})
		_, inp, err := Configure(cfg, logp.NewNopLogger())
		require.Error(t, err, "Configure should fail when journalctl binary doesn't exist in chroot")
		assert.Contains(t, err.Error(), "cannot stat journalctl binary in chroot", "error should indicate journalctl binary is missing")
		jd, ok := inp.(*journald)
		if ok {
			assert.Equal(t, defaultJournalCtlPathChroot, jd.JournalctlPath, "should auto-switch to chroot default")
		}
	})

	t.Run("chroot with non-existent directory returns error", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(mapstr.M{
			"chroot":          "/nonexistent/directory",
			"journalctl_path": "/usr/bin/journalctl",
		})
		_, _, err := Configure(cfg, logp.NewNopLogger())
		require.Error(t, err, "Configure should fail when chroot directory doesn't exist")
		assert.Contains(t, err.Error(), "cannot stat chroot", "error should indicate chroot directory is missing")
	})

	t.Run("chroot with file instead of directory returns error", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "notadir")
		require.NoError(t, os.WriteFile(tmpFile, []byte("test"), 0o644), "should create temp file")
		cfg := conf.MustNewConfigFrom(mapstr.M{
			"chroot":          tmpFile,
			"journalctl_path": "/usr/bin/journalctl",
		})
		_, _, err := Configure(cfg, logp.NewNopLogger())
		require.Error(t, err, "Configure should fail when chroot path is a file")
		assert.Contains(t, err.Error(), "is not a directory", "error should indicate chroot path is not a directory")
	})
}

func decompress(t *testing.T, namegz string) string {
	return decompressGz(t, t.TempDir(), namegz)
}

func decompressAll(t *testing.T, globGz string) string {
	dir := t.TempDir()
	files, err := filepath.Glob(globGz)
	if err != nil {
		t.Fatalf("could not resolve glob: %s", err)
	}

	for _, f := range files {
		decompressGz(t, dir, f)
	}

	return dir
}

func decompressGz(t *testing.T, dir, namegz string) string {
	t.Helper()

	ingz, err := os.Open(namegz)
	require.NoError(t, err)
	defer ingz.Close()

	out := filepath.Join(dir, strings.TrimSuffix(filepath.Base(namegz), ".gz"))

	dst, err := os.Create(out)
	require.NoError(t, err)
	defer dst.Close()

	gr, err := gzip.NewReader(ingz)
	require.NoError(t, err)
	defer gr.Close()

	//nolint:gosec // this is used in tests
	_, err = io.Copy(dst, gr)
	require.NoError(t, err)

	return out
}
