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
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
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
	}{
		"happy path": {
			cursorPending:     -1,
			expectFileDeleted: true,
		},
		"pending events": {
			cursorPending:     2,
			expectFileDeleted: false,
		},
	}

	for name, tc := range testCases {
		env := newInputTestingEnvironment(t)

		f := filestream{
			deleterConfig: deleterConfig{
				retryBackoff: 10 * time.Millisecond,
			},
			scannerCheckInterval: 10 * time.Millisecond,
		}

		data := []byte("foo bar\n")
		logFile := env.mustWriteToFile("logfile.log", data)
		cur := loginp.NewCursorForTest(t.Name()+":"+logFile, int64(len(data)), tc.cursorPending)

		t.Run(name, func(t *testing.T) {
			v2Ctx := v2.Context{
				ID:          t.Name(),
				Cancelation: t.Context(),
				Logger:      env.logger,
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
			f := filestream{
				scannerCheckInterval: 10 * time.Millisecond,
				deleterConfig: deleterConfig{
					GracePeriod: tc.gracePeriod,
				},
			}
			v2Ctx := v2.Context{
				ID:          t.Name(),
				Cancelation: t.Context(),
				Logger:      env.logger,
			}

			start := time.Now()
			got, err := f.waitGracePeriod(v2Ctx, env.logger, cur, logFile)
			delta := time.Now().Sub(start)
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
