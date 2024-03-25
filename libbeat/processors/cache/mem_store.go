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

package cache

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

var memStores = memStoreSet{stores: map[string]*memStore{}}

// memStoreSet is a collection of shared memStore caches.
type memStoreSet struct {
	mu     sync.Mutex
	stores map[string]*memStore
}

// get returns a memStore cache with the provided ID based on the config.
// If a memStore with the ID already exist, its configuration is adjusted
// and its reference count is increased. The returned context.CancelFunc
// reduces the reference count and deletes the memStore from the set if the
// count reaches zero.
func (s *memStoreSet) get(id string, cfg config) (*memStore, context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, ok := s.stores[id]
	if !ok {
		store = newMemStore(cfg, id)
		s.stores[store.id] = store
	}
	store.add(cfg)

	return store, func() {
		store.dropFrom(s)
	}
}

// free removes the memStore with the given ID from the set. free is safe
// for concurrent use.
func (s *memStoreSet) free(id string) {
	s.mu.Lock()
	delete(s.stores, id)
	s.mu.Unlock()
}

// memStore is a memory-backed cache store.
type memStore struct {
	mu       sync.Mutex
	cache    map[string]*CacheEntry
	expiries expiryHeap
	ttl      time.Duration // ttl is the time entries are valid for in the cache.
	refs     int           // refs is the number of processors referring to this store.
	// dirty marks the cache as changed from the
	// state in a backing file if it exists.
	dirty bool

	// id is the index into global cache store for the cache.
	id string

	// cap is the maximum number of elements the cache
	// will hold. If not positive, no limit.
	cap int
	// effort is the number of entries to examine during
	// expired element eviction. If not positive, full effort.
	effort int
}

// newMemStore returns a new memStore configured to apply the give TTL duration.
// The memStore is guaranteed not to grow larger than cap elements. id is the
// look-up into the global cache store the memStore is held in.
func newMemStore(cfg config, id string) *memStore {
	return &memStore{
		id:    id,
		cache: make(map[string]*CacheEntry),

		// Mark the ttl as invalid until we have had a put
		// operation configured. While the shared backing
		// data store is incomplete, and has no put operation
		// defined, the TTL will be invalid, but will never
		// be accessed since all time operations outside put
		// refer to absolute times, held by the CacheEntry.
		ttl:    -1,
		cap:    -1,
		effort: -1,
	}
}

func (c *memStore) String() string { return "memory:" + c.id }

// add updates the receiver for a new operation. It increases the reference
// count for the receiver, and if the config is a put operation and has no
// previous put operation defined, the TTL, cap and effort will be set from
// cfg. add is safe for concurrent use.
func (c *memStore) add(cfg config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refs++

	// We may have already constructed the store with
	// a get or a delete config, so set the TTL, cap
	// and effort if we have a put config. If another
	// put config has already been included, we ignore
	// the put options now.
	if cfg.Put == nil {
		return
	}
	if c.ttl == -1 {
		// putConfig.TTL is a required field, so we don't
		// need to check for nil-ness.
		c.ttl = *cfg.Put.TTL
		c.cap = cfg.Store.Capacity
		c.effort = cfg.Store.Effort
	}
}

// dropFrom decreases the reference count for the memStore and removes it from
// the stores map if the count is zero. dropFrom is safe for concurrent use.
func (c *memStore) dropFrom(stores *memStoreSet) {
	c.mu.Lock()
	c.refs--
	if c.refs < 0 {
		panic("invalid reference count")
	}
	if c.refs == 0 {
		stores.free(c.id)
		// GC assists.
		c.cache = nil
		c.expiries = nil
	}
	c.mu.Unlock()
}

// Get returns the cached value associated with the provided key. If there is
// no value for the key, or the value has expired Get returns ErrNoData. Get
// is safe for concurrent use.
func (c *memStore) Get(key string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.cache[key]
	if !ok {
		return nil, ErrNoData
	}
	if time.Now().After(v.Expires) {
		delete(c.cache, key)
		return nil, ErrNoData
	}
	return v.Value, nil
}

// Put stores the provided value in the cache associated with the given key.
// The value is given an expiry time based on the configured TTL of the cache.
// Put is safe for concurrent use.
func (c *memStore) Put(key string, val any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.evictExpired(now)
	e := &CacheEntry{
		Key:     key,
		Value:   val,
		Expires: now.Add(c.ttl),
	}
	// if the key is being overwritten we remove its previous expiry entry
	// this will prevent expiries heap to grow with large TTLs and recurring keys
	if prev, found := c.cache[key]; found {
		heap.Remove(&c.expiries, prev.index)
	}
	c.cache[key] = e
	heap.Push(&c.expiries, e)
	c.dirty = true
	return nil
}

// evictExpired removes up to effort elements from the cache when the cache
// is below capacity, retaining all elements that have not expired. If the
// cache is at or above capacity, the oldest elements are removed to bring
// it under the capacity limit.
func (c *memStore) evictExpired(now time.Time) {
	for n := 0; (c.effort <= 0 || n < c.effort) && len(c.cache) != 0; n++ {
		if c.expiries[0].Expires.After(now) {
			break
		}
		e := c.expiries.pop()
		delete(c.cache, e.Key)
	}
	if c.cap <= 0 {
		// No cap, so depend on effort.
		return
	}
	for len(c.cache) >= c.cap {
		e := c.expiries.pop()
		delete(c.cache, e.Key)
	}
}

// Delete removes the value associated with the provided key from the cache.
// Delete is safe for concurrent use.
func (c *memStore) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.cache[key]
	if !ok {
		return nil
	}
	heap.Remove(&c.expiries, v.index)
	delete(c.cache, key)
	c.dirty = true
	return nil
}

var _ heap.Interface = (*expiryHeap)(nil)

// expiryHeap is a min-date heap.
//
// TODO: This could be a queue instead, though deletion becomes more
// complicated in that case.
type expiryHeap []*CacheEntry

func (h *expiryHeap) pop() *CacheEntry {
	e := heap.Pop(h).(*CacheEntry)
	e.index = -1
	return e
}

func (h expiryHeap) Len() int {
	return len(h)
}
func (h expiryHeap) Less(i, j int) bool {
	return h[i].Expires.Before(h[j].Expires)
}
func (h expiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *expiryHeap) Push(v any) {
	e := v.(*CacheEntry)
	e.index = len(*h)
	*h = append(*h, e)
}
func (h *expiryHeap) Pop() any {
	v := (*h)[len(*h)-1]
	(*h)[len(*h)-1] = nil // Help GC.
	*h = (*h)[:len(*h)-1]
	return v
}
