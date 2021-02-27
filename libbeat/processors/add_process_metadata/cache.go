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
	cache      map[int]processCacheEntry
	provider   processMetadataProvider
	expiration time.Duration
	rwMutex    sync.RWMutex
}

func newProcessCache(expiration time.Duration, provider processMetadataProvider) processCache {
	return processCache{
		cache:      make(map[int]processCacheEntry),
		expiration: expiration,
		provider:   provider,
	}
}

func (pc *processCache) getEntryUnlocked(pid int) (entry processCacheEntry, valid bool) {
	if entry, valid = pc.cache[pid]; valid {
		valid = entry.expiration.After(time.Now())
	}
	return
}

func (pc *processCache) GetProcessMetadata(pid int) (*processMetadata, error) {
	pc.rwMutex.RLock()
	entry, valid := pc.getEntryUnlocked(pid)
	pc.rwMutex.RUnlock()

	if !valid {
		pc.rwMutex.Lock()
		defer pc.rwMutex.Unlock()
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
