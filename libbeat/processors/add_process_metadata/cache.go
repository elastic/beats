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
	expiration time.Time
}

type processCache struct {
	cache      map[int]processCacheEntry
	provider   processMetadataProvider
	expiration time.Duration
	mutex      sync.Mutex
}

func newProcessCache(expiration time.Duration, provider processMetadataProvider) processCache {
	return processCache{
		cache:      make(map[int]processCacheEntry),
		expiration: expiration,
		provider:   provider,
	}
}

func (pc *processCache) GetProcessMetadata(pid int) (*processMetadata, error) {
	if pid == 0 {
		return nil, nil
	}
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	entry, found := pc.cache[pid]
	if !found || entry.expiration.Before(time.Now()) {
		var err error
		entry.metadata, err = pc.provider.GetProcessMetadata(pid)
		if err != nil {
			return nil, err
		}
		entry.expiration = time.Now().Add(pc.expiration)
		pc.cache[pid] = entry
	}
	return entry.metadata, nil
}
