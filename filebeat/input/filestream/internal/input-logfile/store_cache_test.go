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

package input_logfile

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestStoreCache_AcquireHit(t *testing.T) {
	resetStoreCacheForTest()
	t.Cleanup(resetStoreCacheForTest)

	states := createSampleStore(t, nil).WithGCPeriod(time.Minute)
	first, err := acquireStore(logptest.NewTestingLogger(t, ""), states, "filestream")
	require.NoError(t, err)
	second, err := acquireStore(logptest.NewTestingLogger(t, ""), states, "filestream")
	require.NoError(t, err)
	require.Same(t, first, second)

	globalStoreCache.mu.Lock()
	entry := globalStoreCache.entries[states.StoreKey()]
	require.NotNil(t, entry)
	require.Equal(t, storeActive, entry.state)
	require.Equal(t, 2, entry.users)
	require.Same(t, first, entry.store)
	require.Len(t, globalStoreCache.entries, 1)
	globalStoreCache.mu.Unlock()

	releaseAcquiredStore(first)
	releaseAcquiredStore(second)
}

func TestStoreCache_LastReleaseDrainsStore(t *testing.T) {
	resetStoreCacheForTest()
	t.Cleanup(resetStoreCacheForTest)

	closeStarted := make(chan struct{})
	allowClose := make(chan struct{})
	cleanup := closeStoreWith(func(s *store) {
		close(closeStarted)
		<-allowClose
		s.close()
	})
	t.Cleanup(cleanup)

	states := createSampleStore(t, nil).WithGCPeriod(time.Hour)
	first, err := acquireStore(logptest.NewTestingLogger(t, ""), states, "filestream")
	require.NoError(t, err)
	second, err := acquireStore(logptest.NewTestingLogger(t, ""), states, "filestream")
	require.NoError(t, err)

	releaseAcquiredStore(first)
	globalStoreCache.mu.Lock()
	entry := globalStoreCache.entries[states.StoreKey()]
	require.NotNil(t, entry)
	require.Equal(t, storeActive, entry.state)
	require.Equal(t, 1, entry.users)
	globalStoreCache.mu.Unlock()

	released := make(chan struct{})
	go func() {
		releaseAcquiredStore(second)
		close(released)
	}()
	<-closeStarted

	globalStoreCache.mu.Lock()
	entry = globalStoreCache.entries[states.StoreKey()]
	require.NotNil(t, entry)
	require.Equal(t, storeDraining, entry.state)
	require.Zero(t, entry.users)
	globalStoreCache.mu.Unlock()

	close(allowClose)
	<-released
}

func TestStoreCache_AcquireWaitsForDrainingStore(t *testing.T) {
	resetStoreCacheForTest()
	t.Cleanup(resetStoreCacheForTest)

	states := newCountingStateStore()
	first, err := acquireStore(logptest.NewTestingLogger(t, ""), states, "filestream")
	require.NoError(t, err)
	first.Retain() // Hold a short-lived getRetainedStore-style reference.
	releaseAcquiredStore(first)

	globalStoreCache.mu.Lock()
	entry := globalStoreCache.entries[states.StoreKey()]
	require.NotNil(t, entry)
	require.Equal(t, storeDraining, entry.state)
	globalStoreCache.mu.Unlock()

	type acquireResult struct {
		store *store
		err   error
	}
	result := make(chan acquireResult, 1)
	go func() {
		s, err := acquireStore(logptest.NewTestingLogger(t, ""), states, "filestream")
		result <- acquireResult{store: s, err: err}
	}()

	select {
	case acquired := <-result:
		t.Fatalf("acquire completed while the old store was still retained: %+v", acquired)
	case <-time.After(100 * time.Millisecond):
	}
	require.Equal(t, int32(1), states.storeForCalls.Load())

	first.Release()
	acquired := <-result
	require.NoError(t, acquired.err)
	require.Equal(t, int32(2), states.storeForCalls.Load())
	require.NotEqual(t, first, acquired.store, "new store must be different from the first one")
	releaseAcquiredStore(acquired.store)
}

func TestStoreCache_LastReferenceClosesStoreOnce(t *testing.T) {
	resetStoreCacheForTest()
	t.Cleanup(resetStoreCacheForTest)

	var closes atomic.Int32
	cleanup := closeStoreWith(func(s *store) {
		closes.Add(1)
		s.close()
	})
	t.Cleanup(cleanup)

	states := newCountingStateStore()
	acquired, err := acquireStore(logptest.NewTestingLogger(t, ""), states, "filestream")
	require.NoError(t, err)
	// Model getRetainedStore ownership. A premature cache close while this
	// reference exists would leave an active caller using a closed store.
	acquired.Retain()

	// This drops manager ownership and stops the cleaner, which drops cache
	// ownership. Forgetting the latter leaks the implicit RefCount owner and
	// prevents the store from ever closing.
	releaseAcquiredStore(acquired)
	require.Zero(t, closes.Load(), "the short-lived reference keeps the store open")

	// The final short-lived reference must be sufficient to close the store;
	// more than one close would indicate duplicate release/close handling.
	acquired.Release()
	require.Equal(t, int32(1), closes.Load())
}

type countingStateStore struct {
	registry      *statestore.Registry
	storeForCalls atomic.Int32
}

func newCountingStateStore() *countingStateStore {
	return &countingStateStore{registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend())}
}

func (s *countingStateStore) StoreKey() string { return "counting-state-store" }

func (s *countingStateStore) StoreFor(name string) (*statestore.Store, error) {
	s.storeForCalls.Add(1)
	return s.registry.Get(name)
}

func (*countingStateStore) CleanupInterval() time.Duration { return time.Minute }

func resetStoreCacheForTest() {
	globalStoreCache.mu.Lock()
	entries := globalStoreCache.entries
	globalStoreCache.entries = make(map[string]*storeCacheEntry)
	globalStoreCache.mu.Unlock()
	for _, entry := range entries {
		if entry.state == storeInitializing {
			entry.initErr = errors.New("store cache reset")
			entry.closeReady()
			entry.closeClosed()
			continue
		}
		if entry.state == storeActive {
			entry.cancel()
			entry.cleanerWg.Wait()
			entry.store.Release()
		}
	}
}
