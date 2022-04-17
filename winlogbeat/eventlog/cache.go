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

package eventlog

// This component of the eventlog package provides a cache for storing Handles
// to event message files.

import (
	"expvar"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/winlogbeat/sys"
)

// Stats for the message file caches.
var (
	cacheStats = expvar.NewMap("msg_file_cache")
)

// Constants that control the cache behavior.
const (
	expirationTimeout time.Duration = 2 * time.Minute
	janitorInterval   time.Duration = 30 * time.Second
	initialSize       int           = 10
)

// Function type for loading event message files associated with the given
// event log and source name.
type messageFileLoaderFunc func(eventLogName, sourceName string) sys.MessageFiles

// Function type for freeing Handles.
type freeHandleFunc func(handle uintptr) error

// handleCache provides a synchronized cache that holds MessageFiles.
type messageFilesCache struct {
	cache        *common.Cache
	loader       messageFileLoaderFunc
	freer        freeHandleFunc
	eventLogName string

	// Cache metrics.
	hit  func() // Increments number of cache hits.
	miss func() // Increments number of cache misses.
	size func() // Sets the current cache size.
}

// newHandleCache creates and returns a new handleCache that has been
// initialized (including starting a periodic janitor goroutine to purge
// expired Handles).
func newMessageFilesCache(eventLogName string, loader messageFileLoaderFunc,
	freer freeHandleFunc,
) *messageFilesCache {
	size := &expvar.Int{}
	cacheStats.Set(eventLogName+"Size", size)

	hc := &messageFilesCache{
		loader:       loader,
		freer:        freer,
		eventLogName: eventLogName,
		hit:          func() { cacheStats.Add(eventLogName+"Hits", 1) },
		miss:         func() { cacheStats.Add(eventLogName+"Misses", 1) },
	}
	hc.cache = common.NewCacheWithRemovalListener(expirationTimeout,
		initialSize, hc.evictionHandler)
	hc.cache.StartJanitor(janitorInterval)
	hc.size = func() {
		s := hc.cache.Size()
		size.Set(int64(s))
		debugf("messageFilesCache[%s] size=%d", hc.eventLogName, s)
	}
	return hc
}

// get returns a cached MessageFiles for the given sourceName.
// If no item is cached, then one is loaded, stored, and returned.
// Callers should check the MessageFiles.Err value to see if an error occurred
// while loading the message files.
func (hc *messageFilesCache) get(sourceName string) sys.MessageFiles {
	v := hc.cache.Get(sourceName)
	if v == nil {
		hc.miss()

		// Handle to event message file for sourceName is not cached. Attempt
		// to load the Handles into the cache.
		v = hc.loader(hc.eventLogName, sourceName)

		// Store the newly loaded value. Since this code does not lock we must
		// check if a value was already loaded.
		existing := hc.cache.PutIfAbsent(sourceName, v)
		if existing != nil {
			// A value was already loaded, so free the handles we just created.
			messageFiles, _ := v.(sys.MessageFiles)
			hc.freeHandles(messageFiles)

			// Return the existing cached value.
			messageFiles, _ = existing.(sys.MessageFiles)
			return messageFiles
		}
		hc.size()
	} else {
		hc.hit()
	}

	messageFiles, _ := v.(sys.MessageFiles)
	return messageFiles
}

// evictionHandler is the callback handler that receives notifications when
// a key-value pair is evicted from the messageFilesCache.
func (hc *messageFilesCache) evictionHandler(k common.Key, v common.Value) {
	// Update the size on a different goroutine after the callback completes.
	defer func() { go hc.size() }()

	messageFiles, ok := v.(sys.MessageFiles)
	if !ok {
		return
	}

	debugf("messageFilesCache[%s] Evicting messageFiles %+v for sourceName %v.",
		hc.eventLogName, messageFiles, k)
	hc.freeHandles(messageFiles)
}

// freeHandles free the event message file Handles so that the modules can
// be unloaded. The Handles are no longer valid after being freed.
func (hc *messageFilesCache) freeHandles(mf sys.MessageFiles) {
	for _, fh := range mf.Handles {
		err := hc.freer(fh.Handle)
		if err != nil {
			logp.Warn("messageFilesCache[%s] FreeLibrary error for handle %v",
				hc.eventLogName, fh.Handle)
		}
	}
}
