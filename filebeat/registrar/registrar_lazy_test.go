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

	memBackend, r := newLazyTestRegistrar(t, file.State{Id: "on-disk", Source: "/a.log", TTL: -1})

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

	memBackend, r := newLazyTestRegistrar(t, file.State{Id: "on-disk", Source: "/a.log", TTL: -1})

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { assert.NoError(t, r.Start()) })
	}
	wg.Wait()

	assert.Len(t, r.GetStates(), 1, "the registry must be loaded exactly once")

	r.Stop()

	assert.True(t, memBackend.Stores[testStoreName].IsClosed(), "Run must close the store")
	requireNoLeakedGoroutines(t, goroutines)
}

// TestRegistrarStartAfterStop pins that Stop claims the start, so an input
// created concurrently with shutdown cannot leave a registrar running behind a
// Stop that has already returned.
func TestRegistrarStartAfterStop(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()

	_, r := newLazyTestRegistrar(t)
	r.Stop()

	require.NoError(t, r.Start(), "a Start after Stop must be a no-op, not an error")
	assert.Empty(t, r.GetStates(), "a Start after Stop must not load the registry")

	requireNoLeakedGoroutines(t, goroutines)
}

// TestRegistrarStartedStillPersists is the other half of TestRegistrarNotStarted:
// once an input starts it, the registrar behaves exactly as it did when it was
// started unconditionally.
func TestRegistrarStartedStillPersists(t *testing.T) {
	memBackend, r := newLazyTestRegistrar(t)

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
// scanned. The registrar is not started.
func newLazyTestRegistrar(t *testing.T, states ...file.State) (*storetest.MemoryStore, *Registrar) {
	t.Helper()

	memBackend := storetest.NewMemoryStoreBackend()
	stateStore := &testStateStore{registry: statestore.NewRegistry(memBackend)}

	if len(states) > 0 {
		store, err := stateStore.StoreFor("")
		require.NoError(t, err)
		require.NoError(t, writeStates(store, states))
		store.Close()
	}

	r, err := New(stateStore, &spyLogger{}, time.Second, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	return memBackend, r
}
