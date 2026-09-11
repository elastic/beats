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
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

type storeCacheState uint8

const (
	storeInitializing storeCacheState = iota
	storeActive
	storeDraining
)

func (s storeCacheState) String() string {
	switch s {
	case storeInitializing:
		return "initializing"
	case storeActive:
		return "active"
	case storeDraining:
		return "draining"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

type storeCacheEntry struct {
	state      storeCacheState
	ready      chan struct{}
	closed     chan struct{}
	readyOnce  sync.Once
	closedOnce sync.Once
	initErr    error
	store      *store
	ackCH      *updateChan
	ackUpdater *updateWriter
	users      int
	cancel     context.CancelFunc
	cleanerWg  sync.WaitGroup
	interval   time.Duration
	key        string
}

type storeCache struct {
	mu      sync.Mutex
	entries map[string]*storeCacheEntry
}

var globalStoreCache = storeCache{entries: make(map[string]*storeCacheEntry)}

// acquireStore returns the shared store for a backend. The first caller initializes it;
// concurrent callers wait, and callers arriving during draining wait for its replacement.
func acquireStore(logger *logp.Logger, states statestore.States, prefix string) (*store, error) {
	key := states.StoreKey()
	logger = logger.
		Named("filestream.store_cache").
		WithLazy(zap.String("filestream_store_key", key))
	// Retry after a concurrent initialization or draining store has completed.
	for {
		globalStoreCache.mu.Lock()
		entry := globalStoreCache.entries[key]
		if entry == nil {
			entry = &storeCacheEntry{
				key:    key,
				state:  storeInitializing,
				ready:  make(chan struct{}),
				closed: make(chan struct{}),
			}
			globalStoreCache.entries[key] = entry
			globalStoreCache.mu.Unlock()
			return initializeStoreCacheEntry(logger, key, entry, states, prefix)
		}

		switch entry.state {
		case storeActive:
			entry.users++
			users := entry.users
			entry.store.Retain()
			globalStoreCache.mu.Unlock()
			logger.Debugw("filestream shared store cache hit", "store_users_count", users)
			return entry.store, nil
		case storeInitializing:
			ready := entry.ready
			globalStoreCache.mu.Unlock()
			logger.Debugw("waiting for filestream shared store initialization")
			<-ready
			if entry.initErr != nil {
				return nil, entry.initErr
			}
		case storeDraining:
			closed := entry.closed
			globalStoreCache.mu.Unlock()
			started := time.Now()
			logger.Debugw("waiting for draining filestream shared store")
			<-closed
			logger.Debugw(
				fmt.Sprintf(
					"finished waiting for draining filestream shared store. Waited for %s",
					time.Since(started),
				),
			)
		default:
			state := entry.state
			globalStoreCache.mu.Unlock()
			logger.Errorw(
				"unhandled filestream shared store cache state",
				"store_cache_state", state.String(),
			)
			return nil, fmt.Errorf("unhandled filestream shared store cache state %s for backend %q", state, key)
		}
	}
}

func initializeStoreCacheEntry(
	logger *logp.Logger,
	key string,
	entry *storeCacheEntry,
	states statestore.States,
	prefix string,
) (*store, error) {
	logger.Debugw("initializing filestream shared store cache entry")
	s, err := openStore(logger, states, prefix)
	if err != nil {
		globalStoreCache.mu.Lock()
		entry.initErr = err
		delete(globalStoreCache.entries, key)
		entry.closeReady()
		entry.closeClosed()
		globalStoreCache.mu.Unlock()
		return nil, err
	}

	interval := states.CleanupInterval()
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.onClose = func() { globalStoreCache.storeClosed(key, entry) }
	ackCH := newUpdateChan()
	ackUpdater := newUpdateWriter(s, ackCH)

	globalStoreCache.mu.Lock()
	if globalStoreCache.entries[key] != entry || entry.initErr != nil {
		initErr := entry.initErr
		globalStoreCache.mu.Unlock()
		cancel()
		ackUpdater.Close()
		s.Release()
		if initErr == nil {
			initErr = errors.New("filestream shared store cache entry was reset")
		}
		return nil, initErr
	}

	entry.cleanerWg.Add(1)
	entry.store, entry.ackCH, entry.ackUpdater = s, ackCH, ackUpdater
	entry.interval, entry.cancel = interval, cancel
	entry.state, entry.users = storeActive, 1
	s.cacheEntry = entry
	s.Retain()
	entry.closeReady()
	globalStoreCache.mu.Unlock()
	logger.Debugw(
		"initialized filestream shared store cache entry",
		"store_users_count", 1,
	)

	go func() {
		defer entry.cleanerWg.Done()
		defer entry.store.Release()
		logger.Debugw("filestream shared store cleaner started")
		runCleaner(logger, ctx, s, interval)
		logger.Debugw("filestream shared store cleaner stopped")
	}()

	return s, nil
}

func releaseAcquiredStore(logger *logp.Logger, s *store) {
	logger = logger.Named("filestream.store_cache")

	globalStoreCache.mu.Lock()
	entry := s.cacheEntry
	if entry == nil || entry.state != storeActive {
		globalStoreCache.mu.Unlock()
		s.Release()
		return
	}
	logger = logger.WithLazy(zap.String("filestream_store_key", s.cacheEntry.key))

	entry.users--
	users := entry.users
	if entry.users > 0 {
		globalStoreCache.mu.Unlock()
		s.Release()
		logger.Debugw("released filestream shared store", "store_users_count", users)
		return
	}
	entry.state = storeDraining
	globalStoreCache.mu.Unlock()
	logger.Debug("draining filestream shared store")
	entry.cancel()
	entry.cleanerWg.Wait()
	entry.ackUpdater.Close()
	s.Release()
	logger.Debugw("released filestream shared store", "store_users_count", users)
}

func (c *storeCache) storeClosed(key string, entry *storeCacheEntry) {
	c.mu.Lock()
	if c.entries[key] == entry {
		delete(c.entries, key)
	}
	entry.closeClosed()
	c.mu.Unlock()
}

func (e *storeCacheEntry) closeReady()  { e.readyOnce.Do(func() { close(e.ready) }) }
func (e *storeCacheEntry) closeClosed() { e.closedOnce.Do(func() { close(e.closed) }) }
