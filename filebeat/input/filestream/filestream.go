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
	"errors"
	"io"
	"os"
	"sync"

	"github.com/elastic/go-concert/ctxtool"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	ErrClosed       = errors.New("reader closed")
	ErrFileTruncate = errors.New("detected file being truncated")

	// ErrWouldBlock is returned by Read when the file has no data available
	// right now. Reading is non-blocking: the harvester runner's waker decides
	// when to retry, so the read path never waits on a backoff.
	ErrWouldBlock = errors.New("no data available, would block")
)

// logFile is a non-blocking reader over a single open file.
//
// The file handle is owned by the harvester session, so Close does NOT close
// it, and the close-on-state-change conditions (inactive/removed/renamed and
// close-after-interval) are evaluated by the scheduler's waker rather than by a
// per-file monitor goroutine. logFile therefore only reports end of data: io.EOF
// when close_on_eof (or a GZIP file) reaches the end, ErrFileTruncate when the
// file shrank, or ErrWouldBlock when an active file has nothing to read yet.
type logFile struct {
	file      File
	log       *logp.Logger
	readerCtx ctxtool.CancelContext

	closeOnEOF bool

	offsetMutx sync.Mutex
	offset     int64
}

// newFileReader creates a new log instance to read log sources
func newFileReader(
	log *logp.Logger,
	canceler input.Canceler,
	f File,
	closerConfig closerConfig,
) (*logFile, error) {
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	return &logFile{
		file:       f,
		log:        log,
		closeOnEOF: closerConfig.Reader.OnEOF,
		offset:     offset,
		readerCtx:  ctxtool.WithCancelContext(ctxtool.FromCanceller(canceler)),
	}, nil
}

// Read reads from the file into buf without blocking. It returns:
//   - the bytes read with a nil error when data was available;
//   - io.EOF when close_on_eof (or a GZIP file) reaches the end;
//   - ErrFileTruncate when the file shrank;
//   - ErrWouldBlock when an active file has no data right now;
//   - ErrClosed when the reader's context was cancelled.
func (f *logFile) Read(buf []byte) (int, error) {
	if f.readerCtx.Err() != nil {
		return 0, ErrClosed
	}

	totalN := 0

	n, err := f.file.Read(buf)
	if n > 0 {
		f.updateOffset(n)
	}
	totalN += n

	// Read from source completed without error
	// Either end reached or buffer full
	if err == nil {
		return totalN, nil
	}

	// Move buffer forward for next read
	buf = buf[n:]

	// Checks if an error happened or buffer is full
	// If buffer is full, cannot continue reading.
	// Can happen if n == bufferSize + io.EOF error
	err = f.errorChecks(err)
	if err != nil || len(buf) == 0 {
		return totalN, err
	}

	// Active file at EOF: deliver whatever was read this call, otherwise
	// signal that no data is available right now so the worker can yield.
	//
	// ErrWouldBlock must mean "zero progress this read": if we returned it
	// together with bytes, LineReader.advance buffers those bytes but then
	// returns early on the error, skipping the newline scan that may have
	// just completed a line. Returning nil whenever totalN > 0 lets the
	// pipeline process the data and reserves ErrWouldBlock for a truly empty
	// read. (totalN is never negative: it only accumulates io.Reader counts.)
	if totalN > 0 {
		return totalN, nil
	}
	return 0, ErrWouldBlock
}

func isSameFile(path string, info os.FileInfo) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return os.SameFile(fileInfo, info)
}

// errorChecks determines the cause for EOF errors, and how the EOF event should be handled
// based on the config options.
func (f *logFile) errorChecks(err error) error {
	if !errors.Is(err, io.EOF) {
		f.log.Errorf("Unexpected state reading from %s; error: %s",
			f.file.Name(), err)

		// gzip.ErrChecksum happens after all data is read from a GZIP file, and
		// it's recoverable, nothing else to do. Thus, we return EOF.
		if errors.Is(err, gzip.ErrChecksum) {
			return io.EOF
		}

		return err
	}

	return f.handleEOF()
}

func (f *logFile) handleEOF() error {
	if f.closeOnEOF || f.file.IsGZIP() {
		return io.EOF
	}

	// Re-fetch fileinfo to check if the file was truncated.
	// Errors if the file was removed/rotated after reading and before
	// calling the stat function
	info, statErr := f.file.Stat()
	if statErr != nil {
		f.log.Error("Unexpected error reading from %s; error: %s", f.file.Name(), statErr)
		return statErr
	}

	if info.Size() < f.offset {
		f.log.Debugf("File was truncated as offset (%d) > size (%d): %s", f.offset, info.Size(), f.file.Name())
		return ErrFileTruncate
	}

	return nil
}

// Close cancels the reader. It does NOT close the underlying file handle, which
// is owned by the harvester session.
func (f *logFile) Close() error {
	f.readerCtx.Cancel()
	f.log.Debugf("Closed reader. Path='%s'", f.file.Name())
	return nil
}

// updateOffset advances the tracked read offset.
func (f *logFile) updateOffset(delta int) {
	f.offsetMutx.Lock()
	f.offset += int64(delta)
	f.offsetMutx.Unlock()
}

// ReadOffset returns how far into the file the reader has consumed: the start
// offset plus every byte handed to the pipeline, including bytes buffered for an
// as-yet-incomplete trailing line.
func (f *logFile) ReadOffset() int64 {
	f.offsetMutx.Lock()
	defer f.offsetMutx.Unlock()
	return f.offset
}
