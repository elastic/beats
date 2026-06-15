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

//nolint:errcheck // It's a test file
package filestream

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// --- Poll: close-on-state-change and resume/park decisions --------------

func TestHarvestSession_Poll(t *testing.T) {
	t.Run("unchanged file at EOF stays parked", func(t *testing.T) {
		s := newPollSession(t, closerConfig{}, "hello\n")
		require.Equal(t, loginp.PollPark, s.Poll())
	})

	t.Run("grown file resumes", func(t *testing.T) {
		s := newPollSession(t, closerConfig{}, "hello\n")
		s.state.Offset = 0 // file has more data than we have read
		require.Equal(t, loginp.PollResume, s.Poll())
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
		closer := closerConfig{OnStateChange: stateChangeCloserConfig{Removed: true}}
		s := newPollSession(t, closer, "hello\n")
		require.NoError(t, os.Remove(s.src.newPath))
		require.Equal(t, loginp.PollClose, s.Poll())
	})

	t.Run("renamed file closes", func(t *testing.T) {
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

// --- OpenSession & Test -------------------------------------------------

func TestFilestream_OpenSession_NotFileSource(t *testing.T) {
	inp := testFilestream(t, closerConfig{})
	_, err := inp.OpenSession(
		input.Context{Logger: logp.NewNopLogger(), Cancelation: context.Background()},
		notAFileSource{}, loginp.NewCursorForTest("id", 0, 0), testMetrics(t))
	require.Error(t, err)
}

func TestFilestream_OpenSession_OpenError(t *testing.T) {
	inp := testFilestream(t, closerConfig{})
	src := fileSource{newPath: filepath.Join(t.TempDir(), "does-not-exist"), fileID: "id"}
	_, err := inp.OpenSession(backgroundCtx(), src, loginp.NewCursorForTest("id", 0, 0), testMetrics(t))
	require.Error(t, err, "opening a missing file should fail")
}

// TestLogFile_Read exercises the non-blocking reader's edge branches directly.
func TestLogFile_Read(t *testing.T) {
	t.Run("returns ErrClosed when the canceler is already cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r, err := newFileReader(logp.NewNopLogger(), ctx, &fakeFile{}, readerConfig{}, closerConfig{})
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
		r, err := newFileReader(logp.NewNopLogger(), context.Background(), f, readerConfig{}, closerConfig{})
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
		path := writeTempFile(t, "line\n")
		fi, err := os.Stat(path)
		require.NoError(t, err)
		src := fileSource{newPath: path, fileID: "id", desc: loginp.FileDescriptor{Info: file.ExtendFileInfo(fi)}}
		require.NoError(t, inp.Test(src, input.TestContext{Logger: logp.NewNopLogger(), Cancelation: context.Background()}))
	})
}

// --- scaffolding --------------------------------------------------------

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

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.log")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// newPollSession builds a harvestSession over a real file with its offset at the
// file size (caught up), enough to drive Poll/Close.
func newPollSession(t *testing.T, closer closerConfig, content string) *harvestSession {
	t.Helper()
	inp := testFilestream(t, closer)
	path := writeTempFile(t, content)
	rawFile, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { rawFile.Close() })
	f, err := inp.newFile(rawFile)
	require.NoError(t, err)
	fi, err := os.Stat(path)
	require.NoError(t, err)
	return &harvestSession{
		inp: inp,
		log: logp.NewNopLogger(),
		src: fileSource{
			newPath: path,
			fileID:  "id",
			desc:    loginp.FileDescriptor{Info: file.ExtendFileInfo(fi)},
		},
		file:     f,
		metrics:  testMetrics(t),
		state:    state{Offset: int64(len(content))},
		openedAt: time.Now(),
		lastData: time.Now(),
	}
}

// newReadSession builds a harvestSession with a detected encoding so ReadSlice
// can build its reader pipeline.
func newReadSession(t *testing.T, closer closerConfig, content string, offset int64) *harvestSession {
	t.Helper()
	s := newPollSession(t, closer, content)
	s.state.Offset = offset
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

func (p *countingPublisher) Publish(e beat.Event, _ interface{}) error {
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

func (notAFileSource) Name() string { return "not-a-file" }

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
