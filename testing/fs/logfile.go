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

package fs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LogFile wraps a *os.File and makes it more suitable for tests.
// Methods to search and wait for substrings in lines are provided,
// they keep track of the offset, ensuring ordering when
// when searching.
type LogFile struct {
	*os.File
	offset int64
}

// NewLogFile returns a new LogFile which wraps a os.File meant to be used
// for testing. Methods to search and wait for strings in the file are
// provided. 'dir' and 'pattern' are passed directly to os.CreateTemp.
// It is the callers responsibility to remove the file. To keep the file
// when the test fails, use [TempDir] to create a folder.
func NewLogFile(t testing.TB, dir, pattern string) *LogFile {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatalf("cannot create log file: %s", err)
	}

	lf := &LogFile{
		File: f,
	}

	t.Cleanup(func() {
		if err := f.Sync(); err != nil {
			t.Logf("cannot sync log file: %s", err)
		}

		if err := f.Close(); err != nil {
			t.Logf("cannot close log file: %s", err)
		}
	})

	return lf
}

// WaitLogsContains waits for the specified string s to be present in the logs within
// the given timeout duration and fails the test if s is not found.
// It keeps track of the log file offset, reading only new lines. Each
// subsequent call to WaitLogsContains will only check logs not yet evaluated.
// msgAndArgs should be a format string and arguments that will be printed
// if the logs are not found, providing additional context for debugging.
func (l *LogFile) WaitLogsContains(t testing.TB, s string, timeout time.Duration, msgAndArgs ...any) {
	t.Helper()
	require.EventuallyWithT(
		t,
		func(c *assert.CollectT) {
			found, err := l.FindInLogs(s)
			if err != nil {
				c.Errorf("cannot check the log file: %s", err)
				return
			}

			if !found {
				c.Errorf("did not find '%s' in the logs", s)
			}
		},
		timeout,
		100*time.Millisecond,
		msgAndArgs...)
}

// LogContains searches for str in the log file keeping track of the
// offset. If there is any issue reading the log file, then t.Fatalf is called,
// if str is not present in the logs, t.Errorf is called.
func (l *LogFile) LogContains(t testing.TB, str string) {
	t.Helper()
	found, err := l.FindInLogs(str)
	if err != nil {
		t.Fatalf("cannot read log file: %s", err)
	}

	if !found {
		t.Errorf("'%s' not found in logs", str)
	}
}

// FindInLogs searches for str in the log file keeping track of the offset.
// It returns true if str is found in the logs. If there are any errors,
// it returns false and the error
func (l *LogFile) FindInLogs(str string) (bool, error) {
	// Open the file again so we can seek and not interfere with
	// the logger writing to it.
	f, err := os.Open(l.Name())
	if err != nil {
		return false, fmt.Errorf("cannot open log file for reading: %w", err)
	}

	if _, err := f.Seek(l.offset, io.SeekStart); err != nil {
		return false, fmt.Errorf("cannot seek log file: %w", err)
	}

	r := bufio.NewReader(f)
	for {
		data, err := r.ReadBytes('\n')
		line := string(data)
		l.offset += int64(len(data))

		if err != nil {
			if !errors.Is(err, io.EOF) {
				return false, fmt.Errorf("error reading log file '%s': %w", l.Name(), err)
			}
			break
		}

		if strings.Contains(line, str) {
			return true, nil
		}
	}

	return false, nil
}

// ResetOffset resets the log file offset
func (l *LogFile) ResetOffset() {
	l.offset = 0
}
