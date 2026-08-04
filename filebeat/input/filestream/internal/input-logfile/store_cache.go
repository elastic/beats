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

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

type storeCacheState uint8

const (
	storeInitializing storeCacheState = iota
	storeActive
	storeDraining
)

type storeCacheEntry struct {
	state      storeCacheState
	ready      chan struct{}
	closed     chan struct{}
	readyOnce  sync.Once
	closedOnce sync.Once
	initErr    error
	store      *store
	users      int
	cancel     context.CancelFunc
	cleanerWg  sync.WaitGroup
	interval   time.Duration
}

type storeCache struct {
	mu      sync.Mutex
	entries map[string]*storeCacheEntry
}

var globalStoreCache = storeCache{entries: make(map[string]*storeCacheEntry)}

func acquireStore(logger *logp.Logger, states statestore.States, prefix string) (*store, error) {
	key := states.StoreKey()
	for {
		globalStoreCache.mu.Lock()
		entry := globalStoreCache.entries[key]
		if entry == nil {
			entry = &storeCacheEntry{
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
			logger.Debugw("filestream shared store cache hit", "key", key, "users", users)
			return entry.store, nil
		case storeInitializing:
			ready := entry.ready
			globalStoreCache.mu.Unlock()
			logger.Debugw("waiting for filestream shared store initialization", "key", key)
			<-ready
			if entry.initErr != nil {
				return nil, entry.initErr
			}
		case storeDraining:
			closed := entry.closed
			globalStoreCache.mu.Unlock()
			started := time.Now()
			logger.Debugw("waiting for draining filestream shared store", "key", key)
			<-closed
			logger.Debugw(
				fmt.Sprintf(
					"finished waiting for draining filestream shared store. Waited for %s",
					time.Since(started),
				),
				"key", key,
			)
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

	logger.Debugw("initializing filestream shared store cache entry", "key", key)
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
		// Even a new store needs to be "released"
		s.Release()
		return nil, errors.New("clean up interval must be > 0")
	}
	ctx, cancel := context.WithCancel(context.Background())
	entry.cleanerWg.Add(1)
	s.onClose = func() { globalStoreCache.storeClosed(key, entry) }

	globalStoreCache.mu.Lock()
	if globalStoreCache.entries[key] != entry || entry.initErr != nil {
		initErr := entry.initErr
		globalStoreCache.mu.Unlock()
		cancel()
		s.Release()
		if initErr == nil {
			initErr = errors.New("filestream shared store cache entry was reset")
		}
		return nil, initErr
	}

	entry.store, entry.interval, entry.cancel = s, interval, cancel
	entry.state, entry.users = storeActive, 1
	s.cacheEntry = entry
	s.Retain()
	entry.closeReady()
	globalStoreCache.mu.Unlock()
	logger.Debugw(
		"initialized filestream shared store cache entry",
		"key", key,
		"interval", interval,
		"users", 1,
	)

	go func() {
		defer entry.cleanerWg.Done()
		defer entry.store.Release()
		logger.Debugw("filestream shared store cleaner started", "key", key)
		(&cleaner{log: logger}).run(ctx, s, interval)
		logger.Debugw("filestream shared store cleaner stopped", "key", key)
	}()
	return s, nil
}

func releaseAcquiredStore(logger *logp.Logger, s *store) {
	globalStoreCache.mu.Lock()
	entry := s.cacheEntry
	if entry == nil || entry.state != storeActive {
		globalStoreCache.mu.Unlock()
		s.Release()
		return
	}
	entry.users--
	users := entry.users
	if users > 0 {
		globalStoreCache.mu.Unlock()
		logger.Debugw("released filestream shared store", "users", users)
		s.Release()
		return
	}
	entry.state = storeDraining
	globalStoreCache.mu.Unlock()
	logger.Debug("draining filestream shared store")
	entry.cancel()
	entry.cleanerWg.Wait()
	logger.Debug("releasing filestream shared store cache reference")
	s.Release()
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
