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
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

// TestLogFileCloseOnEOF covers the only close condition logFile still evaluates
// itself: close.reader.on_eof (and GZIP, which always closes on EOF). The
// on-state-change conditions (inactive/removed/renamed) and close-after-interval
// are now evaluated by the harvester runner's waker and covered by integration tests.
func TestLogFileCloseOnEOF(t *testing.T) {
	testCases := []struct {
		name       string
		createFile func(t *testing.T) *os.File
	}{
		{name: "plain: read from file and close on EOF", createFile: createTestPlainLogFile},
		{name: "GZIP: read from file and close on EOF", createFile: createTestGzipLogFile},
	}

	for _, tc := range testCases {
		fs := filestream{
			readerConfig: readerConfig{BufferSize: 512},
			compression:  CompressionAuto}
		f, err := fs.newFile(tc.createFile(t))
		require.NoError(t, err,
			"could not create file for reading")
		defer f.Close()
		defer os.Remove(f.Name())

		t.Run(tc.name, func(t *testing.T) {
			reader, err := newFileReader(
				logp.NewNopLogger(),
				context.TODO(),
				f,
				readerConfig{},
				closerConfig{
					Reader: readerCloserConfig{OnEOF: true},
				},
			)
			if err != nil {
				t.Fatalf("error while creating logReader: %+v", err)
			}

			err = readUntilError(reader)
			assert.ErrorIs(t, err, io.EOF)
		})
	}
}

func TestLogFileTruncated(t *testing.T) {
	tcs := []struct {
		name       string
		createFile func(t *testing.T) *os.File
		truncateFn func(t *testing.T, f File) error
		wantErr    error
	}{
		{name: "plain: ErrFileTruncate",
			createFile: createTestPlainLogFile,
			truncateFn: func(t *testing.T, f File) error {
				return f.OSFile().Truncate(0)
			},
			wantErr: ErrFileTruncate,
		},
		{name: "gzip: io.EOF",
			createFile: createTestGzipLogFile,
			truncateFn: func(t *testing.T, f File) error {
				osf := f.OSFile()
				gzw := gzip.NewWriter(osf)

				_, err := io.Copy(gzw, bytes.NewBuffer([]byte("truncated data\n")))
				require.NoError(t, err, "could not write data to gzip file")

				require.NoErrorf(t, gzw.Close(),
					"could not close gzip writer")
				require.NoError(t, osf.Sync(), "could not sync os file")

				return nil
			},
			wantErr: io.EOF,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			osFile := tc.createFile(t)

			fs := filestream{
				readerConfig: readerConfig{BufferSize: 512},
				compression:  CompressionAuto}

			f, err := fs.newFile(osFile)
			require.NoError(t, err, "could not create file for reading")

			defer f.Close()
			defer os.Remove(f.Name())

			reader, err := newFileReader(
				logp.NewNopLogger(), context.TODO(), f, fs.readerConfig, fs.closerConfig)
			require.NoError(t, err, "error while creating logReader")

			buf := make([]byte, 32)
			_, err = reader.Read(buf)
			assert.NoError(t, err)

			err = tc.truncateFn(t, f)
			require.NoError(t, err, "error while truncating file")

			err = readUntilError(reader)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func createTestPlainLogFile(t *testing.T) *os.File {
	f, err := os.CreateTemp("", "filestream_reader_test")
	require.NoError(t, err, "could not create temp file")

	content := []byte("first log line\nanother interesting line\na third log message\n")
	_, err = f.Write(content)
	require.NoError(t, err, "could not write to temp file")

	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err, "could not seek to start of temp file")

	return f
}

func createTestGzipLogFile(t *testing.T) *os.File {
	plain := createTestPlainLogFile(t)

	f, err := os.CreateTemp("", "filestream_reader_test.*.gz")
	require.NoError(t, err, "could not create temp file")

	data, err := io.ReadAll(plain)
	require.NoError(t, err, "could not read from file")

	var tempBuffer bytes.Buffer
	gw := gzip.NewWriter(&tempBuffer)
	_, err = gw.Write(data)
	require.NoError(t, err, "failed to write plain content to gzip writer")
	err = gw.Close()
	require.NoError(t, err, "failed to close gzip writer")

	_, err = f.Write(tempBuffer.Bytes())
	require.NoError(t, err, "failed to write to gzip file")

	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err, "could not seek to start of gzip file")

	return f
}

func readUntilError(reader *logFile) error {
	buf := make([]byte, 1024)
	_, err := reader.Read(buf)
	for err == nil {
		buf := make([]byte, 1024)
		_, err = reader.Read(buf)
	}
	return err
}

// TestLogFileNonBlocking tests logFile is non-blocking: it returns ErrWouldBlock
// at EOF instead of waiting on the read backoff, and it resumes reading once new
// data is appended.
func TestLogFileNonBlocking(t *testing.T) {
	osFile := createTestPlainLogFile(t)
	fs := filestream{
		readerConfig: readerConfig{BufferSize: 512},
		compression:  CompressionAuto,
	}
	f, err := fs.newFile(osFile)
	require.NoError(t, err, "could not create file for reading")
	t.Cleanup(func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	})

	reader, err := newFileReader(
		logp.NewNopLogger(), context.TODO(), f, readerConfig{}, closerConfig{})
	require.NoError(t, err, "error while creating logReader")

	// Drain the initial content written by createTestPlainLogFile.
	content := readAllAvailable(t, reader)
	require.NotEmpty(t, content, "expected to read the initial file content")

	// At EOF a non-blocking reader must return ErrWouldBlock promptly instead
	// of blocking on the read backoff.
	n, err := readWithTimeout(t, reader, make([]byte, 1024), time.Second)
	assert.Zero(t, n, "no bytes should be read at EOF")
	assert.ErrorIs(t, err, ErrWouldBlock,
		"non-blocking reader must return ErrWouldBlock at EOF")

	// Once new data is appended the reader must pick it up rather than keep
	// returning ErrWouldBlock.
	appendToFile(t, f.Name(), "a new line\n")
	more := readAllAvailable(t, reader)
	assert.Equal(t, "a new line\n", string(more),
		"non-blocking reader must read newly appended data")
}

// readWithTimeout runs reader.Read in a goroutine and fails the test if it does
// not return within timeout, turning a blocking regression into a clear failure
// instead of a hung test.
func readWithTimeout(t *testing.T, reader *logFile, buf []byte, timeout time.Duration) (int, error) {
	t.Helper()
	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1)
	go func() {
		n, err := reader.Read(buf)
		ch <- result{n: n, err: err}
	}()

	select {
	case r := <-ch:
		return r.n, r.err
	case <-time.After(timeout):
		t.Fatalf("Read did not return within %s; the non-blocking reader appears to be blocking", timeout)
		return 0, nil
	}
}

// readAllAvailable reads from a non-blocking reader until ErrWouldBlock and
// returns everything read.
func readAllAvailable(t *testing.T, reader *logFile) []byte {
	t.Helper()
	var out []byte
	for {
		buf := make([]byte, 1024)
		n, err := readWithTimeout(t, reader, buf, time.Second)
		out = append(out, buf[:n]...)
		if err != nil {
			require.ErrorIs(t, err, ErrWouldBlock,
				"unexpected error while draining non-blocking reader")
			return out
		}
	}
}

func appendToFile(t *testing.T, path, data string) {
	t.Helper()
	wf, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err, "could not open file for appending")
	defer wf.Close()

	_, err = wf.WriteString(data)
	require.NoError(t, err, "could not append to file")
	require.NoError(t, wf.Sync(), "could not sync appended data")
}
