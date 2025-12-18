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

func TestLogFileTimedClosing(t *testing.T) {
	testCases := []struct {
		name           string
		createFile     func(t *testing.T) *os.File
		waitBeforeRead time.Duration
		inactive       time.Duration
		closeEOF       bool
		afterInterval  time.Duration
		expectedErr    error
	}{
		{name: "plain: read from file and close inactive",
			createFile:  createTestPlainLogFile,
			inactive:    2 * time.Second,
			expectedErr: ErrInactive,
		},
		{name: "plain: read from file and close after interval",
			createFile:    createTestPlainLogFile,
			afterInterval: 3 * time.Second,
			expectedErr:   ErrClosed,
		},
		{name: "plain: read from file and close on EOF",
			createFile:  createTestPlainLogFile,
			closeEOF:    true,
			expectedErr: io.EOF,
		},
		{name: "GZIP: read from file and close on EOF",
			createFile:  createTestGzipLogFile,
			closeEOF:    true,
			expectedErr: io.EOF,
		},
		{name: "GZIP: read from file and close after interval",
			createFile:     createTestPlainLogFile,
			afterInterval:  3 * time.Second,
			waitBeforeRead: 3 * time.Second,
			expectedErr:    ErrClosed,
		},
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
					OnStateChange: stateChangeCloserConfig{
						CheckInterval: 1 * time.Second,
						Inactive:      tc.inactive,
					},
					Reader: readerCloserConfig{
						OnEOF:         tc.closeEOF,
						AfterInterval: tc.afterInterval,
					},
				},
			)
			if err != nil {
				t.Fatalf("error while creating logReader: %+v", err)
			}

			if tc.waitBeforeRead > 0 {
				// GZIP files aren't kept open, thus we need to wait for
				// 'AfterInterval' to elapse before reading.
				time.Sleep(tc.waitBeforeRead)
			}

			err = readUntilError(reader)
			assert.ErrorIs(t, err, tc.expectedErr)
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
			assert.Nil(t, err)

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
