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

package filestream

import (
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// maxDirCacheAge caps how long a directory listing may be served from cache.
const maxDirCacheAge = time.Second

// dirCacheEntry holds a cached directory listing protected by its own mutex.
// The per-entry mutex allows concurrent reads of distinct directories while
// serialising concurrent reads of the same directory so at most one OS call
// runs per directory at a time.
type dirCacheEntry struct {
	mu      sync.Mutex
	fetched atomic.Int64 // Unix nanoseconds; zero means no valid entry
	names   []string
	entries []os.DirEntry
}

func (e *dirCacheEntry) age() time.Duration {
	f := e.fetched.Load()
	if f == 0 {
		return time.Duration(int64(^uint64(0) >> 1)) // max int64 — always stale
	}
	return time.Since(time.Unix(0, f))
}

// set stores names/entries and timestamps the entry. Passing nil for both
// clears the entry so the next caller retries (used when an OS read fails).
func (e *dirCacheEntry) set(names []string, entries []os.DirEntry) {
	e.names = names
	e.entries = entries
	if names != nil || entries != nil {
		e.fetched.Store(time.Now().UnixNano())
	} else {
		e.fetched.Store(0)
	}
}

// dirCache is a process-wide cache of directory listings. A single instance is
// shared across all filestream Plugin() invocations via acquireSharedDirReader.
//
// Locking discipline:
//   - mu guards the entries map (held only while looking up / inserting entries).
//   - Each dirCacheEntry has its own mutex that serialises concurrent reads of
//     the same directory; reads for distinct directories proceed fully in parallel.
//   - Sweep runs inline on every miss path (under defer, after e.mu is released).
//     A swept timestamp throttles actual work to at most once per maxDirCacheAge.
type dirCache struct {
	mu      sync.Mutex
	entries map[string]*dirCacheEntry
	swept   time.Time
}

func newDirCache() *dirCache {
	return &dirCache{entries: make(map[string]*dirCacheEntry)}
}

// reset clears all cached entries. Called when the last receiver releases the
// singleton so the GC can reclaim any pinned slices.
func (c *dirCache) reset() {
	c.mu.Lock()
	c.entries = make(map[string]*dirCacheEntry)
	c.swept = time.Time{}
	c.mu.Unlock()
}

// entry returns the cache entry for dir, creating it if absent.
func (c *dirCache) entry(dir string) *dirCacheEntry {
	c.mu.Lock()
	e := c.entries[dir]
	if e == nil {
		e = &dirCacheEntry{}
		c.entries[dir] = e
	}
	c.mu.Unlock()
	return e
}

// readDirNames returns sorted file names for dir from the cache (when fresh
// and shared=true) or from the OS. shared=false bypasses the cache so
// sub-directories are always read directly; only the walk root is cached.
func (c *dirCache) readDirNames(dir string, maxAge time.Duration, shared bool) ([]string, error) {
	if !shared {
		return osDirNames(dir)
	}
	e := c.entry(dir)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.age() < maxAge {
		if e.names != nil {
			return e.names, nil
		}
		if e.entries != nil {
			e.names = entryNames(e.entries)
			return e.names, nil
		}
	}
	defer c.sweep()
	names, err := osDirNames(dir)
	if err != nil {
		e.set(nil, nil)
		return nil, err
	}
	e.set(names, nil)
	return names, nil
}

// readDirEntries returns sorted DirEntries for dir from the cache (when fresh
// and shared=true) or from the OS. shared=false bypasses the cache.
func (c *dirCache) readDirEntries(dir string, maxAge time.Duration, shared bool) ([]os.DirEntry, error) {
	if !shared {
		return os.ReadDir(dir)
	}
	e := c.entry(dir)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.age() < maxAge && e.entries != nil {
		return e.entries, nil
	}
	defer c.sweep()
	entries, err := os.ReadDir(dir)
	if err != nil {
		e.set(nil, nil)
		return nil, err
	}
	e.set(nil, entries)
	return entries, nil
}

// sweep removes entries older than maxDirCacheAge from the map. It is throttled
// by the swept timestamp so at most one sweep runs per maxDirCacheAge even if
// many misses occur concurrently. Sweep is called via defer after each OS read,
// so it executes after e.mu has been released — it only needs c.mu.
func (c *dirCache) sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if now.Sub(c.swept) < maxDirCacheAge {
		return
	}
	c.swept = now
	for dir, e := range c.entries {
		if now.Sub(time.Unix(0, e.fetched.Load())) >= maxDirCacheAge {
			delete(c.entries, dir)
		}
	}
}

// osDirNames reads sorted entry names from dir directly from the OS.
// Using Readdirnames + sort avoids allocating DirEntry objects, which is
// significantly faster for leaf directories that only need names.
func osDirNames(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

// entryNames extracts sorted names from a slice of DirEntry values.
func entryNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}

// Process-wide singleton dir cache shared across all filestream Plugin()
// invocations. acquireSharedDirReader / release follow the same ref-counted
// acquire/release pattern as oteltelemetry.AcquireSystemBridge.
var (
	dirReaderMu   sync.Mutex
	dirReaderInst *dirCache
	dirReaderRefs int
)

// acquireSharedDirReader returns the process-wide dirCache and a release
// function the caller must invoke on shutdown. On first call the singleton is
// created. Subsequent calls increment the reference count. When the last caller
// releases, the cache is cleared and the singleton is discarded — no background
// goroutine is involved, so cleanup is immediate and complete.
func acquireSharedDirReader() (*dirCache, func()) {
	dirReaderMu.Lock()
	defer dirReaderMu.Unlock()

	if dirReaderInst == nil {
		dirReaderInst = newDirCache()
	}
	dirReaderRefs++

	var once sync.Once
	return dirReaderInst, func() {
		once.Do(func() {
			dirReaderMu.Lock()
			defer dirReaderMu.Unlock()
			dirReaderRefs--
			if dirReaderRefs <= 0 {
				dirReaderInst.reset()
				dirReaderInst = nil
				dirReaderRefs = 0
			}
		})
	}
}
