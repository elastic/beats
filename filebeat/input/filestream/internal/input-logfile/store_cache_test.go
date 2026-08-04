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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestStoreCache_AcquireHit(t *testing.T) {
	setupStoreCacheTest(t)

	logger, logs := newObserverLogger(t)
	registry := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
	firstStates := newCountingStateStoreWithRegistry("shared-backend", registry)
	secondStates := newCountingStateStoreWithRegistry("shared-backend", registry)

	first, err := acquireStore(logger, firstStates, "filestream")
	require.NoError(t, err)
	second, err := acquireStore(logger, secondStates, "filestream")
	require.NoError(t, err)
	require.Same(t, first, second)
	require.Equal(t, int32(1), firstStates.storeForCalls.Load())
	require.Zero(t, secondStates.storeForCalls.Load())

	entry := snapshotStoreCacheEntry(firstStates.StoreKey())
	require.True(t, entry.found)
	require.Equal(t, storeActive, entry.state)
	require.Equal(t, 2, entry.users)
	require.Same(t, first, entry.store)
	require.Equal(t, 1, storeCacheEntryCount())
	require.Eventually(t, func() bool {
		return logs.FilterMessage("filestream shared store cleaner started").Len() == 1
	}, time.Second, time.Millisecond)

	releaseAcquiredStore(logger, first)
	releaseAcquiredStore(logger, second)
}

func TestStoreCache_LastReleaseDrainsStore(t *testing.T) {
	setupStoreCacheTest(t)

	closeStarted := make(chan struct{})
	allowClose := make(chan struct{})
	cleanup := closeStoreWith(func(s *store) {
		close(closeStarted)
		<-allowClose
		s.close()
	})
	t.Cleanup(cleanup)

	logger := logp.NewNopLogger()
	states := createSampleStore(t, nil).WithGCPeriod(time.Hour)
	first, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)
	second, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)

	releaseAcquiredStore(logger, first)
	entry := snapshotStoreCacheEntry(states.StoreKey())
	require.True(t, entry.found)
	require.Equal(t, storeActive, entry.state)
	require.Equal(t, 1, entry.users)

	released := make(chan struct{})
	go func() {
		releaseAcquiredStore(logger, second)
		close(released)
	}()
	<-closeStarted

	entry = snapshotStoreCacheEntry(states.StoreKey())
	require.True(t, entry.found)
	require.Equal(t, storeDraining, entry.state)
	require.Zero(t, entry.users)

	close(allowClose)
	<-released
}

func TestStoreCache_AcquireWaitsForDrainingStore(t *testing.T) {
	setupStoreCacheTest(t)

	logger, logs := newObserverLogger(t)

	states := newCountingStateStore("draining-backend")
	first, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)
	first.Retain() // Hold a short-lived getRetainedStore-style reference.
	releaseAcquiredStore(logger, first)

	entry := snapshotStoreCacheEntry(states.StoreKey())
	require.True(t, entry.found)
	require.Equal(t, storeDraining, entry.state)

	type acquireResult struct {
		store *store
		err   error
	}
	result := make(chan acquireResult, 1)
	go func() {
		s, err := acquireStore(logger, states, "filestream")
		result <- acquireResult{store: s, err: err}
	}()

	// acquireStore logs this immediately before waiting for the draining store
	// to close. The event is therefore a deterministic barrier that the second
	// acquisition cannot complete until the retained reference is released.
	require.Eventually(t, func() bool {
		return logs.FilterMessage("waiting for draining filestream shared store").Len() == 1
	}, time.Second, time.Millisecond)
	require.Equal(t, int32(1), states.storeForCalls.Load())

	first.Release()
	var acquired acquireResult
	select {
	case acquired = <-result:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for acquisition after releasing the draining store")
	}
	require.NoError(t, acquired.err)
	require.Equal(t, int32(2), states.storeForCalls.Load())
	require.NotSame(t, first, acquired.store, "new store must be different from the first one")
	releaseAcquiredStore(logger, acquired.store)
}

func TestStoreCache_LastReferenceClosesStoreOnce(t *testing.T) {
	setupStoreCacheTest(t)

	var closes atomic.Int32
	cleanup := closeStoreWith(func(s *store) {
		closes.Add(1)
		s.close()
	})
	t.Cleanup(cleanup)

	states := newCountingStateStore("last-reference-backend")
	logger := logp.NewNopLogger()
	acquired, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)
	// Model getRetainedStore ownership. A premature cache close while this
	// reference exists would leave an active caller using a closed store.
	acquired.Retain()

	// This drops manager ownership and stops the cleaner, which drops cache
	// ownership. Forgetting the latter leaks the implicit RefCount owner and
	// prevents the store from ever closing.
	releaseAcquiredStore(logger, acquired)
	require.Zero(t, closes.Load(), "the short-lived reference keeps the store open")

	// The final short-lived reference must be sufficient to close the store;
	// more than one close would indicate duplicate release/close handling.
	acquired.Release()
	require.Equal(t, int32(1), closes.Load())
}

func TestStoreCache_ConcurrentInitialization(t *testing.T) {
	setupStoreCacheTest(t)

	logger, logs := newObserverLogger(t)

	states := newBlockingStateStore("concurrent-initialization-backend", nil)
	const acquisitions = 10
	const waiters = acquisitions - 1
	type acquireResult struct {
		store *store
		err   error
	}
	results := make(chan acquireResult, acquisitions)

	// The first caller owns initialization and remains in StoreFor until the
	// remaining callers have observed the initializing cache entry.
	go func() {
		s, err := acquireStore(logger, states, "filestream")
		results <- acquireResult{store: s, err: err}
	}()
	<-states.firstStoreForStarted

	for range waiters {
		go func() {
			s, err := acquireStore(logger, states, "filestream")
			results <- acquireResult{store: s, err: err}
		}()
	}
	// The event is logged immediately before each caller waits for the
	// initializing entry. This proves the test is exercising that path.
	require.Eventually(t, func() bool {
		return logs.FilterMessage("waiting for filestream shared store initialization").Len() == waiters
	}, time.Second, time.Millisecond)

	close(states.releaseFirstStoreFor)

	var first *store
	acquiredStores := make([]*store, 0, acquisitions)
	for range acquisitions {
		result := <-results
		require.NoError(t, result.err)
		if first == nil {
			first = result.store
		} else {
			require.Same(t, first, result.store)
		}
		acquiredStores = append(acquiredStores, result.store)
	}
	for _, acquired := range acquiredStores {
		releaseAcquiredStore(logger, acquired)
	}
	require.Equal(t, int32(1), states.storeForCalls.Load())
}

func TestStoreCache_InitializationFailureCanRetry(t *testing.T) {
	setupStoreCacheTest(t)

	logger, logs := newObserverLogger(t)

	states := newBlockingStateStore("failing-once-backend", errTestStoreInitialization)
	const waiters = 4
	type acquireResult struct{ err error }
	results := make(chan acquireResult, waiters+1)
	// This first acquisition owns initialization and blocks in StoreFor until
	// the test chooses whether initialisation can continue and fails.
	go func() {
		_, err := acquireStore(logger, states, "filestream")
		results <- acquireResult{err: err}
	}()
	<-states.firstStoreForStarted

	// These acquisitions find the initializing cache entry. They must wait for
	// the first attempt instead of opening competing stores of their own.
	for range waiters {
		go func() {
			_, err := acquireStore(logger, states, "filestream")
			results <- acquireResult{err: err}
		}()
	}
	// acquireStore logs this immediately before blocking on entry.ready. Using
	// that existing lifecycle event as the barrier proves these are current
	// waiters without adding production-only synchronization state to the cache.
	require.Eventually(t, func() bool {
		return logs.FilterMessage("waiting for filestream shared store initialization").Len() == waiters
	}, time.Second, time.Millisecond)

	// Failing the initializer wakes every current waiter with the same error;
	// nobody receives a store from this failed cache entry.
	close(states.releaseFirstStoreFor)
	for range waiters + 1 {
		require.ErrorIs(t, (<-results).err, errTestStoreInitialization)
	}
	require.Equal(t, int32(1), states.storeForCalls.Load())

	// The failed placeholder must be gone, so a later acquisition can create a
	// new entry and successfully open a replacement store.
	retried, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)
	require.Equal(t, int32(2), states.storeForCalls.Load())
	releaseAcquiredStore(logger, retried)
}

func TestStoreCache_DifferentBackendsAreIsolated(t *testing.T) {
	setupStoreCacheTest(t)

	logger, logs := newObserverLogger(t)
	firstStates := newCountingStateStore("first-backend")
	secondStates := newCountingStateStore("second-backend")

	first, err := acquireStore(logger, firstStates, "filestream")
	require.NoError(t, err)
	second, err := acquireStore(logger, secondStates, "filestream")
	require.NoError(t, err)
	require.NotSame(t, first, second)

	require.Equal(t, 2, storeCacheEntryCount())
	require.True(t, snapshotStoreCacheEntry(firstStates.StoreKey()).found)
	require.True(t, snapshotStoreCacheEntry(secondStates.StoreKey()).found)
	require.Equal(t, int32(1), firstStates.storeForCalls.Load())
	require.Equal(t, int32(1), secondStates.storeForCalls.Load())
	require.Eventually(t, func() bool {
		return logs.FilterMessage("filestream shared store cleaner started").Len() == 2
	}, time.Second, time.Millisecond)

	releaseAcquiredStore(logger, first)
	releaseAcquiredStore(logger, second)
}

func TestStoreCache_ConcurrentAcquireRelease(t *testing.T) {
	setupStoreCacheTest(t)

	states := newCountingStateStore("concurrent-acquire-release-backend")
	logger := logp.NewNopLogger()
	const workers = 10
	const iterations = 20
	errs := make(chan error, workers*iterations)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			<-start
			for range iterations {
				s, err := acquireStore(logger, states, "filestream")
				if err != nil {
					errs <- err
					continue
				}
				releaseAcquiredStore(logger, s)
			}
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	require.False(t, snapshotStoreCacheEntry(states.StoreKey()).found)
}

type countingStateStore struct {
	registry      *statestore.Registry
	storeForCalls atomic.Int32
	key           string
}

type cacheEntrySnapshot struct {
	found bool
	state storeCacheState
	users int
	store *store
}

func snapshotStoreCacheEntry(key string) cacheEntrySnapshot {
	globalStoreCache.mu.Lock()
	defer globalStoreCache.mu.Unlock()

	entry := globalStoreCache.entries[key]
	if entry == nil {
		return cacheEntrySnapshot{}
	}
	return cacheEntrySnapshot{
		found: true,
		state: entry.state,
		users: entry.users,
		store: entry.store,
	}
}

func storeCacheEntryCount() int {
	globalStoreCache.mu.Lock()
	defer globalStoreCache.mu.Unlock()
	return len(globalStoreCache.entries)
}

func newObserverLogger(t *testing.T) (*logp.Logger, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	logger, err := logp.NewZapLogger(zap.New(core))
	require.NoError(t, err)
	return logger, logs
}

func newCountingStateStore(key string) *countingStateStore {
	return newCountingStateStoreWithRegistry(key, statestore.NewRegistry(storetest.NewMemoryStoreBackend()))
}

func newCountingStateStoreWithRegistry(key string, registry *statestore.Registry) *countingStateStore {
	return &countingStateStore{registry: registry, key: key}
}

func (s *countingStateStore) StoreKey() string { return s.key }

func (s *countingStateStore) StoreFor(name string) (*statestore.Store, error) {
	s.storeForCalls.Add(1)
	return s.registry.Get(name)
}

func (*countingStateStore) CleanupInterval() time.Duration { return time.Minute }

var errTestStoreInitialization = errors.New("test store initialization failed")

type blockingStateStore struct {
	*countingStateStore
	firstStoreForStarted chan struct{}
	releaseFirstStoreFor chan struct{}
	firstStoreForError   error
}

func newBlockingStateStore(key string, firstStoreForError error) *blockingStateStore {
	return &blockingStateStore{
		countingStateStore:   newCountingStateStore(key),
		firstStoreForStarted: make(chan struct{}),
		releaseFirstStoreFor: make(chan struct{}),
		firstStoreForError:   firstStoreForError,
	}
}

func (s *blockingStateStore) StoreFor(name string) (*statestore.Store, error) {
	if s.storeForCalls.Add(1) == 1 {
		close(s.firstStoreForStarted)
		<-s.releaseFirstStoreFor
		if s.firstStoreForError != nil {
			return nil, s.firstStoreForError
		}
	}
	return s.registry.Get(name)
}

func setupStoreCacheTest(t *testing.T) {
	t.Helper()
	resetStoreCacheForTest()
	t.Cleanup(func() {
		require.Zero(t, storeCacheEntryCount(), "all acquired store references must be released")
	})
}

// resetStoreCacheForTest provides isolation before a test starts. Tests must
// release all references themselves.
// This helper never releases a store reference or alters an entry lifecycle;
// cleanup detects ownership leaks instead.
func resetStoreCacheForTest() {
	globalStoreCache.mu.Lock()
	defer globalStoreCache.mu.Unlock()
	globalStoreCache.entries = make(map[string]*storeCacheEntry)
}
