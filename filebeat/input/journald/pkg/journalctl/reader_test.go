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

//go:build linux

package journalctl

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

//go:embed testdata/corner-cases.json
var coredumpJSON []byte

// TestEventWithNonStringData ensures the Reader can read data that is not a
// string. There is at least one real example of that: coredumps.
// This test uses a real example captured from journalctl -o json.

// If needed more test cases can be added in the future
func TestEventWithNonStringData(t *testing.T) {
	testCases := []json.RawMessage{}
	if err := json.Unmarshal(coredumpJSON, &testCases); err != nil {
		t.Fatalf("could not unmarshal the contents from 'testdata/message-byte-array.json' into map[string]any: %s", err)
	}

	for idx, rawEvent := range testCases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			mock := JctlMock{
				NextFunc: func(canceler input.Canceler) ([]byte, error) {
					return rawEvent, nil
				},
			}
			r := Reader{
				logger: logp.L(),
				jctl:   &mock,
			}

			_, err := r.Next(context.Background())
			if err != nil {
				t.Fatalf("did not expect an error: %s", err)
			}
		})
	}
}

//go:embed testdata/sample-journal-event.json
var jdEvent []byte

func TestRestartsJournalctlOnError(t *testing.T) {
	logger, observedLogs := logptest.NewTestingLoggerWithObserver(t, "")
	ctx := context.Background()

	mock := JctlMock{
		NextFunc: func(canceler input.Canceler) ([]byte, error) {
			return jdEvent, errors.New("journalctl exited with code 42")
		},
	}

	factoryCalls := atomic.Uint32{}
	factory := func(canceller input.Canceler, logger *logp.Logger, args ...string) (Jctl, error) {
		factoryCalls.Add(1)
		// Add a log to make debugging easier and better mimic the behaviour of the real factory/journalctl
		logger.Debugf("starting new mock journalclt ID: %d", factoryCalls.Load())
		// If no calls to next have been made, return a mock
		// that will fail every time Next is called
		if len(mock.NextCalls()) == 0 {
			return &mock, nil
		}

		// If calls have been made, change the Next function to always succeed
		// and return it
		mock.NextFunc = func(canceler input.Canceler) ([]byte, error) {
			return jdEvent, nil
		}

		return &mock, nil
	}

	reader, err := New(logger, ctx, nil, nil, nil, journalfield.IncludeMatches{}, []int{}, SeekHead, "", 0, "", false, factory)
	if err != nil {
		t.Fatalf("cannot instantiate journalctl reader: %s", err)
	}

	isEntryEmpty := func(entry JournalEntry) bool {
		return len(entry.Fields) == 0 && entry.Cursor == "" && entry.MonotonicTimestamp == 0 && entry.RealtimeTimestamp == 0
	}

	// In the first call the mock will return an error, simulating journalctl crashing
	// the reader must handle it and only return the next valid entry and no error
	entry, err := reader.Next(ctx)
	if err != nil {
		t.Fatalf("expecting no error, got: %s", err)
	}
	if isEntryEmpty(entry) {
		t.Fatal("the first call to Next cannot return an empty entry")
	}

	// We need to assert the reader correctly handled the "crash" from journalctl
	// so we look for the log messages, there should be exactly 3:
	//  - First journalctl start
	//  - an error with the exit code 42
	//  - the second journalctl start
	// The exact log messages are:
	//  - starting new mock journalclt ID: 1
	//  - reader error: 'journalctl exited with code 42', restarting...
	//  - starting new mock journalclt ID: 2

	logs := observedLogs.TakeAll()
	if len(logs) != 3 {
		t.Fatalf("expecting 3 log lines from 'input.journald.reader.journalctl-runner', got %d", len(logs))
	}

	if logs[0].Message != "starting new mock journalclt ID: 1" {
		t.Fatalf("first log message must be the mock starting wit ID 1, got '%s' instead", logs[0].Message)
	}

	if logs[1].Message != "reader error: 'journalctl exited with code 42', restarting..." {
		t.Fatalf("second log message must reader error with journalctl exit code 42, got '%s' instead", logs[1].Message)
	}

	if logs[2].Message != "starting new mock journalclt ID: 2" {
		t.Fatalf("third log message must be the mock starting wit ID 2, got '%s' instead", logs[2].Message)
	}

	// Call Next a couple more times to ensure we can read past the error
	for i := 0; i < 2; i++ {
		entry, err := reader.Next(ctx)
		if err != nil {
			t.Fatalf("did not expect an error when calling Next 'after journalctl restart': %s", err)
		}

		if isEntryEmpty(entry) {
			t.Fatal("the second and third calls to Next must succeed")
		}
	}
}

func TestNewUsesMergeFlag(t *testing.T) {
	f := func(_ input.Canceler, _ *logp.Logger, s ...string) (Jctl, error) {
		return nil, nil
	}
	r, err := New(
		logp.NewNopLogger(),
		t.Context(),
		nil,
		nil,
		nil,
		journalfield.IncludeMatches{},
		nil,
		SeekHead,
		"",
		0,
		"",
		true,
		f)

	if err != nil {
		t.Fatalf("did not expect an error when calling New: %s", err)
	}

	if r == nil {
		t.Fatal("the returned reader cannot be nil")
	}

	if !slices.Contains(r.args, "--merge") {
		t.Fatalf("did not find '--merge' in the arguments to journalctl. Args: %s", r.args)
	}
}
