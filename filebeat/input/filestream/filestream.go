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
	"os"
	"time"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
)

var (
	ErrFileTruncate = errors.New("detected file being truncated")
	ErrClosed       = errors.New("reader closed")
)

// logFile contains all log related data
type logFile struct {
	file          *os.File
	log           *logp.Logger
	ctx           context.Context
	cancelReading context.CancelFunc

	closeInactive      time.Duration
	closeAfterInterval time.Duration
	closeOnEOF         bool

	offset       int64
	lastTimeRead time.Time
	backoff      backoff.Backoff
	tg           unison.TaskGroup
}

// newFileReader creates a new log instance to read log sources
func newFileReader(
	log *logp.Logger,
	canceler input.Canceler,
	f *os.File,
	config readerConfig,
	closerConfig readerCloserConfig,
) (*logFile, error) {
	offset, err := f.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	l := &logFile{
		file:               f,
		log:                log,
		closeInactive:      closerConfig.Inactive,
		closeAfterInterval: closerConfig.AfterInterval,
		closeOnEOF:         closerConfig.OnEOF,
		offset:             offset,
		lastTimeRead:       time.Now(),
		backoff:            backoff.NewExpBackoff(canceler.Done(), config.Backoff.Init, config.Backoff.Max),
		tg:                 unison.TaskGroup{},
	}

	l.ctx, l.cancelReading = ctxtool.WithFunc(ctxtool.FromCanceller(canceler), func() {
		err := l.tg.Stop()
		if err != nil {
			l.log.Errorf("Error while stopping filestream logFile reader: %v", err)
		}
	})

	l.startFileMonitoringIfNeeded()

	return l, nil
}

// Read reads from the reader and updates the offset
// The total number of bytes read is returned.
func (f *logFile) Read(buf []byte) (int, error) {
	totalN := 0

	for f.ctx.Err() == nil {
		n, err := f.file.Read(buf)
		if n > 0 {
			f.offset += int64(n)
			f.lastTimeRead = time.Now()
		}
		totalN += n

		// Read from source completed without error
		// Either end reached or buffer full
		if err == nil {
			// reset backoff for next read
			f.backoff.Reset()
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

		f.log.Debugf("End of file reached: %s; Backoff now.", f.file.Name())
		f.backoff.Wait()
	}

	return 0, ErrClosed
}

func (f *logFile) startFileMonitoringIfNeeded() {
	if f.closeInactive == 0 && f.closeAfterInterval == 0 {
		return
	}

	if f.closeInactive > 0 {
		f.tg.Go(func(ctx unison.Canceler) error {
			f.closeIfTimeout(ctx)
			return nil
		})
	}

	if f.closeAfterInterval > 0 {
		f.tg.Go(func(ctx unison.Canceler) error {
			f.closeIfInactive(ctx)
			return nil
		})
	}
}

func (f *logFile) closeIfTimeout(ctx unison.Canceler) {
	timer := time.NewTimer(f.closeAfterInterval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			f.cancelReading()
			return
		}
	}
}

func (f *logFile) closeIfInactive(ctx unison.Canceler) {
	// This can be made configureble if users need a more flexible
	// cheking for inactive files.
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			age := time.Since(f.lastTimeRead)
			if age > f.closeInactive {
				f.cancelReading()
				return
			}
		}
	}
}

// errorChecks determines the cause for EOF errors, and how the EOF event should be handled
// based on the config options.
func (f *logFile) errorChecks(err error) error {
	if err != io.EOF {
		f.log.Error("Unexpected state reading from %s; error: %s", f.file.Name(), err)
		return err
	}

	return f.handleEOF()
}

func (f *logFile) handleEOF() error {
	if f.closeOnEOF {
		return io.EOF
	}

	// Refetch fileinfo to check if the file was truncated.
	// Errors if the file was removed/rotated after reading and before
	// calling the stat function
	info, statErr := f.file.Stat()
	if statErr != nil {
		f.log.Error("Unexpected error reading from %s; error: %s", f.file.Name(), statErr)
		return statErr
	}

	// check if file was truncated
	if info.Size() < f.offset {
		f.log.Debugf("File was truncated as offset (%d) > size (%d): %s", f.offset, info.Size(), f.file.Name())
		return ErrFileTruncate
	}

	return nil
}

// Close
func (f *logFile) Close() error {
	f.cancelReading()
	return f.file.Close()
}
