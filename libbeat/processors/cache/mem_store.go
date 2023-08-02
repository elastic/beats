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
	"sync"
	"time"
)

// memStore is a memory-backed cache store.
type memStore struct {
	mu       sync.Mutex
	cache    map[string]*CacheEntry
	expiries expiryHeap
	ttl      time.Duration // ttl is the time entries are valid for in the cache.

	// cap is the maximum number of elements the cache
	// will hold. If not positive, no limit.
	cap int
	// effort is the number of entries to examine during
	// expired element eviction. If not positive, full effort.
	effort int
}

// newMemStore returns a new memStore configured to apply the give TTL duration.
// The memStore is guaranteed not to grow larger than cap elements.
func newMemStore(cfg config) *memStore {
	// Mark the ttl as invalid until we have had a put operation
	// configured.
	ttl := time.Duration(-1)
	cap := -1
	effort := -1
	if cfg.Put != nil {
		// putConfig.TTL is a required field, so we don't
		// need to check for nil-ness.
		ttl = *cfg.Put.TTL
		cap = cfg.Store.Capacity
		effort = cfg.Store.Effort
	}
	return &memStore{
		cache:  make(map[string]*CacheEntry),
		ttl:    ttl,
		cap:    cap,
		effort: effort,
	}
}

// setPutOptions allows concurrency-safe updating of the put options. While the shared
// backing data store is incomplete, and has no put operation defined, the TTL
// will be invalid, but will never be accessed since all time operations outside
// put refer to absolute times.
func (c *memStore) setPutOptions(cfg config) {
	if cfg.Put == nil {
		return
	}
	c.mu.Lock()
	if c.ttl == -1 {
		// putConfig.TTL is a required field, so we don't
		// need to check for nil-ness.
		c.ttl = *cfg.Put.TTL
		c.cap = cfg.Store.Capacity
		c.effort = cfg.Store.Effort
	}
	c.mu.Unlock()
}

// Get return the cached value associated with the provided key. If there is
// no value for the key, or the value has expired Get returns ErrNoData.
func (c *memStore) Get(key string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.cache[key]
	if !ok {
		return nil, ErrNoData
	}
	if time.Now().After(v.expires) {
		delete(c.cache, key)
		return nil, ErrNoData
	}
	return v.value, nil
}

// Put stores the provided value in the cache associated with the given key.
// The value is given an expiry time based on the configured TTL of the cache.
func (c *memStore) Put(key string, val any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.evictExpired(now)
	e := &CacheEntry{
		key:     key,
		value:   val,
		expires: now.Add(c.ttl),
	}
	c.cache[key] = e
	heap.Push(&c.expiries, e)
	return nil
}

// evictExpired removes up to effort elements from the cache when the cache
// is below capacity, retaining all elements that have not expired. If the
// cache is at or above capacity, the oldest elements are removed to bring
// it under the capacity limit.
func (c *memStore) evictExpired(now time.Time) {
	for n := 0; (c.effort <= 0 || n < c.effort) && len(c.cache) != 0; n++ {
		if c.expiries[0].expires.After(now) {
			break
		}
		e := c.expiries.pop()
		delete(c.cache, e.key)
	}
	if c.cap <= 0 {
		// No cap, so depend on effort.
		return
	}
	for len(c.cache) >= c.cap {
		e := c.expiries.pop()
		delete(c.cache, e.key)
	}
}

// Delete removes the value associated with the provided key from the cache.
func (c *memStore) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.cache[key]
	if !ok {
		return nil
	}
	heap.Remove(&c.expiries, v.index)
	delete(c.cache, key)
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
	return h[i].expires.Before(h[j].expires)
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
