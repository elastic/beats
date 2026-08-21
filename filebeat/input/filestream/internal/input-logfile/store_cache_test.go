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

	"github.com/elastic/beats/v7/filebeat/input/v2/statemanager"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestStoreCache_AcquireHit(t *testing.T) {
	setupCacheForTest(t)

	logger, logs := newObserverLogger(t)
	registry := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
	firstStates := newCountingStateStoreWithRegistry("shared-backend", registry)
	secondStates := newCountingStateStoreWithRegistry("shared-backend", registry)

	first, release1, err := acquireStore(logger, firstStates, "filestream")
	require.NoError(t, err)
	second, release2, err := acquireStore(logger, secondStates, "filestream")
	require.NoError(t, err)
	require.Same(t, first.store, second.store)
	require.Same(t, first.ackCH, second.ackCH)
	require.Equal(t, int32(1), firstStates.storeForCalls.Load())
	require.Zero(t, secondStates.storeForCalls.Load())

	require.Equal(t, 1, globalCache.Len())
	require.Eventually(t, func() bool {
		return logs.FilterMessage("filestream shared store cleaner started").Len() == 1
	}, time.Second, time.Millisecond)

	release1()
	release2()
}

func TestStoreCache_LastReleaseDrainsStore(t *testing.T) {
	setupCacheForTest(t)

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
	first, release1, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)
	_, release2, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)
	require.Equal(t, 1, globalCache.Len())

	release1()
	require.Equal(t, 1, globalCache.Len(), "one user still active; cache entry must remain")

	released := make(chan struct{})
	go func() {
		release2()
		close(released)
	}()
	<-closeStarted
	// The entry has been removed from the map (drain started) but the store
	// close is blocked by closeStoreWith.
	require.Equal(t, 0, globalCache.Len(), "draining entry is no longer in the active map")

	close(allowClose)
	<-released
	_ = first // suppress unused variable
}

// TestStoreCache_NewAcquireOpensFreshEntryDuringDrain verifies the updated
// drain semantics: when the last reference is released the cache entry is
// removed from the map immediately, so a concurrent Acquire for the same key
// does NOT wait for the drain to complete — it opens a brand-new entry right
// away. This differs from the old behaviour, where a new caller would block on
// the draining entry's closed channel until the store's ref-count reached zero.
func TestStoreCache_NewAcquireOpensFreshEntryDuringDrain(t *testing.T) {
	drainStarted := make(chan struct{})
	allowDrain := make(chan struct{})
	var onceBlock sync.Once

	// Block the closeFn on the first (and only first) drain. We install the
	// blocking logic in the cache itself so it fires before ackUpdater.Close()
	// and store.Release(), keeping the test independent of closeStoreWith.
	globalCache = statemanager.NewCache[*logfileEntry](func(e *logfileEntry) {
		onceBlock.Do(func() {
			close(drainStarted)
			<-allowDrain
		})
		e.ackUpdater.Close()
		e.store.Release()
	})
	t.Cleanup(func() {
		require.Zero(t, globalCache.Len(), "all acquired store references must be released")
	})

	states := newCountingStateStore("drain-concurrent-backend")
	logger := logp.NewNopLogger()

	_, release1, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)

	// release1() triggers drain; closeFn blocks until allowDrain is closed.
	drainDone := make(chan struct{})
	go func() {
		release1()
		close(drainDone)
	}()
	<-drainStarted
	// The entry is removed from the map as soon as the last user releases,
	// even though closeFn has not yet returned.
	require.Equal(t, 0, globalCache.Len(), "draining entry must not be in the active map")

	// A new Acquire must not wait for the drain; it must open a fresh entry.
	_, release2, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)
	require.Equal(t, 1, globalCache.Len(), "fresh entry must be in the active map")
	require.Equal(t, int32(2), states.storeForCalls.Load(), "a new store must have been opened for the fresh entry")

	// Unblock the drain, then release the second entry.
	close(allowDrain)
	<-drainDone
	release2()
	require.Equal(t, 0, globalCache.Len())
}

func TestStoreCache_LastReferenceClosesStoreOnce(t *testing.T) {
	setupCacheForTest(t)

	var closes atomic.Int32
	cleanup := closeStoreWith(func(s *store) {
		closes.Add(1)
		s.close()
	})
	t.Cleanup(cleanup)

	states := newCountingStateStore("last-reference-backend")
	logger := logp.NewNopLogger()
	entry, release, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)
	// Model getRetainedStore ownership: an extra Retain held during Create.
	entry.store.Retain()

	// Dropping the cache reference triggers drain, which calls closeFn only
	// after the cleaner goroutine exits. The retained reference prevents the
	// store from being closed until it is explicitly released.
	release()
	require.Zero(t, closes.Load(), "the short-lived reference keeps the store open")

	entry.store.Release()
	require.Equal(t, int32(1), closes.Load(), "the store must close exactly once")
}

func TestStoreCache_ConcurrentInitialization(t *testing.T) {
	setupCacheForTest(t)

	logger, logs := newObserverLogger(t)

	states := newBlockingStateStore("concurrent-initialization-backend", nil)
	const acquisitions = 10
	const waiters = acquisitions - 1
	type acquireResult struct {
		entry   *logfileEntry
		release func()
		err     error
	}
	results := make(chan acquireResult, acquisitions)

	go func() {
		e, rel, err := acquireStore(logger, states, "filestream")
		results <- acquireResult{entry: e, release: rel, err: err}
	}()
	<-states.firstStoreForStarted

	for range waiters {
		go func() {
			e, rel, err := acquireStore(logger, states, "filestream")
			results <- acquireResult{entry: e, release: rel, err: err}
		}()
	}
	require.Eventually(t, func() bool {
		return logs.FilterMessage("waiting for filestream shared store initialization").Len() == waiters
	}, time.Second, time.Millisecond)

	close(states.releaseFirstStoreFor)

	// Collect all results before releasing any reference. Releasing inside the
	// loop would drain the cache entry while goroutines are still waking up from
	// <-ready (between onWait and <-ready), causing them to see a missing entry
	// and open a second store.
	all := make([]acquireResult, 0, acquisitions)
	for range acquisitions {
		r := <-results
		require.NoError(t, r.err)
		all = append(all, r)
	}
	firstEntry := all[0].entry
	for _, r := range all[1:] {
		require.Same(t, firstEntry.store, r.entry.store)
	}
	for _, r := range all {
		r.release()
	}
	require.Equal(t, int32(1), states.storeForCalls.Load())
}

func TestStoreCache_InitializationFailureCanRetry(t *testing.T) {
	setupCacheForTest(t)

	logger, logs := newObserverLogger(t)

	states := newBlockingStateStore("failing-once-backend", errTestStoreInitialization)
	const waiters = 4
	type acquireResult struct{ err error }
	results := make(chan acquireResult, waiters+1)
	go func() {
		_, _, err := acquireStore(logger, states, "filestream")
		results <- acquireResult{err: err}
	}()
	<-states.firstStoreForStarted

	for range waiters {
		go func() {
			_, _, err := acquireStore(logger, states, "filestream")
			results <- acquireResult{err: err}
		}()
	}
	require.Eventually(t, func() bool {
		return logs.FilterMessage("waiting for filestream shared store initialization").Len() == waiters
	}, time.Second, time.Millisecond)

	close(states.releaseFirstStoreFor)
	for range waiters + 1 {
		require.ErrorIs(t, (<-results).err, errTestStoreInitialization)
	}
	require.Equal(t, int32(1), states.storeForCalls.Load())

	retried, release, err := acquireStore(logger, states, "filestream")
	require.NoError(t, err)
	require.Equal(t, int32(2), states.storeForCalls.Load())
	release()
	_ = retried
}

func TestStoreCache_DifferentBackendsInitializeIndependently(t *testing.T) {
	setupCacheForTest(t)

	logger, logs := newObserverLogger(t)
	firstStates := newBlockingStateStore("first-backend", nil)
	secondStates := newCountingStateStore("second-backend")

	type acquireResult struct {
		entry   *logfileEntry
		release func()
		err     error
	}
	firstResult := make(chan acquireResult, 1)
	go func() {
		e, rel, err := acquireStore(logger, firstStates, "filestream")
		firstResult <- acquireResult{entry: e, release: rel, err: err}
	}()
	<-firstStates.firstStoreForStarted

	secondResult := make(chan acquireResult, 1)
	go func() {
		e, rel, err := acquireStore(logger, secondStates, "filestream")
		secondResult <- acquireResult{entry: e, release: rel, err: err}
	}()
	var second *logfileEntry
	var release2 func()
	select {
	case result := <-secondResult:
		require.NoError(t, result.err, "a different backend must initialize while the first backend is blocked")
		second = result.entry
		release2 = result.release
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for a different backend to initialize")
	}
	require.Equal(t, 2, globalCache.Len(), "both backends must have active cache entries")

	close(firstStates.releaseFirstStoreFor)
	result := <-firstResult
	require.NoError(t, result.err)
	first := result.entry
	require.NotSame(t, first.store, second.store)

	require.Equal(t, 2, globalCache.Len())
	require.Equal(t, int32(1), firstStates.storeForCalls.Load())
	require.Equal(t, int32(1), secondStates.storeForCalls.Load())
	require.Eventually(t, func() bool {
		return logs.FilterMessage("filestream shared store cleaner started").Len() == 2
	}, time.Second, time.Millisecond)

	result.release()
	release2()
}

func TestStoreCache_ConcurrentAcquireRelease(t *testing.T) {
	setupCacheForTest(t)

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
				_, release, err := acquireStore(logger, states, "filestream")
				if err != nil {
					errs <- err
					continue
				}
				release()
			}
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	require.Equal(t, 0, globalCache.Len(), "all references released; cache must be empty")
}

type countingStateStore struct {
	registry      *statestore.Registry
	storeForCalls atomic.Int32
	key           string
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

// setupCacheForTest resets the global cache and registers a cleanup that
// asserts all acquired references have been released.
func setupCacheForTest(t *testing.T) {
	t.Helper()
	resetCacheForTest()
	t.Cleanup(func() {
		require.Zero(t, globalCache.Len(), "all acquired store references must be released")
	})
}

// resetCacheForTest replaces the global cache with a fresh instance to
// provide test isolation. Tests must release all references themselves.
func resetCacheForTest() {
	globalCache = statemanager.NewCache[*logfileEntry](func(e *logfileEntry) {
		e.ackUpdater.Close()
		e.store.Release()
	})
}
