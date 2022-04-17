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
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

func TestLogFileTimedClosing(t *testing.T) {
	testCases := map[string]struct {
		inactive      time.Duration
		closeEOF      bool
		afterInterval time.Duration
		expectedErr   error
	}{
		"read from file and close inactive": {
			inactive:    2 * time.Second,
			expectedErr: ErrClosed,
		},
		"read from file and close after interval": {
			afterInterval: 3 * time.Second,
			expectedErr:   ErrClosed,
		},
		"read from file and close on EOF": {
			closeEOF:    true,
			expectedErr: io.EOF,
		},
	}

	for name, test := range testCases {
		test := test

		f := createTestLogFile()
		defer f.Close()
		defer os.Remove(f.Name())

		t.Run(name, func(t *testing.T) {
			reader, err := newFileReader(
				logp.L(),
				context.TODO(),
				f,
				readerConfig{},
				closerConfig{
					OnStateChange: stateChangeCloserConfig{
						CheckInterval: 1 * time.Second,
						Inactive:      test.inactive,
					},
					Reader: readerCloserConfig{
						OnEOF:         test.closeEOF,
						AfterInterval: test.afterInterval,
					},
				},
			)
			if err != nil {
				t.Fatalf("error while creating logReader: %+v", err)
			}

			err = readUntilError(reader)

			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestLogFileTruncated(t *testing.T) {
	f := createTestLogFile()
	defer f.Close()
	defer os.Remove(f.Name())

	reader, err := newFileReader(logp.L(), context.TODO(), f, readerConfig{}, closerConfig{})
	if err != nil {
		t.Fatalf("error while creating logReader: %+v", err)
	}

	buf := make([]byte, 1024)
	_, err = reader.Read(buf)
	assert.Nil(t, err)

	err = f.Truncate(0)
	if err != nil {
		t.Fatalf("error while truncating file: %+v", err)
	}

	err = readUntilError(reader)

	assert.Equal(t, ErrFileTruncate, err)
}

func createTestLogFile() *os.File {
	f, err := ioutil.TempFile("", "filestream_reader_test")
	if err != nil {
		panic(err)
	}
	content := []byte("first log line\nanother interesting line\na third log message\n")
	if _, err := f.Write(content); err != nil {
		panic(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		panic(err)
	}
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
