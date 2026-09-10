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
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// dirListing holds the entries and sorted names for a single directory.
type dirListing struct {
	entries []os.DirEntry
	names   []string
	fetched time.Time
}

// dirReader abstracts reading a directory's entries. The abstraction allows a
// caching implementation to be shared across fileScanner instances so that many
// inputs watching the same base directory make only one readdir syscall per TTL
// window instead of one per input.
type dirReader interface {
	// readDir returns the sorted entries and names for dir.
	readDir(dir string) (dirListing, error)
}

// osDirReader reads directories directly from the OS without caching.
type osDirReader struct{}

func (osDirReader) readDir(dir string) (dirListing, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dirListing{}, err
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return dirListing{entries: entries, names: names}, nil
}

// Process-wide singleton dir reader shared across all filestream Plugin()
// instances. acquireSharedDirReader / releaseSharedDirReader follow the same
// ref-counted acquire/release pattern as oteltelemetry.AcquireSystemBridge.
var (
	dirReaderMu   sync.Mutex
	dirReaderInst *cachedDirReader
	dirReaderRefs int
)

// acquireSharedDirReader returns the process-wide cachedDirReader and a release
// function the caller must invoke on shutdown. On first call the singleton is
// created and its sweep goroutine started. Subsequent calls increment the
// reference count. When the last caller releases, the sweep goroutine is stopped
// and the singleton is discarded.
func acquireSharedDirReader() (dirReader, func()) {
	dirReaderMu.Lock()
	defer dirReaderMu.Unlock()

	if dirReaderInst == nil {
		dirReaderInst = newCachedDirReader(time.Second)
	}
	dirReaderRefs++

	var once sync.Once
	return dirReaderInst, func() {
		once.Do(func() {
			dirReaderMu.Lock()
			defer dirReaderMu.Unlock()
			dirReaderRefs--
			if dirReaderRefs <= 0 {
				dirReaderInst.stop()
				dirReaderInst = nil
				dirReaderRefs = 0
			}
		})
	}
}

// cachedDirReader caches directory listings for a configurable TTL. A single
// instance shared across all fileScanner instances reduces readdir syscalls when
// many inputs watch the same base directory.
//
// Locking discipline:
//   - mu (RWMutex) guards cache and pathCounts. Readers hold RLock; writers hold Lock.
//   - dirMus holds one *sync.Mutex per directory path. A goroutine acquires the
//     per-dir lock before calling os.ReadDir, so concurrent reads for the same
//     directory are serialised into one syscall while reads for distinct directories
//     proceed in parallel.
type cachedDirReader struct {
	mu         sync.RWMutex
	defaultTTL time.Duration
	// pathCounts tracks how many callers have registered each (dir, interval) pair.
	// The effective TTL for a dir is the minimum interval with a positive count.
	// When a caller deregisters, its count is decremented; if it reaches zero the
	// entry is removed and the TTL for that dir may rise.
	pathCounts map[string]map[time.Duration]int
	cache      map[string]dirListing
	dirMus     sync.Map // map[string]*sync.Mutex — per-directory fetch locks
	stopCh     chan struct{}
}

func newCachedDirReader(defaultTTL time.Duration) *cachedDirReader {
	c := &cachedDirReader{
		defaultTTL: defaultTTL,
		pathCounts: make(map[string]map[time.Duration]int),
		cache:      make(map[string]dirListing),
		stopCh:     make(chan struct{}),
	}
	go c.sweepLoop()
	return c
}

// registerPath records that an input with the given check interval is watching
// dir. It returns a deregister function the caller must invoke when the input
// stops. While at least one caller is registered for a dir the effective TTL
// for that dir equals the minimum registered interval; when all callers
// deregister it reverts to defaultTTL.
func (c *cachedDirReader) registerPath(dir string, interval time.Duration) func() {
	c.mu.Lock()
	if c.pathCounts[dir] == nil {
		c.pathCounts[dir] = make(map[time.Duration]int)
	}
	c.pathCounts[dir][interval]++
	c.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			c.mu.Lock()
			if counts, ok := c.pathCounts[dir]; ok {
				counts[interval]--
				if counts[interval] == 0 {
					delete(counts, interval)
				}
				if len(counts) == 0 {
					delete(c.pathCounts, dir)
				}
				// Invalidate the cached entry so the next fetch uses the new TTL.
				delete(c.cache, dir)
			}
			c.mu.Unlock()
		})
	}
}

// ttlForDir returns the effective TTL for dir. Must be called with c.mu held (read or write).
func (c *cachedDirReader) ttlForDir(dir string) time.Duration {
	counts, ok := c.pathCounts[dir]
	if !ok || len(counts) == 0 {
		return c.defaultTTL
	}
	min := time.Duration(math.MaxInt64)
	for d := range counts {
		if d < min {
			min = d
		}
	}
	return min
}

// sweepLoop removes expired cache entries every defaultTTL. Run in a goroutine.
func (c *cachedDirReader) sweepLoop() {
	ticker := time.NewTicker(c.defaultTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for dir, listing := range c.cache {
				if now.Sub(listing.fetched) >= c.ttlForDir(dir) {
					delete(c.cache, dir)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}

// stop signals sweepLoop to exit.
func (c *cachedDirReader) stop() {
	close(c.stopCh)
}

// readDir returns the cached listing for dir, refreshing it from the OS if the
// entry is absent or older than the effective TTL for that dir.
//
// Reads for distinct directories proceed concurrently. Reads for the same
// directory are serialised by a per-directory mutex so that at most one
// os.ReadDir syscall runs for any given dir at a time; subsequent waiters
// receive the result from the cache rather than issuing their own syscall.
func (c *cachedDirReader) readDir(dir string) (dirListing, error) {
	// Fast path: serve from cache under a read lock.
	c.mu.RLock()
	e, ok := c.cache[dir]
	ttl := c.ttlForDir(dir)
	c.mu.RUnlock()
	if ok && time.Since(e.fetched) < ttl {
		return e, nil
	}

	// Slow path: acquire the per-directory lock so only one goroutine calls
	// os.ReadDir for this dir at a time while others block here.
	muVal, _ := c.dirMus.LoadOrStore(dir, new(sync.Mutex))
	dirMu := muVal.(*sync.Mutex) //nolint:errcheck
	dirMu.Lock()
	defer dirMu.Unlock()

	// Re-check now that we hold the per-dir lock; another goroutine may have
	// already fetched and populated the cache while we were waiting.
	c.mu.RLock()
	e, ok = c.cache[dir]
	ttl = c.ttlForDir(dir)
	c.mu.RUnlock()
	if ok && time.Since(e.fetched) < ttl {
		return e, nil
	}

	// os.ReadDir returns entries sorted by filename.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dirListing{}, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	listing := dirListing{entries: entries, names: names, fetched: time.Now()}

	c.mu.Lock()
	c.cache[dir] = listing
	c.mu.Unlock()

	return listing, nil
}

// globBase returns the base directory for a glob pattern: the longest path
// prefix that contains no wildcard metacharacters.
func globBase(pattern string) string {
	i := strings.IndexAny(pattern, "*?[")
	if i < 0 {
		return filepath.Dir(pattern)
	}
	return filepath.Dir(pattern[:i])
}
