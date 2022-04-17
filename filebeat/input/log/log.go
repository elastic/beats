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

package log

import (
	"io"
	"os"
	"time"

	"github.com/menderesk/beats/v7/filebeat/harvester"
	"github.com/menderesk/beats/v7/filebeat/input/file"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

// Log contains all log related data
type Log struct {
	logger *logp.Logger

	fs           harvester.Source
	offset       int64
	config       LogConfig
	lastTimeRead time.Time
	backoff      time.Duration
	done         chan struct{}
}

// NewLog creates a new log instance to read log sources
func NewLog(
	logger *logp.Logger,
	fs harvester.Source,
	config LogConfig,
) (*Log, error) {
	var offset int64
	if seeker, ok := fs.(io.Seeker); ok {
		var err error
		offset, err = seeker.Seek(0, os.SEEK_CUR)
		if err != nil {
			return nil, err
		}
	}

	return &Log{
		logger:       logger,
		fs:           fs,
		offset:       offset,
		config:       config,
		lastTimeRead: time.Now(),
		backoff:      config.Backoff,
		done:         make(chan struct{}),
	}, nil
}

// Read reads from the reader and updates the offset
// The total number of bytes read is returned.
func (f *Log) Read(buf []byte) (int, error) {
	totalN := 0

	for {
		select {
		case <-f.done:
			return 0, ErrClosed
		default:
		}

		err := f.checkFileDisappearedErrors()
		if err != nil {
			return totalN, err
		}

		n, err := f.fs.Read(buf)
		if n > 0 {
			f.offset += int64(n)
			f.lastTimeRead = time.Now()
		}
		totalN += n

		// Read from source completed without error
		// Either end reached or buffer full
		if err == nil {
			// reset backoff for next read
			f.backoff = f.config.Backoff
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

		f.logger.Debugf("End of file reached: %s; Backoff now.", f.fs.Name())
		f.wait()
	}
}

// errorChecks determines the cause for EOF errors, and how the EOF event should be handled
// based on the config options.
func (f *Log) errorChecks(err error) error {
	if err != io.EOF {
		f.logger.Errorf("Unexpected state reading from %s; error: %s", f.fs.Name(), err)
		return err
	}

	// At this point we have hit EOF!

	// Stdin is not continuable
	if !f.fs.Continuable() {
		f.logger.Debugf("Source is not continuable: %s", f.fs.Name())
		return err
	}

	if f.config.CloseEOF {
		return err
	}

	// Refetch fileinfo to check if the file was truncated.
	// Errors if the file was removed/rotated after reading and before
	// calling the stat function
	info, statErr := f.fs.Stat()
	if statErr != nil {
		f.logger.Errorf("Unexpected error reading from %s; error: %s", f.fs.Name(), statErr)
		return statErr
	}

	// check if file was truncated
	if info.Size() < f.offset {
		f.logger.Debugf("File was truncated as offset (%d) > size (%d): %s", f.offset, info.Size(), f.fs.Name())
		return ErrFileTruncate
	}

	// Check file wasn't read for longer then CloseInactive
	age := time.Since(f.lastTimeRead)
	if age > f.config.CloseInactive {
		return ErrInactive
	}

	return nil
}

// checkFileDisappearedErrors checks if the log file has been removed or renamed (rotated).
func (f *Log) checkFileDisappearedErrors() error {
	// No point doing a stat call on the file if configuration options are
	// not enabled
	if !f.config.CloseRenamed && !f.config.CloseRemoved {
		return nil
	}

	// Refetch fileinfo to check if the file was renamed or removed.
	// Errors if the file was removed/rotated after reading and before
	// calling the stat function
	info, statErr := f.fs.Stat()
	if statErr != nil {
		f.logger.Errorf("Unexpected error reading from %s; error: %s", f.fs.Name(), statErr)
		return statErr
	}

	if f.config.CloseRenamed {
		// Check if the file can still be found under the same path
		if !file.IsSameFile(f.fs.Name(), info) {
			f.logger.Debugf("close_renamed is enabled and file %s has been renamed", f.fs.Name())
			return ErrRenamed
		}
	}

	if f.config.CloseRemoved {
		// Check if the file name exists. See https://github.com/menderesk/filebeat/issues/93
		if f.fs.Removed() {
			f.logger.Debugf("close_removed is enabled and file %s has been removed", f.fs.Name())
			return ErrRemoved
		}
	}

	return nil
}

func (f *Log) wait() {
	// Wait before trying to read file again. File reached EOF.
	select {
	case <-f.done:
		return
	case <-time.After(f.backoff):
	}

	// Increment backoff up to maxBackoff
	if f.backoff < f.config.MaxBackoff {
		f.backoff = f.backoff * time.Duration(f.config.BackoffFactor)
		if f.backoff > f.config.MaxBackoff {
			f.backoff = f.config.MaxBackoff
		}
	}
}

// Close closes the done channel but no th the file handler
func (f *Log) Close() error {
	close(f.done)
	// Note: File reader is not closed here because that leads to race conditions
	return nil
}
