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

package registrar

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type spyLogger struct {
	n int
}

func (s *spyLogger) Published(n int) bool {
	s.n += n
	return true
}

const testStoreName = "test"

type testStateStore struct {
	registry *statestore.Registry
}

func (s *testStateStore) StoreFor(string) (*statestore.Store, error) {
	return s.registry.Get(testStoreName)
}

func (s *testStateStore) CleanupInterval() time.Duration {
	return time.Second
}

// TestRunDrainsPendingBatchOnShutdown verifies the shutdown property: a state
// already in the channel when done fires is persisted and acked, so it is not
// replayed on restart.
func TestRunDrainsPendingBatchOnShutdown(t *testing.T) {
	testCases := []struct {
		name    string
		timeout time.Duration
	}{
		{"directIn path", 0},
		{"collectIn path", time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := logptest.NewTestingLogger(t, "")
			memBackend := storetest.NewMemoryStoreBackend()
			stateStore := &testStateStore{registry: statestore.NewRegistry(memBackend)}

			spy := &spyLogger{}
			r, err := New(stateStore, spy, tc.timeout, logger)
			if err != nil {
				t.Fatal(err)
			}

			state := file.State{Id: "test-id", Source: "/path/to/file.log", TTL: -1}

			r.Channel <- []file.State{state}
			close(r.done)

			r.Run()

			table := memBackend.Stores[testStoreName].Table
			if _, ok := table[fileStatePrefix+state.Id]; !ok {
				t.Fatalf("expected drained state %q to be persisted, store has: %v", state.Id, table)
			}

			if spy.n != 1 {
				t.Fatalf("expected commitStateUpdates to ack the single drained state once, got Published count %d", spy.n)
			}
		})
	}
}

// TestRunEmptyChannelShutdown verifies that Run returns without deadlock when
// done fires and the channel is empty. This exercises the default branch of the
// drain select.
func TestRunEmptyChannelShutdown(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	memBackend := storetest.NewMemoryStoreBackend()
	stateStore := &testStateStore{registry: statestore.NewRegistry(memBackend)}

	spy := &spyLogger{}
	r, err := New(stateStore, spy, 0, logger)
	if err != nil {
		t.Fatal(err)
	}

	close(r.done)

	r.Run()

	if spy.n != 0 {
		t.Fatalf("expected no states to be acked, got Published count %d", spy.n)
	}
}

// TestRunConcurrentShutdownAndBatch runs Run in a background goroutine (via
// Start), pre-loads a state into the channel, then calls Stop immediately.
// Many iterations expose the scheduling race: the goroutine may not be
// scheduled before r.done closes, so the select sees both channels ready
// simultaneously — the exact case the shutdown drain must handle.
func TestRunConcurrentShutdownAndBatch(t *testing.T) {
	const iterations = 100

	testCases := []struct {
		name    string
		timeout time.Duration
	}{
		{"directIn path", 0},
		{"collectIn path", time.Second},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < iterations; i++ {
				logger := logptest.NewTestingLogger(t, "")
				memBackend := storetest.NewMemoryStoreBackend()
				stateStore := &testStateStore{registry: statestore.NewRegistry(memBackend)}
				spy := &spyLogger{}
				r, err := New(stateStore, spy, tc.timeout, logger)
				if err != nil {
					t.Fatalf("iter %d: New: %v", i, err)
				}

				state := file.State{Id: "test-id", Source: "/path/to/file.log", TTL: -1}

				// Pre-load the buffered channel before Run starts so that both
				// r.Channel and r.done can be ready in the first select evaluation.
				r.Channel <- []file.State{state}

				if err := r.Start(); err != nil {
					t.Fatalf("iter %d: Start: %v", i, err)
				}
				// Immediately stop — races with the pre-loaded batch.
				r.Stop()

				if _, ok := memBackend.Stores[testStoreName].Table[fileStatePrefix+state.Id]; !ok {
					t.Fatalf("iter %d (%s): state %q was not persisted", i, tc.name, state.Id)
				}
			}
		})
	}
}

var _ statestore.States = (*testStateStore)(nil)
