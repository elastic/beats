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

package journalctl

import (
	"context"
	_ "embed"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

//go:embed testdata/corner-cases.json
var coredumpJSON []byte

// TestEventWithNonStringData ensures the Reader can read data that is not a
// string. There is at least one real example of that: coredumps.
// This test uses a real example captured from journalctl -o json.
//
// If needed more test cases can be added in the future
// func TestEventWithNonStringData(t *testing.T) {
// 	t.Skip("TODO: Re-write this test to test the correct type.")
// 	testCases := []json.RawMessage{}
// 	if err := json.Unmarshal(coredumpJSON, &testCases); err != nil {
// 		t.Fatalf("could not unmarshal the contents from 'testdata/message-byte-array.json' into map[string]any: %s", err)
// 	}

// 	for idx, event := range testCases {
// 		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
// 			// stdout := io.NopCloser(&bytes.Buffer{})
// 			// stderr := io.NopCloser(&bytes.Buffer{})
// 			r := Reader{
// 				logger: logp.L(),
// 				// dataChan: make(chan []byte),
// 				// errChan:  make(chan string),
// 				// stdout:   stdout,
// 				// stderr:   stderr,
// 			}

// 			go func() {
// 				r.dataChan <- []byte(event)
// 			}()

// 			_, err := r.Next(context.Background())
// 			if err != nil {
// 				t.Fatalf("did not expect an error: %s", err)
// 			}
// 		})
// 	}
// }

//go:embed testdata/sample-journal-event.json
var jdEvent []byte

func TestRestartsJournalctlOnError(t *testing.T) {
	ctx := context.Background()

	mock := JctlMock{
		NextFunc: func(canceler input.Canceler) ([]byte, error) {
			return []byte(jdEvent), errors.New("journalctl exited with code 42")
		},
	}

	factoryCalls := atomic.Uint32{}
	factory := func(canceller input.Canceler, logger *logp.Logger, binary string, args ...string) (Jctl, error) {
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
			return []byte(jdEvent), nil
		}

		return &mock, nil
	}

	reaqder, err := New(logp.L(), ctx, nil, nil, nil, journalfield.IncludeMatches{}, SeekHead, "", 0, "", factory)
	if err != nil {
		t.Fatalf("cannot instantiate journalctl reader: %s", err)
	}

	isEntryEmpty := func(entry JournalEntry) bool {
		return len(entry.Fields) == 0 && entry.Cursor == "" && entry.MonotonicTimestamp == 0 && entry.RealtimeTimestamp == 0
	}

	// In the first call the mock will return an error, simulating journalctl crashing
	// we expect no error and an empty entry. The input can handle this situation.
	entry, err := reaqder.Next(ctx)
	if err != nil {
		t.Fatalf("did not expect an error when calling Next: %s", err)
	}
	if !isEntryEmpty(entry) {
		t.Fatal("the first call to Next must return an empty JournalEntry because 'journalctl has crashed'")
	}

	for i := 0; i < 2; i++ {
		entry, err := reaqder.Next(ctx)
		if err != nil {
			t.Fatalf("did not expect an error when calling Next 'after journalctl restart': %s", err)
		}

		if isEntryEmpty(entry) {
			t.Fatal("the second and third calls to Next must succeed")
		}
	}
}
