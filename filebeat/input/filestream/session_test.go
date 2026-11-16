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
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest/observer"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/monitoring"
)

// --- Poll: close-on-state-change and resume/park decisions --------------

func TestHarvestSession_Poll(t *testing.T) {
	t.Run("unchanged file at EOF stays parked", func(t *testing.T) {
		s := newPollSession(t, closerConfig{}, "hello\n")
		require.Equal(t, loginp.PollPark, s.Poll())
	})

	t.Run("grown file resumes", func(t *testing.T) {
		s := newPollSession(t, closerConfig{}, "hello\n")
		s.readOffset = 0 // file has more data than we have read
		require.Equal(t, loginp.PollResume, s.Poll())
	})

	t.Run("unpublished trailing partial line still parks", func(t *testing.T) {
		s := newPollSession(t, closerConfig{}, "hello\npartial")
		s.state.Offset = int64(len("hello\n")) // only the terminated line published
		require.Equal(t, loginp.PollPark, s.Poll())
	})

	t.Run("done session closes", func(t *testing.T) {
		s := newPollSession(t, closerConfig{}, "hello\n")
		s.done = true
		require.Equal(t, loginp.PollClose, s.Poll())
	})

	t.Run("after_interval closes", func(t *testing.T) {
		closer := closerConfig{Reader: readerCloserConfig{AfterInterval: time.Nanosecond}}
		s := newPollSession(t, closer, "hello\n")
		s.openedAt = time.Now().Add(-time.Minute)
		require.Equal(t, loginp.PollClose, s.Poll())
	})

	t.Run("removed file closes", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("removing an open file is not supported on Windows; covered by the fakeFile stat subtests")
		}
		closer := closerConfig{OnStateChange: stateChangeCloserConfig{Removed: true}}
		s := newPollSession(t, closer, "hello\n")
		require.NoError(t, os.Remove(s.src.newPath))
		require.Equal(t, loginp.PollClose, s.Poll())
	})

	t.Run("renamed file closes", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("renaming an open file is not supported on Windows; covered by the fakeFile stat subtests")
		}
		closer := closerConfig{OnStateChange: stateChangeCloserConfig{Renamed: true}}
		s := newPollSession(t, closer, "hello\n")
		require.NoError(t, os.Rename(s.src.newPath, s.src.newPath+".moved"))
		require.Equal(t, loginp.PollClose, s.Poll())
	})

	t.Run("inactive file closes", func(t *testing.T) {
		closer := closerConfig{OnStateChange: stateChangeCloserConfig{Inactive: time.Minute}}
		s := newPollSession(t, closer, "hello\n")
		s.lastData = time.Now().Add(-time.Hour)
		require.Equal(t, loginp.PollClose, s.Poll())
	})

	t.Run("inactive file with delete enabled resumes for the worker to delete", func(t *testing.T) {
		closer := closerConfig{OnStateChange: stateChangeCloserConfig{Inactive: time.Minute}}
		s := newPollSession(t, closer, "hello\n")
		s.inp.deleterConfig.Enabled = true
		s.lastData = time.Now().Add(-time.Hour)
		require.Equal(t, loginp.PollResume, s.Poll())
		require.True(t, s.pendingDelete, "the session should be flagged for deletion")
	})

	t.Run("stat reporting not-exist closes when close.removed", func(t *testing.T) {
		s := newPollSession(t, closerConfig{OnStateChange: stateChangeCloserConfig{Removed: true}}, "x\n")
		s.file = &fakeFile{statErr: os.ErrNotExist}
		require.Equal(t, loginp.PollClose, s.Poll())
	})

	t.Run("unexpected stat error keeps the file parked", func(t *testing.T) {
		s := newPollSession(t, closerConfig{OnStateChange: stateChangeCloserConfig{Removed: true}}, "x\n")
		s.file = &fakeFile{statErr: errors.New("boom")}
		require.Equal(t, loginp.PollPark, s.Poll())
	})
}

// --- ReadSlice ----------------------------------------------------------

func TestHarvestSession_ReadSlice(t *testing.T) {
	t.Run("reads a file to EOF and reports done", func(t *testing.T) {
		s := newReadSession(t, closerConfig{Reader: readerCloserConfig{OnEOF: true}}, "a\nb\nc\n", 0)
		pub := &countingPublisher{}

		verdict, err := s.ReadSlice(backgroundCtx(), pub)
		require.NoError(t, err)
		require.Equal(t, loginp.SliceDone, verdict)
		require.Len(t, pub.events, 3, "all three lines should be published")
		require.Equal(t, int64(6), s.state.Offset, "offset should advance to EOF")
	})

	t.Run("a GZIP source updates the GZIP-specific metrics", func(t *testing.T) {
		s := newReadSession(t, closerConfig{Reader: readerCloserConfig{OnEOF: true}}, "a\nb\nc\n", 0)
		// Mark the source as GZIP. buildPipeline branches on the file's detected
		// compression, not on this flag, so a plain-text body still reads while
		// the GZIP metric counters are exercised.
		s.src.desc.GZIP = true
		pub := &countingPublisher{}

		verdict, err := s.ReadSlice(backgroundCtx(), pub)
		require.NoError(t, err)
		require.Equal(t, loginp.SliceDone, verdict)
		require.Len(t, pub.events, 3)

		require.Equal(t, uint64(3), s.metrics.MessagesGZIPRead.Get(), "gzip_messages_read_total")
		require.Equal(t, uint64(3), s.metrics.EventsGZIPProcessed.Get(), "gzip_events_processed_total")
		require.Positive(t, s.metrics.BytesGZIPProcessed.Get(), "gzip_bytes_processed_total")
	})

	t.Run("a trailing partial line is read but yields and then parks", func(t *testing.T) {
		content := "a\nb\npartial"
		closer := closerConfig{OnStateChange: stateChangeCloserConfig{Inactive: time.Minute}}
		s := newReadSession(t, closer, content, 0)
		pub := &countingPublisher{}

		verdict, err := s.ReadSlice(backgroundCtx(), pub)
		require.NoError(t, err)
		require.Equal(t, loginp.SliceYield, verdict)
		require.Len(t, pub.events, 2, "only the two terminated lines are published")
		require.Equal(t, int64(len("a\nb\n")), s.state.Offset, "published offset lags the partial line")
		require.Equal(t, int64(len(content)), s.readOffset, "readOffset reaches EOF including the partial line")

		require.Equal(t, loginp.PollPark, s.Poll(), "an unchanged file must park, not resume")
	})

	t.Run("reading filtered-out lines still counts as activity for close_inactive", func(t *testing.T) {
		s := newReadSession(t, closerConfig{Reader: readerCloserConfig{OnEOF: true}}, "drop1\ndrop2\n", 0)
		// Exclude every line, so nothing is published.
		s.inp.hasLineFilter = true
		s.inp.readerConfig.ExcludeLines = []match.Matcher{match.MustCompile("drop")}
		// Make lastData stale; reading the (dropped) lines must refresh it.
		s.lastData = time.Now().Add(-time.Hour)

		pub := &countingPublisher{}
		verdict, err := s.ReadSlice(backgroundCtx(), pub)
		require.NoError(t, err)
		require.Equal(t, loginp.SliceDone, verdict)
		require.Empty(t, pub.events, "all lines are excluded, nothing is published")
		require.WithinDuration(t, time.Now(), s.lastData, time.Minute,
			"reading filtered-out lines must refresh close_inactive activity")
	})

	t.Run("a done session reads nothing", func(t *testing.T) {
		s := newReadSession(t, closerConfig{}, "a\n", 0)
		s.done = true
		verdict, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
		require.NoError(t, err)
		require.Equal(t, loginp.SliceDone, verdict)
	})

	t.Run("a nil-file session reads nothing", func(t *testing.T) {
		s := newReadSession(t, closerConfig{}, "a\n", 0)
		s.file = nil
		verdict, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
		require.NoError(t, err)
		require.Equal(t, loginp.SliceDone, verdict)
	})

	t.Run("a cancelled context stops the read", func(t *testing.T) {
		s := newReadSession(t, closerConfig{}, "a\nb\n", 0)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		verdict, err := s.ReadSlice(input.Context{Logger: logp.NewNopLogger(), Cancelation: ctx}, &countingPublisher{})
		require.Equal(t, loginp.SliceDone, verdict)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("a publish error stops the read and is counted", func(t *testing.T) {
		s := newReadSession(t, closerConfig{Reader: readerCloserConfig{OnEOF: true}}, "a\nb\n", 0)
		pub := &countingPublisher{err: errors.New("publish failed")}
		verdict, err := s.ReadSlice(backgroundCtx(), pub)
		require.Equal(t, loginp.SliceDone, verdict)
		require.Error(t, err)
	})

	t.Run("a seek error stops the read", func(t *testing.T) {
		s := newReadSession(t, closerConfig{}, "a\n", 0)
		s.file = &fakeFile{seekErr: errors.New("seek failed")}
		verdict, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
		require.Equal(t, loginp.SliceDone, verdict)
		require.Error(t, err)
	})

	t.Run("a truncated file (offset past EOF) reports done", func(t *testing.T) {
		// Reading from an offset beyond the file size makes the reader detect
		// truncation and end the slice.
		s := newReadSession(t, closerConfig{}, "a\nb\nc\n", 100)
		verdict, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
		require.NoError(t, err)
		require.Equal(t, loginp.SliceDone, verdict)
	})

	t.Run("an unexpected read error ends the slice", func(t *testing.T) {
		s := newReadSession(t, closerConfig{}, "a\n", 0)
		s.file = &fakeFile{readFunc: func([]byte) (int, error) { return 0, errors.New("disk error") }}
		verdict, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
		require.NoError(t, err, "an unexpected read error ends the slice without surfacing the error")
		require.Equal(t, loginp.SliceDone, verdict)
	})

	t.Run("a slice budget yields before EOF even though more data remains", func(t *testing.T) {
		content := strings.Repeat("line\n", 100)
		s := newReadSession(t, closerConfig{}, content, 0)
		s.inp.sliceBudget = -time.Hour
		pub := &countingPublisher{}

		verdict, err := s.ReadSlice(backgroundCtx(), pub)
		require.NoError(t, err)
		require.Equal(t, loginp.SliceBudget, verdict,
			"a time-boxed slice with data remaining must report SliceBudget, not SliceYield or done")
		require.Empty(t, pub.events, "an already-past budget must cut the slice short before any line is read")
	})

	t.Run("no slice budget reads to EOF regardless of content size", func(t *testing.T) {
		// sliceBudget's zero value (the default, when nothing needs a bounded
		// slice) must not change behaviour: the slice still runs to EOF.
		content := strings.Repeat("line\n", 100)
		s := newReadSession(t, closerConfig{Reader: readerCloserConfig{OnEOF: true}}, content, 0)
		require.Zero(t, s.inp.sliceBudget)
		pub := &countingPublisher{}

		verdict, err := s.ReadSlice(backgroundCtx(), pub)
		require.NoError(t, err)
		require.Equal(t, loginp.SliceDone, verdict)
		require.Len(t, pub.events, 100)
	})
}

// --- Close --------------------------------------------------------------

func TestHarvestSession_Close(t *testing.T) {
	t.Run("closes the file once and is idempotent", func(t *testing.T) {
		s := newPollSession(t, closerConfig{}, "hello\n")
		require.NoError(t, s.Close())
		require.Nil(t, s.file, "the file handle should be released")
		require.NoError(t, s.Close(), "Close must be idempotent")
	})

	t.Run("a session without a file closes cleanly", func(t *testing.T) {
		s := newPollSession(t, closerConfig{}, "hello\n")
		s.file.Close()
		s.file = nil
		require.NoError(t, s.Close())
	})
}

func TestHarvestSession_TrackSource(t *testing.T) {
	assert.NotPanics(t, func() {
		(&harvestSession{}).TrackSource(fileSource{})
	})

	tbl := newFileStateTable()
	h := tbl.NewHandle()
	s := &harvestSession{harvesterState: h}

	s.TrackSource(notAFileSource{})
	assert.Empty(t, tbl.entries, "a non-file source must not list the session")

	s.TrackSource(fileSource{fileID: "fingerprint::new", desc: completeDesc("sum")})
	assert.Same(t, h, tbl.entries["fingerprint::new"], "the session must be listed under the source's identity")
	assert.Equal(t, "sum", h.FingerprintSum(), "the entry must take the source's descriptor")
}

// TestHarvestSession_TrackSourceCarriesDescriptor reproduces a migration that
// lands while OpenSession is still running. The scanner's UpdateDescriptor
// under the new key finds no entry, so TrackSource must deliver the completed
// fingerprint.
func TestHarvestSession_TrackSourceCarriesDescriptor(t *testing.T) {
	inp := testFilestream(t, closerConfig{Reader: readerCloserConfig{OnEOF: true}})
	inp.includeFileFingerprint = true
	inp.harvesterState = newFileStateTable()

	oldSrc := tempFileSource(t, "first line\nsecond line\n")
	oldSrc.fileID = "fingerprint::short"
	oldSrc.desc.Fingerprint = loginp.FingerprintID{Raw: "deadbeef"}
	newSrc := oldSrc
	newSrc.fileID = "fingerprint::full"
	newSrc.desc.Fingerprint = completeFP("full-sum")

	s := mustOpenSession(t, inp, oldSrc, "id", testMetrics(t))
	assert.Empty(t, inp.harvesterState.entries, "OpenSession must not list the file under its possibly stale identity")

	inp.harvesterState.UpdateDescriptor(newSrc.Name(), newSrc.desc)
	s.TrackSource(newSrc)

	assert.Equal(t, "full-sum", s.harvesterState.FingerprintSum(),
		"TrackSource must carry the completed fingerprint")

	pub := &countingPublisher{}
	verdict, err := s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err)
	assert.Equal(t, loginp.SliceDone, verdict, "close.on_eof must end the slice")
	assert.Len(t, pub.events, 2, "both lines must be published")
	for _, e := range pub.events {
		fp, err := e.Fields.GetValue("log.file.fingerprint")
		assert.NoError(t, err, "every event after the migration must carry log.file.fingerprint")
		assert.Equal(t, "full-sum", fp, "events must carry the completed fingerprint")
	}
}

// --- OpenSession & Test -------------------------------------------------

func TestFilestream_OpenSession_NotFileSource(t *testing.T) {
	inp := testFilestream(t, closerConfig{})
	_, err := inp.OpenSession(
		input.Context{Logger: logp.NewNopLogger(), Cancelation: context.Background()},
		notAFileSource{}, "id", loginp.NewCursorForTest("id", 0, 0), testMetrics(t))
	require.Error(t, err)
}

func TestFilestream_OpenSession_OpenError(t *testing.T) {
	inp := testFilestream(t, closerConfig{})
	src := fileSource{newPath: filepath.Join(t.TempDir(), "does-not-exist"), fileID: "id"}
	_, err := inp.OpenSession(backgroundCtx(), src, "id", loginp.NewCursorForTest("id", 0, 0), testMetrics(t))
	require.Error(t, err, "opening a missing file should fail")
}

// TestFilestream_OpenSession_HarvesterOffsetMetric asserts OpenSession
// registers a harvester ingestion progress offset (keyed by id, the source's
// registry key) for non-GZIP sources, ReadSlice keeps it in sync with the
// published offset, and Close cleans it up. GZIP sources are excluded because
// their progress can't be represented by a plain offset/size comparison.
func TestFilestream_OpenSession_HarvesterOffsetMetric(t *testing.T) {
	t.Run("registers, updates, and cleans up the offset", func(t *testing.T) {
		inp := testFilestream(t, closerConfig{Reader: readerCloserConfig{OnEOF: true}})
		metrics := testMetrics(t)
		src := tempFileSource(t, "a\nbc\n")

		s := mustOpenSession(t, inp, src, "harvester-id", metrics)
		require.NotNil(t, s.metricsOffset, "a non-GZIP source must register an offset")
		require.Zero(t, s.metricsOffset.Load())

		verdict, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
		require.NoError(t, err)
		require.Equal(t, loginp.SliceDone, verdict)
		require.EqualValues(t, len("a\nbc\n"), s.metricsOffset.Load(),
			"the registered offset must track the published offset")

		metrics.UpdateHarvesterBuckets([]loginp.HarvesterFile{{ID: "harvester-id", Size: int64(len("a\nbc\n"))}})
		require.EqualValues(t, 1, metrics.FilesIngestedPercent100.Get())

		require.NoError(t, s.Close())
		// A second Close must not double-cleanup (harvesterOffsets keys by
		// identity, not count, but this guards against a panic on re-entry).
		require.NoError(t, s.Close())
	})

	t.Run("GZIP sources are excluded", func(t *testing.T) {
		inp := testFilestream(t, closerConfig{})
		metrics := testMetrics(t)
		src := tempFileSource(t, "a\n")
		src.desc.GZIP = true

		s := mustOpenSession(t, inp, src, "gzip-id", metrics)
		assert.Nil(t, s.metricsOffset)
		assert.Nil(t, s.cleanupMetricsOffset)
	})
}

// TestLogFile_Read exercises the non-blocking reader's edge branches directly.
func TestLogFile_Read(t *testing.T) {
	t.Run("returns ErrClosed when the canceler is already cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r, err := newFileReader(logp.NewNopLogger(), ctx, &fakeFile{}, closerConfig{})
		require.NoError(t, err)
		_, err = r.Read(make([]byte, 8))
		require.ErrorIs(t, err, ErrClosed)
	})

	t.Run("delivers buffered bytes when EOF arrives with data on an active file", func(t *testing.T) {
		// An active (non close_on_eof) file whose read returns data together with
		// io.EOF must return the bytes with a nil error, not ErrWouldBlock.
		f := &fakeFile{
			readFunc: func(b []byte) (int, error) { return copy(b, "hi"), io.EOF },
			statFI:   fakeFileInfo{size: 100}, // size >= offset: not truncated
		}
		r, err := newFileReader(logp.NewNopLogger(), context.Background(), f, closerConfig{})
		require.NoError(t, err)
		n, err := r.Read(make([]byte, 8))
		require.NoError(t, err)
		require.Equal(t, 2, n)
	})
}

func TestFilestream_Test(t *testing.T) {
	inp := testFilestream(t, closerConfig{Reader: readerCloserConfig{OnEOF: true}})

	t.Run("not a file source errors", func(t *testing.T) {
		require.Error(t, inp.Test(notAFileSource{}, input.TestContext{Logger: logp.NewNopLogger(), Cancelation: context.Background()}))
	})

	t.Run("missing file errors", func(t *testing.T) {
		src := fileSource{newPath: filepath.Join(t.TempDir(), "does-not-exist"), fileID: "id"}
		require.Error(t, inp.Test(src, input.TestContext{Logger: logp.NewNopLogger(), Cancelation: context.Background()}))
	})

	t.Run("valid source passes", func(t *testing.T) {
		src := tempFileSource(t, "line\n")
		require.NoError(t, inp.Test(src, input.TestContext{Logger: logp.NewNopLogger(), Cancelation: context.Background()}))
	})
}

// --- Buffered parser state across slices (multiline) --------------------

// TestHarvestSession_MultilineAcrossSlices asserts the lines of one multiline
// event that arrive in different slices end up in a single event: the pipeline
// is retained while it buffers lines, nothing is published before the multiline
// timeout, and the event is flushed once the timeout elapses even though the
// file never changed again.
func TestHarvestSession_MultilineAcrossSlices(t *testing.T) {
	tests := map[string]struct {
		multiline map[string]any
		first     string
		more      []string
		want      string
	}{
		"pattern": {
			multiline: map[string]any{
				"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "300ms",
			},
			first: "[2015] hello\n",
			more:  []string{"  first\n", "  second\n"},
			want:  "[2015] hello\n  first\n  second",
		},
		"while_pattern": {
			multiline: map[string]any{"type": "while_pattern", "pattern": `^\s`, "timeout": "300ms"},
			first:     "  one\n",
			more:      []string{"  two\n", "  three\n"},
			want:      "  one\n  two\n  three",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, logs := newMultilineSession(t, closerConfig{}, tc.multiline, tc.first)
			pub := &countingPublisher{}
			total := int64(len(tc.first))

			verdict, err := s.ReadSlice(backgroundCtx(), pub)
			require.NoError(t, err, "first slice must not fail")
			require.Equal(t, loginp.SliceYield, verdict, "first slice must yield at EOF")
			require.Empty(t, pub.events, "an unfinished multiline event must not be published")
			require.NotNil(t, s.pipeline, "the pipeline must be retained while it buffers lines")
			firstDeadline := s.PendingDeadline()
			require.False(t, firstDeadline.IsZero(), "a buffered multiline event must have a flush deadline")
			require.Zero(t, s.Offset(), "the published offset must not advance past buffered lines")
			require.Equal(t, total, s.ReadOffset(), "the read offset must include buffered lines")
			require.Equal(t, loginp.PollPark, s.Poll(), "an unchanged file with a future deadline must stay parked")

			for _, line := range tc.more {
				appendToFile(t, s.src.newPath, line)
				total += int64(len(line))

				require.Equal(t, loginp.PollResume, s.Poll(), "a grown file must resume")
				verdict, err = s.ReadSlice(backgroundCtx(), pub)
				require.NoError(t, err, "continuation slice must not fail")
				require.Equal(t, loginp.SliceYield, verdict, "continuation slice must yield at EOF")
				require.Empty(t, pub.events, "no event may be published before the multiline timeout")
				require.NotNil(t, s.pipeline, "the pipeline must stay retained across continuation slices")
				require.Zero(t, s.Offset(), "the published offset must not advance past buffered lines")
				require.Equal(t, total, s.ReadOffset(), "the read offset must include buffered lines")
			}
			require.True(t, s.PendingDeadline().After(firstDeadline),
				"each continuation line must postpone the flush deadline")
			require.Equal(t, loginp.PollPark, s.Poll(), "before the deadline the session must stay parked")

			lastDataBeforeFlush := s.lastData
			time.Sleep(time.Until(s.PendingDeadline()) + 20*time.Millisecond)

			require.Equal(t, loginp.PollResume, s.Poll(),
				"an elapsed deadline must resume the session even though the file did not change")
			verdict, err = s.ReadSlice(backgroundCtx(), pub)
			require.NoError(t, err, "flush slice must not fail")
			require.Equal(t, loginp.SliceYield, verdict, "flush slice must yield at EOF")
			require.Len(t, pub.events, 1, "the buffered lines must be published as a single event")
			require.Equal(t, tc.want, messageOf(t, pub.events[0]), "unexpected multiline event content")
			require.Equal(t, total, s.Offset(), "the published offset must reach the end of the event")
			require.Nil(t, s.pipeline, "the pipeline must be released once nothing is buffered")
			require.True(t, s.PendingDeadline().IsZero(), "no deadline may remain after the flush")
			require.Equal(t, 1, logs.FilterMessageSnippet("Multiline event flushed because timeout reached.").Len(),
				"the timeout flush must be logged")
			require.Equal(t, lastDataBeforeFlush, s.lastData,
				"flushing buffered lines reads no data and must not count as file activity")
			require.Equal(t, loginp.PollPark, s.Poll(), "after the flush the idle file must park")
		})
	}
}

// TestHarvestSession_MultilineCompletedByNextEvent asserts a retained pipeline
// is not re-seeked: lines buffered by an earlier slice are completed by a later
// one without being duplicated or lost.
func TestHarvestSession_MultilineCompletedByNextEvent(t *testing.T) {
	ml := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "1h"}
	first := "[1] a\n"
	s, _ := newMultilineSession(t, closerConfig{}, ml, first)
	pub := &countingPublisher{}

	verdict, err := s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "first slice must not fail")
	require.Equal(t, loginp.SliceYield, verdict, "first slice must yield")
	require.Empty(t, pub.events, "the first event is still incomplete")

	more := "  b\n[2] c\n"
	appendToFile(t, s.src.newPath, more)
	require.Equal(t, loginp.PollResume, s.Poll(), "a grown file must resume")

	verdict, err = s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "second slice must not fail")
	require.Equal(t, loginp.SliceYield, verdict, "second slice must yield")
	require.Len(t, pub.events, 1, "the next event's first line completes the previous one")
	require.Equal(t, "[1] a\n  b", messageOf(t, pub.events[0]), "unexpected completed event")
	require.Equal(t, int64(len("[1] a\n  b\n")), s.Offset(), "only the completed event may be published")
	require.Equal(t, int64(len(first+more)), s.ReadOffset(), "the read offset must cover the buffered line too")
	require.NotNil(t, s.pipeline, "the new event's line is buffered, so the pipeline must stay retained")
}

// TestHarvestSession_MultilineCount asserts count mode, which has no timeout,
// keeps buffering across slices until enough lines arrive, never asks to be
// polled for a deadline, and publishes a single event once complete.
func TestHarvestSession_MultilineCount(t *testing.T) {
	s, _ := newMultilineSession(t, closerConfig{}, map[string]any{"type": "count", "count_lines": 3}, "a\n")
	pub := &countingPublisher{}

	verdict, err := s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "first slice must not fail")
	require.Equal(t, loginp.SliceYield, verdict, "first slice must yield")
	require.Empty(t, pub.events, "1 of 3 lines must not be published")
	require.NotNil(t, s.pipeline, "the pipeline must be retained while lines are buffered")
	require.True(t, s.PendingDeadline().IsZero(), "count mode has no flush deadline")
	require.Equal(t, loginp.PollPark, s.Poll(), "count mode must park without a deadline")

	appendToFile(t, s.src.newPath, "b\n")
	require.Equal(t, loginp.PollResume, s.Poll(), "a grown file must resume")
	verdict, err = s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "second slice must not fail")
	require.Equal(t, loginp.SliceYield, verdict, "second slice must yield")
	require.Empty(t, pub.events, "2 of 3 lines must not be published")
	require.NotNil(t, s.pipeline, "the pipeline must stay retained")
	require.Equal(t, loginp.PollPark, s.Poll(), "count mode must keep parking")

	appendToFile(t, s.src.newPath, "c\n")
	require.Equal(t, loginp.PollResume, s.Poll(), "a grown file must resume")
	verdict, err = s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "third slice must not fail")
	require.Equal(t, loginp.SliceYield, verdict, "third slice must yield")
	require.Len(t, pub.events, 1, "3 lines must be published as one event")
	require.Equal(t, "a\nb\nc", messageOf(t, pub.events[0]), "unexpected count event content")
	require.Equal(t, int64(len("a\nb\nc\n")), s.Offset(), "the published offset must cover all three lines")
	require.Nil(t, s.pipeline, "the pipeline must be released once the event is published")
}

// TestHarvestSession_PendingFlushedOnClose asserts a close condition met while
// the pipeline buffers lines does not drop them: Poll hands the session to a
// worker, which publishes the buffered event and then finishes the session.
func TestHarvestSession_PendingFlushedOnClose(t *testing.T) {
	pattern := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "1h"}
	count := map[string]any{"type": "count", "count_lines": 5}

	tests := map[string]struct {
		closer  closerConfig
		ml      map[string]any
		want    string
		trigger func(t *testing.T, s *harvestSession)
		skipWin bool
	}{
		"after_interval": {
			closer: closerConfig{Reader: readerCloserConfig{AfterInterval: time.Hour}},
			ml:     pattern,
			want:   "[1] a\n  b",
			trigger: func(_ *testing.T, s *harvestSession) {
				s.openedAt = time.Now().Add(-2 * time.Hour)
			},
		},
		"inactive": {
			closer: closerConfig{OnStateChange: stateChangeCloserConfig{Inactive: time.Minute}},
			ml:     pattern,
			want:   "[1] a\n  b",
			trigger: func(_ *testing.T, s *harvestSession) {
				s.lastData = time.Now().Add(-time.Hour)
			},
		},
		"inactive flushes a partial count": {
			closer: closerConfig{OnStateChange: stateChangeCloserConfig{Inactive: time.Minute}},
			ml:     count,
			want:   "[1] a\n  b",
			trigger: func(_ *testing.T, s *harvestSession) {
				s.lastData = time.Now().Add(-time.Hour)
			},
		},
		"removed": {
			closer:  closerConfig{OnStateChange: stateChangeCloserConfig{Removed: true}},
			ml:      pattern,
			want:    "[1] a\n  b",
			skipWin: true,
			trigger: func(t *testing.T, s *harvestSession) {
				require.NoError(t, os.Remove(s.src.newPath), "removing the file must succeed")
			},
		},
		"renamed": {
			closer:  closerConfig{OnStateChange: stateChangeCloserConfig{Renamed: true}},
			ml:      pattern,
			want:    "[1] a\n  b",
			skipWin: true,
			trigger: func(t *testing.T, s *harvestSession) {
				require.NoError(t, os.Rename(s.src.newPath, s.src.newPath+".moved"), "renaming the file must succeed")
			},
		},
	}

	content := "[1] a\n  b\n"
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.skipWin && runtime.GOOS == "windows" {
				t.Skip("removing/renaming an open file is not supported on Windows")
			}
			s, _ := newMultilineSession(t, tc.closer, tc.ml, content)
			pub := &countingPublisher{}

			verdict, err := s.ReadSlice(backgroundCtx(), pub)
			require.NoError(t, err, "buffering slice must not fail")
			require.Equal(t, loginp.SliceYield, verdict, "buffering slice must yield")
			require.Empty(t, pub.events, "the event is incomplete, nothing may be published yet")
			require.NotNil(t, s.pipeline, "the pipeline must be retained")

			tc.trigger(t, s)

			require.Equal(t, loginp.PollResume, s.Poll(),
				"a close condition with buffered lines must resume so a worker can publish them")
			require.True(t, s.pendingClose, "the session must be flagged for closing")

			verdict, err = s.ReadSlice(backgroundCtx(), pub)
			require.NoError(t, err, "closing slice must not fail")
			require.Equal(t, loginp.SliceDone, verdict, "the session must finish after flushing")
			require.Len(t, pub.events, 1, "the buffered lines must be published before closing")
			require.Equal(t, tc.want, messageOf(t, pub.events[0]), "unexpected flushed event content")
			require.Equal(t, int64(len(content)), s.Offset(), "the published offset must cover the flushed lines")
			require.Nil(t, s.pipeline, "the pipeline must be released")
			require.False(t, s.pendingClose, "the close flag must be consumed")
		})
	}
}

// TestHarvestSession_CloseWithoutPendingClosesImmediately asserts the worker
// hand-off is only used when lines are buffered: an idle session is closed by
// Poll directly.
func TestHarvestSession_CloseWithoutPendingClosesImmediately(t *testing.T) {
	ml := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "1h"}
	closer := closerConfig{Reader: readerCloserConfig{AfterInterval: time.Hour}}
	// An empty file leaves nothing buffered.
	s, _ := newMultilineSession(t, closer, ml, "")

	verdict, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
	require.NoError(t, err, "slice must not fail")
	require.Equal(t, loginp.SliceYield, verdict, "slice must yield at EOF")
	require.Nil(t, s.pipeline, "an idle file with multiline configured must not retain a pipeline")
	require.True(t, s.PendingDeadline().IsZero(), "an idle file must have no flush deadline")

	s.openedAt = time.Now().Add(-2 * time.Hour)
	require.Equal(t, loginp.PollClose, s.Poll(), "nothing buffered: Poll must close without a worker")
	require.False(t, s.pendingClose, "the close flag must stay unset")
}

// TestHarvestSession_PendingFlushedBeforeDelete asserts that on close_inactive
// with delete.enabled, buffered lines are published before the file is removed,
// so no data is lost with the file.
func TestHarvestSession_PendingFlushedBeforeDelete(t *testing.T) {
	ml := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "1h"}
	closer := closerConfig{OnStateChange: stateChangeCloserConfig{Inactive: time.Minute}}
	content := "[1] a\n  b\n"
	s, _ := newMultilineSession(t, closer, ml, content)
	pub := &countingPublisher{}

	var removed []string
	var publishedAtRemove int
	s.cursor = loginp.NewCursorForTest("id", 0, 0)
	s.inp.deleterConfig.Enabled = true
	s.inp.waitGracePeriodFn = func(input.Context, *logp.Logger, loginp.Cursor, string, time.Duration, time.Duration, func(string) (os.FileInfo, error)) (bool, error) {
		return true, nil
	}
	s.inp.removeFn = func(path string) error {
		pub.mu.Lock()
		publishedAtRemove = len(pub.events)
		pub.mu.Unlock()
		removed = append(removed, path)
		return nil
	}

	verdict, err := s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "buffering slice must not fail")
	require.Equal(t, loginp.SliceYield, verdict, "buffering slice must yield")
	require.NotNil(t, s.pipeline, "the pipeline must be retained")

	s.lastData = time.Now().Add(-time.Hour)
	require.Equal(t, loginp.PollResume, s.Poll(), "inactive + delete must resume for a worker")
	require.True(t, s.pendingDelete, "the session must be flagged for deletion")

	verdict, err = s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "delete slice must not fail")
	require.Equal(t, loginp.SliceDone, verdict, "the session must finish")
	require.Equal(t, 1, publishedAtRemove, "the buffered event must be published before the file is removed")
	require.Equal(t, []string{s.src.newPath}, removed, "the file must be removed exactly once")
	require.Equal(t, int64(len(content)), s.Offset(), "the whole file must have been published")
	require.Nil(t, s.pipeline, "the pipeline must be released")
}

// TestHarvestSession_SliceBudgetKeepsPending asserts a slice cut short by its
// time budget keeps a pipeline that buffers lines, but not one that buffers
// nothing.
func TestHarvestSession_SliceBudgetKeepsPending(t *testing.T) {
	ml := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "1h"}
	s, _ := newMultilineSession(t, closerConfig{}, ml, "[1] a\n")
	pub := &countingPublisher{}

	verdict, err := s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "buffering slice must not fail")
	require.Equal(t, loginp.SliceYield, verdict, "buffering slice must yield")
	require.NotNil(t, s.pipeline, "the pipeline must be retained")

	s.inp.sliceBudget = -time.Hour // already elapsed: yields before reading
	verdict, err = s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "budget slice must not fail")
	require.Equal(t, loginp.SliceBudget, verdict, "an elapsed budget must report SliceBudget")
	require.NotNil(t, s.pipeline, "buffered lines must survive a budget yield")
	require.Empty(t, pub.events, "nothing may be published")

	s.inp.sliceBudget = 0
	appendToFile(t, s.src.newPath, "  b\n")
	verdict, err = s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "continuation slice must not fail")
	require.Equal(t, loginp.SliceYield, verdict, "continuation slice must yield")
	require.NotNil(t, s.pipeline, "the pipeline must still hold the combined lines")
	require.EqualValues(t, len("[1] a\n  b\n"), s.ReadOffset(), "the continuation line must be read once")

	// Without buffered state a budget yield releases the pipeline, as before.
	s2 := newReadSession(t, closerConfig{}, strings.Repeat("line\n", 10), 0)
	s2.inp.sliceBudget = -time.Hour
	verdict, err = s2.ReadSlice(backgroundCtx(), &countingPublisher{})
	require.NoError(t, err, "budget slice must not fail")
	require.Equal(t, loginp.SliceBudget, verdict, "an elapsed budget must report SliceBudget")
	require.Nil(t, s2.pipeline, "without buffered state no pipeline may be retained")
}

// TestHarvestSession_FlushReleasesPending asserts Flush (the read_until_eof
// drain) publishes buffered lines and releases the pipeline, and is a no-op
// afterwards or when the context was cancelled.
func TestHarvestSession_FlushReleasesPending(t *testing.T) {
	ml := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "1h"}
	content := "[1] a\n  b\n"

	t.Run("publishes buffered lines", func(t *testing.T) {
		s, _ := newMultilineSession(t, closerConfig{}, ml, content)
		pub := &countingPublisher{}
		_, err := s.ReadSlice(backgroundCtx(), pub)
		require.NoError(t, err, "buffering slice must not fail")
		require.Empty(t, pub.events, "the event is incomplete")

		require.NoError(t, s.Flush(backgroundCtx(), pub), "Flush must not fail")
		require.Len(t, pub.events, 1, "Flush must publish the buffered event")
		require.Equal(t, "[1] a\n  b", messageOf(t, pub.events[0]), "unexpected flushed event content")
		require.Equal(t, int64(len(content)), s.Offset(), "the published offset must cover the flushed lines")
		require.Nil(t, s.pipeline, "Flush must release the pipeline")

		require.NoError(t, s.Flush(backgroundCtx(), pub), "a second Flush must be a no-op")
		require.Len(t, pub.events, 1, "a second Flush must not publish again")
	})

	t.Run("cancelled context publishes nothing and releases the pipeline", func(t *testing.T) {
		s, _ := newMultilineSession(t, closerConfig{}, ml, content)
		pub := &countingPublisher{}
		_, err := s.ReadSlice(backgroundCtx(), pub)
		require.NoError(t, err, "buffering slice must not fail")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err = s.Flush(input.Context{Logger: logp.NewNopLogger(), Cancelation: ctx}, pub)
		require.ErrorIs(t, err, context.Canceled, "Flush must report the cancellation")
		require.Empty(t, pub.events, "nothing may be published after cancellation")
		require.Nil(t, s.pipeline, "the pipeline must be released")
	})

	t.Run("nothing buffered is a no-op", func(t *testing.T) {
		s := newReadSession(t, closerConfig{}, "a\n", 0)
		pub := &countingPublisher{}
		require.NoError(t, s.Flush(backgroundCtx(), pub), "Flush must not fail")
		require.Empty(t, pub.events, "nothing may be published")
	})
}

// TestHarvestSession_CloseReleasesPipeline asserts closing a parked session
// that retains a pipeline closes the pipeline too.
func TestHarvestSession_CloseReleasesPipeline(t *testing.T) {
	ml := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "1h"}
	s, _ := newMultilineSession(t, closerConfig{}, ml, "[1] a\n")
	_, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
	require.NoError(t, err, "buffering slice must not fail")
	require.NotNil(t, s.pipeline, "the pipeline must be retained")
	require.NotNil(t, s.logReader, "the file reader must be retained")

	pipeline, logReader := s.pipeline, s.logReader

	require.NoError(t, s.Close(), "Close must not fail")
	require.Nil(t, s.pipeline, "Close must release the retained pipeline")
	require.Nil(t, s.logReader, "Close must release the retained file reader")
	require.Nil(t, s.file, "Close must release the file")

	pending, _ := reader.PendingDeadline(pipeline)
	require.False(t, pending, "Close must close the top of the retained pipeline")
	_, err = logReader.Read(make([]byte, 1))
	require.ErrorIs(t, err, ErrClosed, "Close must propagate to the bottom of the retained pipeline")
}

// TestHarvestSession_PendingAndTruncation asserts a file truncated while lines
// are buffered still publishes them and ends the session.
func TestHarvestSession_PendingAndTruncation(t *testing.T) {
	ml := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "1h"}
	s, _ := newMultilineSession(t, closerConfig{}, ml, "[1] a\n  b\n")
	pub := &countingPublisher{}

	_, err := s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "buffering slice must not fail")
	require.NotNil(t, s.pipeline, "the pipeline must be retained")

	require.NoError(t, os.Truncate(s.src.newPath, 3), "truncating the file must succeed")
	require.Equal(t, loginp.PollResume, s.Poll(), "a size change must resume")

	verdict, err := s.ReadSlice(backgroundCtx(), pub)
	require.NoError(t, err, "truncation must end the slice without an error")
	require.Equal(t, loginp.SliceDone, verdict, "truncation must finish the session")
	require.Len(t, pub.events, 1, "buffered lines must be published before the session ends")
	require.Nil(t, s.pipeline, "the pipeline must be released")
}

// TestHarvestSession_PollFlushesTimedOutPendingOnStatError asserts an elapsed
// deadline resumes the session even if stat keeps failing, so the buffered
// lines are flushed and the waker does not spin on the elapsed deadline.
func TestHarvestSession_PollFlushesTimedOutPendingOnStatError(t *testing.T) {
	ml := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "100ms"}
	s, _ := newMultilineSession(t, closerConfig{OnStateChange: stateChangeCloserConfig{Removed: true}}, ml, "[1] a\n")
	_, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
	require.NoError(t, err, "buffering slice must not fail")

	s.file = &fakeFile{statErr: errors.New("boom")}
	require.Equal(t, loginp.PollPark, s.Poll(), "before the deadline a stat error keeps the session parked")

	time.Sleep(time.Until(s.PendingDeadline()) + 20*time.Millisecond)
	require.Equal(t, loginp.PollResume, s.Poll(), "after the deadline a stat error must still resume to flush")
}

// TestHarvestSession_ReadActivityRefreshesLastData asserts bytes buffered by a
// parser count as file activity for close_inactive, even though no message is
// returned by the pipeline.
func TestHarvestSession_ReadActivityRefreshesLastData(t *testing.T) {
	ml := map[string]any{"type": "pattern", "pattern": `^\[`, "negate": true, "match": "after", "timeout": "1h"}
	s, _ := newMultilineSession(t, closerConfig{}, ml, "[1] a\n")
	s.lastData = time.Now().Add(-time.Hour)

	_, err := s.ReadSlice(backgroundCtx(), &countingPublisher{})
	require.NoError(t, err, "slice must not fail")
	require.WithinDuration(t, time.Now(), s.lastData, time.Minute,
		"reading bytes into a parser's buffer must refresh close_inactive activity")

	// A slice that reads nothing new must not refresh it.
	s.lastData = time.Now().Add(-time.Hour)
	_, err = s.ReadSlice(backgroundCtx(), &countingPublisher{})
	require.NoError(t, err, "slice must not fail")
	require.Greater(t, time.Since(s.lastData), 30*time.Minute,
		"a slice that read nothing new must not refresh close_inactive activity")
}

// --- scaffolding --------------------------------------------------------

// newMultilineSession is newReadSession with a multiline parser configured and
// an observable logger, to assert on the parser's log output.
func newMultilineSession(
	t *testing.T,
	closer closerConfig,
	multiline map[string]any,
	content string,
) (*harvestSession, *observer.ObservedLogs) {
	t.Helper()
	s := newReadSession(t, closer, content, 0)

	cfg, err := conf.NewConfigFrom(map[string]any{
		"parsers": []map[string]any{{"multiline": multiline}},
	})
	require.NoError(t, err, "building the parsers config must succeed")
	var pc parser.Config
	require.NoError(t, cfg.Unpack(&pc), "unpacking the parsers config must succeed")
	s.inp.parsers = pc

	log, logs := logptest.NewTestingLoggerWithObserver(t, "")
	s.log = log
	return s, logs
}

func messageOf(t *testing.T, e beat.Event) string {
	t.Helper()
	v, err := e.Fields.GetValue("message")
	require.NoError(t, err, "event must have a message field")
	msg, ok := v.(string)
	require.Truef(t, ok, "message must be a string, got %T", v)
	return msg
}

func testFilestream(t *testing.T, closer closerConfig) *filestream {
	t.Helper()
	encFactory, ok := encoding.FindEncoding("")
	require.True(t, ok, "the default encoding must be available")
	return &filestream{
		compression:     CompressionNone,
		encodingFactory: encFactory,
		readerConfig:    readerConfig{BufferSize: 1024, MaxBytes: 1 << 20, LineTerminator: readfile.AutoLineTerminator},
		closerConfig:    closer,
	}
}

func testMetrics(t *testing.T) *loginp.Metrics {
	t.Helper()
	return loginp.NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())
}

func backgroundCtx() input.Context {
	return input.Context{Logger: logp.NewNopLogger(), Cancelation: context.Background()}
}

func mustOpenSession(t *testing.T, inp *filestream, src fileSource, id string, metrics *loginp.Metrics) *harvestSession {
	t.Helper()
	sess, err := inp.OpenSession(backgroundCtx(), src, id, loginp.NewCursorForTest(id, 0, 0), metrics)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sess.Close() })
	s, ok := sess.(*harvestSession)
	require.Truef(t, ok, "Got %T instead of *harvestSession", sess)
	return s
}

func tempFileSource(t *testing.T, content string) fileSource {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.log")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	fi, err := os.Stat(path)
	require.NoError(t, err)
	return fileSource{newPath: path, fileID: "id", desc: loginp.FileDescriptor{Info: file.ExtendFileInfo(fi)}}
}

// newPollSession builds a harvestSession over a real file with its offset at the
// file size (caught up), enough to drive Poll/Close.
func newPollSession(t *testing.T, closer closerConfig, content string) *harvestSession {
	t.Helper()
	inp := testFilestream(t, closer)
	src := tempFileSource(t, content)
	rawFile, err := file.ReadOpen(src.newPath)
	require.NoError(t, err)
	t.Cleanup(func() { rawFile.Close() })
	f, err := inp.newFile(rawFile)
	require.NoError(t, err)
	return &harvestSession{
		inp:        inp,
		log:        logp.NewNopLogger(),
		src:        src,
		file:       f,
		metrics:    testMetrics(t),
		state:      state{Offset: int64(len(content))},
		readOffset: int64(len(content)), // caught up to EOF
		openedAt:   time.Now(),
		lastData:   time.Now(),
	}
}

// newReadSession builds a harvestSession with a detected encoding so ReadSlice
// can build its reader pipeline.
func newReadSession(t *testing.T, closer closerConfig, content string, offset int64) *harvestSession {
	t.Helper()
	s := newPollSession(t, closer, content)
	s.state.Offset = offset
	s.readOffset = offset // nothing read yet, as after OpenSession
	enc, err := s.inp.encodingFactory(s.file)
	require.NoError(t, err)
	s.enc = enc
	return s
}

// countingPublisher records published events; err, when set, fails Publish.
type countingPublisher struct {
	mu     sync.Mutex
	events []beat.Event
	err    error
}

func (p *countingPublisher) Publish(e beat.Event, _ any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, e)
	return nil
}

// notAFileSource is a loginp.Source that is not a fileSource.
type notAFileSource struct{}

func (notAFileSource) Name() string    { return "not-a-file" }
func (notAFileSource) LogPath() string { return "not-a-file" }

// fakeFile is a File whose Stat/Seek/Read can be programmed, to exercise error
// and edge branches that a real file does not reach.
type fakeFile struct {
	statErr  error
	statFI   fs.FileInfo
	seekErr  error
	readFunc func([]byte) (int, error)
}

func (f *fakeFile) Stat() (fs.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return f.statFI, nil
}
func (f *fakeFile) Read(b []byte) (int, error) {
	if f.readFunc != nil {
		return f.readFunc(b)
	}
	return 0, io.EOF
}
func (f *fakeFile) Seek(int64, int) (int64, error) {
	if f.seekErr != nil {
		return 0, f.seekErr
	}
	return 0, nil
}
func (f *fakeFile) Close() error     { return nil }
func (f *fakeFile) Name() string     { return "fake" }
func (f *fakeFile) OSFile() *os.File { return nil }
func (f *fakeFile) IsGZIP() bool     { return false }

// fakeFile must satisfy the File interface.
var _ File = (*fakeFile)(nil)

// fakeFileInfo is a minimal fs.FileInfo reporting a fixed size.
type fakeFileInfo struct{ size int64 }

func (fakeFileInfo) Name() string       { return "fake" }
func (f fakeFileInfo) Size() int64      { return f.size }
func (fakeFileInfo) Mode() os.FileMode  { return 0 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return false }
func (fakeFileInfo) Sys() any           { return nil }
