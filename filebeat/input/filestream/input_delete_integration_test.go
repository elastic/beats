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

//go:build integration

package filestream

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestFilestreamDelete(t *testing.T) {
	testCases := map[string]map[string]any{
		"on EOF": {
			"prospector.scanner.check_interval":     "1s",
			"close.reader.on_eof":                   true,
			"delete.enabled":                        true,
			"prospector.scanner.fingerprint.length": 64,
			"delete.grace_period":                   0,
		},
		"on Inactive": {
			"prospector.scanner.check_interval":     "1s",
			"close.on_state_change.inactive":        "1s",
			"close.reader.on_eof":                   false,
			"delete.enabled":                        true,
			"prospector.scanner.fingerprint.length": 64,
			"delete.grace_period":                   0,
		},
	}

	for name, conf := range testCases {
		t.Run(name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)
			logfile := strings.ReplaceAll(t.Name(), "/", "_") + ".log"
			conf["id"] = "fake-ID-" + uuid.Must(uuid.NewV4()).String()
			conf["paths"] = []string{env.abspath(logfile)}
			inp := env.mustCreateInput(conf)

			testlines := bytes.NewBuffer(nil)
			for i := range 10 {
				fmt.Fprintf(testlines, "[%02d] sample log line\n", i)
			}
			env.mustWriteToFile(logfile, testlines.Bytes())

			ctx, cancelInput := context.WithCancel(context.Background())
			env.startInput(ctx, t.Name(), inp)
			defer cancelInput()

			env.waitUntilEventCount(10)
			logFile := env.abspath(logfile)
			require.Eventuallyf(t,
				func() bool {
					_, err := os.Stat(logFile)
					return errors.Is(err, os.ErrNotExist)
				},
				10*time.Second,
				time.Second,
				"%q was not deleted", logFile)
		})
	}
}

func TestFilestreamDeleteFile(t *testing.T) {
	testCases := map[string]struct {
		cursorPending     int
		expectFileDeleted bool
		expectError       bool
		canDelete         bool
		gracePeriodErr    error
	}{
		"happy path": {
			cursorPending:     -1,
			expectFileDeleted: true,
			canDelete:         true,
		},

		"pending events": {
			cursorPending:     2,
			expectFileDeleted: false,
			canDelete:         true,
		},

		"waitGracePeriod returns false": {
			cursorPending:     -1,
			expectFileDeleted: false,
		},

		"waitGracePeriod returns error": {
			cursorPending:     -1,
			expectFileDeleted: false,
			expectError:       true,
			canDelete:         true,
			gracePeriodErr:    errors.New("any error"),
		},
	}

	for name, tc := range testCases {
		env := newInputTestingEnvironment(t)

		f := filestream{
			deleterConfig: deleterConfig{
				retryBackoff: 10 * time.Millisecond,
			},
			scannerCheckInterval: 10 * time.Millisecond,
			removeFn:             os.Remove,
			waitGracePeriodFn: func(
				ctx v2.Context,
				logger *logp.Logger,
				cursor loginp.Cursor,
				path string,
				gracePeriod time.Duration,
				checkInterval time.Duration,
				statFn func(string) (os.FileInfo, error),
			) (bool, error) {
				return tc.canDelete, tc.gracePeriodErr
			},
		}

		data := []byte("foo bar\n")
		logFile := env.mustWriteToFile("logfile.log", data)
		cur := loginp.NewCursorForTest(
			t.Name()+":"+logFile,
			int64(len(data)),
			tc.cursorPending)

		t.Run(name, func(t *testing.T) {
			v2Ctx := v2.Context{
				ID:          t.Name(),
				Cancelation: t.Context(),
				Logger:      env.testLogger.Logger,
			}

			err := f.deleteFile(v2Ctx, v2Ctx.Logger, cur, logFile)
			if !tc.expectError && err != nil {
				t.Fatalf("did not expect an error from 'deleteFile': %s", err)
			} else if tc.expectError && err == nil {
				t.Fatal("expecting an error deleteFile 'from'")
			}

			requireFileDeleted(t, logFile, tc.expectFileDeleted)
		})
	}
}

func TestFilestreamDeleteFileRemoveRetries(t *testing.T) {
	removeErr := errors.New("oops")
	env := newInputTestingEnvironment(t)
	tickChan := make(chan time.Time)

	f := filestream{
		deleterConfig: deleterConfig{
			retries:      2,
			retryBackoff: 10 * time.Millisecond,
		},
		scannerCheckInterval: time.Millisecond,
		waitGracePeriodFn: func(
			ctx v2.Context,
			logger *logp.Logger,
			cursor loginp.Cursor,
			path string,
			gracePeriod time.Duration,
			checkInterval time.Duration,
			statFn func(string) (os.FileInfo, error),
		) (bool, error) {
			return true, nil
		},
		removeFn: func(string) error { return removeErr },
	}

	tickFn := func(d time.Duration) <-chan time.Time {
		// Ensure tickFn is called with the correct parameter
		if d != f.deleterConfig.retryBackoff {
			t.Errorf(
				"'tickFn' called with %q, expecting %q",
				d.String(),
				f.deleterConfig.retryBackoff.String())
		}

		return tickChan
	}

	f.tickFn = tickFn

	data := []byte("foo bar")
	logFile := env.mustWriteToFile("log.log", data)
	cur := loginp.NewCursorForTest(
		t.Name()+":"+logFile,
		int64(len(data)),
		-1)

	// 1. Test we return when the context is cancelled
	t.Run("retry stops when context is cancelled", func(t *testing.T) {
		// Ensure we get errors reported as part of the correct sub test
		env.t = t
		ctx, cancel := context.WithCancel(t.Context())
		v2Ctx := v2.Context{
			ID:          t.Name(),
			Cancelation: ctx,
			Logger:      env.testLogger.Logger,
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		var deleteErr error
		go func() {
			defer wg.Done()
			deleteErr = f.deleteFile(v2Ctx, env.testLogger.Logger, cur, logFile)
		}()

		cancel()
		wg.Wait()

		if !errors.Is(deleteErr, context.Canceled) {
			t.Fatalf("expecting 'context cancelled' when the context is cancelled, got: %v", deleteErr)
		}
	})

	t.Run("file is externally removed", func(t *testing.T) {
		// Ensure we get errors reported as part of the correct sub test
		env.t = t
		v2Ctx := v2.Context{
			ID:          t.Name(),
			Cancelation: t.Context(),
			Logger:      env.testLogger.Logger,
		}

		count := atomic.Int32{}
		removeFn := func(string) error {
			count.Add(1)
			return removeErr
		}

		f.removeFn = removeFn
		wg := sync.WaitGroup{}
		wg.Add(1)
		var deleteErr error
		deleteDone := atomic.Bool{}
		go func() {
			defer wg.Done()
			defer deleteDone.Store(true)
			deleteErr = f.deleteFile(v2Ctx, env.testLogger.Logger, cur, logFile)
		}()

		tickChan <- time.Now()
		if count.Load() != 2 {
			t.Fatalf("removeFn must have been called twice, but it was called %d", count.Load())
		}

		if deleteDone.Load() {
			t.Fatal("deleteFile must still be running")
		}

		fileRemoved := atomic.Bool{}
		f.removeFn = func(s string) error {
			count.Add(1)
			fileRemoved.Store(true)
			return nil
		}

		// Run the retry loop
		tickChan <- time.Now()
		wg.Wait()

		if deleteErr != nil {
			t.Fatalf("expecting no error, got %v", deleteErr)
		}
		if !fileRemoved.Load() {
			t.Fatal("expecting 'removeFn' to be called")
		}

		env.logContains(fmt.Sprintf("'%s' removed", logFile))
	})

	t.Run("file removed externally is a success", func(t *testing.T) {
		// Ensure we get errors reported as part of the correct sub test
		env.t = t
		v2Ctx := v2.Context{
			ID:          t.Name(),
			Cancelation: t.Context(),
			Logger:      env.testLogger.Logger,
		}

		count := atomic.Int32{}
		removeFn := func(string) error {
			count.Add(1)
			if count.Load() == 2 {
				return os.ErrNotExist
			}

			return removeErr
		}

		f.removeFn = removeFn
		wg := sync.WaitGroup{}
		wg.Add(1)
		var gotErr error
		go func() {
			defer wg.Done()
			gotErr = f.deleteFile(v2Ctx, env.testLogger.Logger, cur, logFile)
		}()

		// Run the retry loop
		tickChan <- time.Now()
		wg.Wait()

		if gotErr != nil {
			t.Fatalf("expecting no error, got %v", gotErr)
		}

		env.logContains(fmt.Sprintf("could not remove '%s', retrying in 2s. Error: %s", logFile, removeErr))
	})

	t.Run("exhausted retries returns error", func(t *testing.T) {
		// Ensure we get errors reported as part of the correct sub test
		env.t = t
		v2Ctx := v2.Context{
			ID:          t.Name(),
			Cancelation: t.Context(),
			Logger:      env.testLogger.Logger,
		}

		count := atomic.Int32{}
		removeFn := func(string) error {
			count.Add(1)
			return removeErr
		}

		f.removeFn = removeFn
		f.deleterConfig.retries = 10
		wg := sync.WaitGroup{}
		wg.Add(1)
		var gotErr error
		deleteDone := atomic.Bool{}
		go func() {
			defer wg.Done()
			defer deleteDone.Store(true)
			gotErr = f.deleteFile(v2Ctx, env.testLogger.Logger, cur, logFile)
		}()

		for i := range 9 {
			tickChan <- time.Now()

			// Wait for removeFn to be called
			require.Eventually(t,
				func() bool {
					//nolint:gosec // It's a test, i is always very small
					return count.Load() == int32(i+2)
				},
				time.Second,
				time.Millisecond,
				"removeFn was not called")

			if deleteDone.Load() {
				t.Fatalf("delete cannot be done while retrying")
			}
		}

		// Last retry
		tickChan <- time.Now()

		wg.Wait()
		expectedErrMsg := fmt.Sprintf(
			"cannot remove '%s' after %d retries. Last error: %s",
			logFile,
			f.deleterConfig.retries,
			removeErr)
		if gotErr == nil {
			t.Fatal("expecting error from deleteFile")
		}
		if gotErr.Error() != expectedErrMsg {
			t.Fatalf("expecting error message to be %q, got %q", expectedErrMsg, gotErr.Error())
		}
	})
}

// TestFilestreamDeleteFileReturnsError tests how filebeat.Run handles the
// error returned by filebeat.deleteFile, this ensures the harvester is
// correctly closed and errors are correctly reported.
func TestFilestreamDeleteFileReturnsError(t *testing.T) {
	env := newInputTestingEnvironment(t)
	encodingFactory, ok := encoding.FindEncoding("")
	if !ok {
		t.Fatal("cannot find Plain encoding factory")
	}

	testCases := map[string]struct {
		expectedErr       error
		expectFileDeleted bool
	}{
		"error returned": {
			expectedErr:       errors.New("oops"),
			expectFileDeleted: false,
		},
		"no error - file removed": {
			expectedErr:       nil,
			expectFileDeleted: true,
		},
	}

	for name, tc := range testCases {
		f := filestream{
			readerConfig:    defaultReaderConfig(),
			encodingFactory: encodingFactory,
			deleterConfig: deleterConfig{
				retryBackoff: 10 * time.Millisecond,
				Enabled:      true,
			},
			closerConfig: closerConfig{
				Reader: readerCloserConfig{
					OnEOF: true,
				},
			},
			scannerCheckInterval: 10 * time.Millisecond,
			removeFn:             os.Remove,
			waitGracePeriodFn: func(
				ctx v2.Context,
				logger *logp.Logger,
				cursor loginp.Cursor,
				path string,
				gracePeriod time.Duration,
				checkInterval time.Duration,
				statFn func(string) (os.FileInfo, error),
			) (bool, error) {
				return true, tc.expectedErr
			},
		}

		data := []byte("foo bar\n")
		logFile := env.mustWriteToFile("logfile.log", data)
		cur := loginp.NewCursorForTest(
			t.Name()+":"+logFile,
			int64(len(data)),
			-1)

		v2Ctx := v2.Context{
			ID:          t.Name(),
			Cancelation: t.Context(),
			Logger:      env.testLogger.Logger,
		}

		fs := fileSource{
			fileID:  "foo bar",
			newPath: logFile,
		}

		t.Run(name, func(t *testing.T) {
			err := f.Run(v2Ctx, fs, cur, nil, loginp.NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger()))
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf(
					"filebeat.Run did not return the error generated by deleteFile. "+
						"Expecting '%v', got '%v'",
					tc.expectedErr,
					err)
			}

			requireFileDeleted(t, logFile, tc.expectFileDeleted)
		})
	}
}

func TestFilestreamWaitGracePeriod(t *testing.T) {
	testCases := map[string]struct {
		data         []byte
		cursorOffset int64
		realSize     bool
		expectError  bool
		expected     bool
		gracePeriod  time.Duration
		doesNotExist bool
	}{
		"happy path": {
			data:     []byte("foo bar\n"),
			realSize: true,
			expected: true,
		},
		"different size from cursor": {
			data:         []byte("foo bar\n"),
			cursorOffset: 42,
			expected:     false,
		},
		"file does not exist": {
			expected:     false,
			doesNotExist: true,
		},
		"happy path with grace period": {
			data:        []byte("foo bar\n"),
			realSize:    true,
			expected:    true,
			gracePeriod: 200 * time.Millisecond,
		},
		"grace period and different file size": {
			data:         []byte("foo bar\n"),
			cursorOffset: 42,
			expected:     false,
			gracePeriod:  200 * time.Millisecond,
		},
		"grace period and file does not exist": {
			doesNotExist: true,
			expected:     false,
			gracePeriod:  200 * time.Millisecond,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)
			var logFile string

			if tc.doesNotExist {
				logFile = "foo-bar"
			} else {
				logFile = env.mustWriteToFile("logfile.log", tc.data)
			}

			offset := tc.cursorOffset
			if tc.realSize {
				st, err := os.Stat(logFile)
				if err != nil {
					t.Fatalf("cannot stat %q: %s", logFile, err)
				}
				offset = st.Size()
			}

			cur := loginp.NewCursorForTest("foo-bar", offset, -1)
			v2Ctx := v2.Context{
				ID:          t.Name(),
				Cancelation: t.Context(),
				Logger:      env.testLogger.Logger,
			}

			start := time.Now()
			got, err := waitGracePeriod(
				v2Ctx,
				env.testLogger.Logger,
				cur,
				logFile,
				tc.gracePeriod,
				10*time.Millisecond,
				os.Stat,
			)
			delta := time.Since(start)
			if !tc.expectError && err != nil {
				t.Fatalf("did not expect an error from 'deleteFile': %s", err)
			} else if tc.expectError && err == nil {
				t.Fatal("expecting an error deleteFile")
			}

			if tc.gracePeriod != 0 && tc.expected {
				if delta < tc.gracePeriod {
					t.Errorf("grace period was not respected, 'waitGracePeriod' returned in %s", delta)
				}
			}
			if got != tc.expected {
				t.Fatalf("expecting '%t' when calling waitGracePeriod, got '%t'", tc.expected, got)
			}
		})
	}
}

func TestFilestreamWaitGracePeriodContextCancelled(t *testing.T) {
	env := newInputTestingEnvironment(t)

	logFile := env.mustWriteToFile("logfile.log", []byte("foo bar"))
	st, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("cannot stat %q: %s", logFile, err)
	}
	offset := st.Size()

	cur := loginp.NewCursorForTest("foo-bar", offset, -1)
	gracePeriod := 500 * time.Millisecond

	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		ID:          t.Name(),
		Cancelation: ctx,
		Logger:      env.testLogger.Logger,
	}

	cancel()
	start := time.Now()
	got, err := waitGracePeriod(
		v2Ctx,
		env.testLogger.Logger,
		cur,
		logFile,
		gracePeriod,
		10*time.Millisecond,
		os.Stat,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expecting context cancelled error, got: %s", err)
	}
	if got != false {
		t.Fatal("expecting false when calling waitGracePeriod because the context is cancelled")
	}
	delta := time.Since(start)
	if delta >= gracePeriod {
		t.Fatal("waitGracePeriod did not return before the grace period")
	}
}

func TestFilestreamWaitGracePeriodStatError(t *testing.T) {
	env := newInputTestingEnvironment(t)

	logFile := env.mustWriteToFile("logfile.log", []byte("foo bar"))
	st, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("cannot stat %q: %s", logFile, err)
	}
	offset := st.Size()

	cur := loginp.NewCursorForTest("foo-bar", offset, -1)

	v2Ctx := v2.Context{
		ID:          t.Name(),
		Cancelation: t.Context(),
		Logger:      env.testLogger.Logger,
	}

	statErr := errors.New("Oops")
	statFn := func(string) (os.FileInfo, error) {
		return nil, statErr
	}

	testCases := map[string]struct {
		gracePeriod  time.Duration
		scanInterval time.Duration
	}{
		"stat returns error while waiting grace period": {
			gracePeriod:  time.Second,
			scanInterval: time.Millisecond,
		},
		"stat returns error after grace period": {
			gracePeriod:  time.Millisecond,
			scanInterval: time.Second,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			canDelete, err := waitGracePeriod(
				v2Ctx,
				env.testLogger.Logger,
				cur,
				logFile,
				tc.gracePeriod,
				tc.scanInterval,
				statFn,
			)
			errMsg := fmt.Sprintf("cannot stat '%s': %s", logFile, statErr)
			if errMsg != err.Error() {
				t.Fatalf("expecting error message %q, got %q", errMsg, err.Error())
			}

			if canDelete {
				t.Fatal("waitGracePeriod must return false")
			}
		})
	}
}

func requireFileDeleted(t *testing.T, path string, expectDeleted bool) {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !expectDeleted {
				t.Fatalf("%q was deleted", path)
			}
			return
		}

		t.Fatalf("cannot stat file: %s", err)
	}

	if expectDeleted {
		t.Fatalf("%q was not deleted", path)
	}
}
