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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestRegistrarNotStarted covers the Beat that configures no log-family input —
// filestream only, say. Nothing calls Start, so the registrar must cost nothing:
// no goroutine, no registry scan, and no statestore handle left open once the
// Beat shuts down.
func TestRegistrarNotStarted(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()

	memBackend, r, _ := newLazyTestRegistrar(t, file.State{Id: "on-disk", Source: "/a.log", TTL: -1})

	assert.Empty(t, r.GetStates(),
		"a registrar that was never started must not have scanned the registry")

	r.Stop()

	assert.True(t, memBackend.Stores[testStoreName].IsClosed(),
		"Stop must close the store when Run never ran to close it")
	requireNoLeakedGoroutines(t, goroutines)
}

// TestRegistrarStartIsIdempotent covers the trigger: every log-family input
// calls Start as it is created, and only the first may run the registrar.
func TestRegistrarStartIsIdempotent(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()

	memBackend, r, logs := newLazyTestRegistrar(t, file.State{Id: "on-disk", Source: "/a.log", TTL: -1})

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { assert.NoError(t, r.Start()) })
	}
	wg.Wait()

	assert.Len(t, r.GetStates(), 1, "the registry states must be loaded")
	// SetStates replaces the states, so the log line is what counts the loads.
	assert.Equal(t, 1, logs.FilterMessageSnippet("States Loaded from registrar").Len(),
		"the registry must be loaded exactly once")

	r.Stop()

	assert.True(t, memBackend.Stores[testStoreName].IsClosed(), "Run must close the store")
	requireNoLeakedGoroutines(t, goroutines)
}

// TestRegistrarStartAfterStop pins that Stop claims the start, so an input
// created concurrently with shutdown cannot leave a registrar running behind a
// Stop that has already returned.
func TestRegistrarStartAfterStop(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()

	_, r, logs := newLazyTestRegistrar(t, file.State{Id: "on-disk", Source: "/a.log", TTL: -1})
	r.Stop()

	require.ErrorIs(t, r.Start(), ErrStopped, "a Start after Stop must report that nothing was started")
	assert.Empty(t, r.GetStates(), "a Start after Stop must not load the registry")
	assert.Zero(t, logs.FilterMessageSnippet("States Loaded from registrar").Len(),
		"a Start after Stop must not load the registry")

	requireNoLeakedGoroutines(t, goroutines)
}

// TestRegistrarStartErrorIsSticky pins that a failed start is reported to every
// caller. Otherwise an input created on a later config reload would get a
// registrar nothing reads from and block publishing states.
func TestRegistrarStartErrorIsSticky(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()

	memBackend, r, _ := newLazyTestRegistrar(t, file.State{Id: "on-disk", Source: "/a.log", TTL: -1})
	// Closing the store makes loading the states fail.
	require.NoError(t, memBackend.Stores[testStoreName].Close())

	first := r.Start()
	require.Error(t, first, "Start must fail when the states cannot be loaded")
	assert.EqualError(t, r.Start(), first.Error(), "every later Start must report the same failure")

	r.Stop()
	assert.EqualError(t, r.Start(), first.Error(), "Stop must not replace the original start failure")
	requireNoLeakedGoroutines(t, goroutines)
}

// TestRegistrarStopIsIdempotent pins that Stop can be called more than once.
func TestRegistrarStopIsIdempotent(t *testing.T) {
	_, r, _ := newLazyTestRegistrar(t)
	require.NoError(t, r.Start())

	r.Stop()
	assert.NotPanics(t, r.Stop, "a second Stop must be a no-op")
}

// TestRegistrarStartedStillPersists is the other half of TestRegistrarNotStarted:
// once an input starts it, the registrar behaves exactly as it did when it was
// started unconditionally.
func TestRegistrarStartedStillPersists(t *testing.T) {
	memBackend, r, _ := newLazyTestRegistrar(t)

	require.NoError(t, r.Start())

	state := file.State{Id: "published", Source: "/b.log", TTL: -1}
	r.Channel <- []file.State{state}
	r.Stop()

	assert.Contains(t, memBackend.Stores[testStoreName].Table, fileStatePrefix+state.Id,
		"a started registrar must persist the states it is sent")
}

// requireNoLeakedGoroutines fails the test if the goroutine count does not
// return to what it was before, which is the whole point of not starting a
// registrar nothing needs.
func requireNoLeakedGoroutines(t *testing.T, c *resources.GoroutinesChecker) {
	t.Helper()

	_, err := c.WaitUntilOriginalCount()
	require.NoError(t, err, "the registrar must not leave a goroutine behind")
}

// newLazyTestRegistrar returns a registrar over a memory-backed store
// pre-populated with states, so a test can tell whether the registry was
// scanned. The registrar is not started. The returned logs are the registrar's.
func newLazyTestRegistrar(t *testing.T, states ...file.State) (*storetest.MemoryStore, *Registrar, *observer.ObservedLogs) {
	t.Helper()

	memBackend := storetest.NewMemoryStoreBackend()
	stateStore := &testStateStore{registry: statestore.NewRegistry(memBackend)}

	if len(states) > 0 {
		store, err := stateStore.StoreFor("", "")
		require.NoError(t, err)
		require.NoError(t, writeStates(store, states))
		store.Close()
	}

	logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
	r, err := New(stateStore, &spyLogger{}, time.Second, logger)
	require.NoError(t, err)
	return memBackend, r, logs
}
