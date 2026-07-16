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

//go:build !windows

package filestream

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

// these tests are separated as one cannot delete/rename files
// while another process is working with it on Windows

// TestLogFileRenamed verifies the harvester-refactor contract for a renamed
// file: logFile no longer detects the rename itself (that moved to the session's
// Poll, driven by the waker). An already-open reader keeps draining its open
// descriptor and returns ErrWouldBlock at EOF even with OnStateChange.Renamed
// configured; it only closes (ErrClosed) once its reader context is cancelled.
func TestLogFileRenamed(t *testing.T) {
	f := createTestLogFile(t)
	defer f.Close()

	renamedFile := f.Name() + ".renamed"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, err := newFileReader(
		logp.NewNopLogger(),
		ctx,
		f,
		closerConfig{
			OnStateChange: stateChangeCloserConfig{
				CheckInterval: 1 * time.Second,
				Renamed:       true,
			},
		},
	)
	require.NoError(t, err, "error while creating logReader")

	// Drain the initial content the file was created with.
	require.NotEmpty(t, readAllAvailable(t, reader),
		"expected to read the initial file content")

	require.NoError(t, os.Rename(f.Name(), renamedFile),
		"error while renaming file")
	defer os.Remove(renamedFile)

	// The rename does not close the reader: logFile does not act on
	// OnStateChange, so at EOF it just reports ErrWouldBlock.
	_, err = readWithTimeout(t, reader, make([]byte, 1024), time.Second)
	assert.ErrorIs(t, err, ErrWouldBlock,
		"a renamed file must not close the reader; the open fd keeps reading")

	// Cancelling the reader context is what closes it.
	cancel()
	assert.Equal(t, ErrClosed, readUntilError(reader),
		"a cancelled reader context must return ErrClosed")
}

// TestLogFileRemoved verifies the harvester-refactor contract for a removed
// file: like a rename, logFile does not detect the removal itself. On a
// non-Windows system the open descriptor stays valid, so the reader keeps
// returning ErrWouldBlock at EOF even with OnStateChange.Removed configured, and
// only closes (ErrClosed) once its reader context is cancelled.
func TestLogFileRemoved(t *testing.T) {
	f := createTestLogFile(t)
	defer f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, err := newFileReader(
		logp.NewNopLogger(),
		ctx,
		f,
		closerConfig{
			OnStateChange: stateChangeCloserConfig{
				CheckInterval: 1 * time.Second,
				Removed:       true,
			},
		},
	)
	require.NoError(t, err, "error while creating logReader")

	// Drain the initial content the file was created with.
	require.NotEmpty(t, readAllAvailable(t, reader),
		"expected to read the initial file content")

	require.NoError(t, os.Remove(f.Name()), "error while removing file")

	// The removal does not close the reader: the open fd remains valid, so at
	// EOF the reader reports ErrWouldBlock rather than closing.
	_, err = readWithTimeout(t, reader, make([]byte, 1024), time.Second)
	assert.ErrorIs(t, err, ErrWouldBlock,
		"a removed file must not close the reader; the open fd keeps reading")

	// Cancelling the reader context is what closes it.
	cancel()
	assert.Equal(t, ErrClosed, readUntilError(reader),
		"a cancelled reader context must return ErrClosed")
}

// createTestLogFile creates a temporary plain-text log file with a few lines of
// content, wrapped as a filestream File ready to be passed to newFileReader.
func createTestLogFile(t *testing.T) File {
	t.Helper()
	fs := filestream{
		readerConfig: readerConfig{BufferSize: 512},
		compression:  CompressionNone,
	}
	f, err := fs.newFile(createTestPlainLogFile(t))
	require.NoError(t, err, "could not create test log file")
	return f
}
