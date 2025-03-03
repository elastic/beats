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

package add_process_metadata

import (
	"sync"
	"time"
)

type processCacheEntry struct {
	metadata   *processMetadata
	err        error
	expiration time.Time
}

type processCache struct {
	provider   processMetadataProvider
	expiration time.Duration

	cap    int // cap is the maximum number of elements the cache will hold.
	effort int // effort is the number of entries to examine during expired element eviction.

	rwMutex sync.RWMutex // rwMutex protects the cache map.
	cache   map[int]processCacheEntry
}

func newProcessCache(expiration time.Duration, cap, effort int, provider processMetadataProvider) processCache {
	return processCache{
		cache:      make(map[int]processCacheEntry),
		expiration: expiration,
		cap:        cap,
		effort:     effort,
		provider:   provider,
	}
}

func (pc *processCache) getEntryUnlocked(pid int) (entry processCacheEntry, valid bool) {
	if entry, valid = pc.cache[pid]; valid {
		valid = entry.expiration.After(time.Now())
	}
	return entry, valid
}

func (pc *processCache) GetProcessMetadata(pid int) (*processMetadata, error) {
	pc.rwMutex.RLock()
	entry, valid := pc.getEntryUnlocked(pid)
	pc.rwMutex.RUnlock()

	if !valid {
		pc.rwMutex.Lock()
		defer pc.rwMutex.Unlock()

		pc.tryEvictExpired()
		if len(pc.cache) >= pc.cap {
			pc.evictRandomEntry()
		}

		// Make sure someone else didn't generate this entry while we were
		// waiting for the write lock
		if entry, valid = pc.getEntryUnlocked(pid); !valid {
			entry.metadata, entry.err = pc.provider.GetProcessMetadata(pid)
			entry.expiration = time.Now().Add(pc.expiration)
			pc.cache[pid] = entry
		}
	}
	return entry.metadata, entry.err
}

// tryEvictExpired implements a random sampling expired element cache
// eviction policy.
func (pc *processCache) tryEvictExpired() {
	now := time.Now()
	n := 0
	for pid, entry := range pc.cache {
		if n >= pc.effort {
			return
		}
		if now.After(entry.expiration) {
			delete(pc.cache, pid)
		}
		n++
	}
}

// evictRandomEntry implements a random cache eviction policy.
func (pc *processCache) evictRandomEntry() {
	for pid := range pc.cache {
		delete(pc.cache, pid)
		return
	}
}
