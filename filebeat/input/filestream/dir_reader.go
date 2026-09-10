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
type cachedDirReader struct {
	mu     sync.Mutex
	ttl    time.Duration
	cache  map[string]dirListing
	stopCh chan struct{}
}

func newCachedDirReader(ttl time.Duration) *cachedDirReader {
	c := &cachedDirReader{
		ttl:    ttl,
		cache:  make(map[string]dirListing),
		stopCh: make(chan struct{}),
	}
	go c.sweepLoop()
	return c
}

// sweepLoop removes expired cache entries every TTL. Run in a goroutine.
func (c *cachedDirReader) sweepLoop() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, v := range c.cache {
				if now.Sub(v.fetched) >= c.ttl {
					delete(c.cache, k)
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

// readDir returns the cached listing for dir, refreshing it from the OS if
// the entry is absent or older than the TTL. The lock is held during the
// underlying os.ReadDir call so that concurrent callers for the same directory
// wait for a single syscall rather than each issuing their own.
func (c *cachedDirReader) readDir(dir string) (dirListing, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.cache[dir]; ok && time.Since(e.fetched) < c.ttl {
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
	c.cache[dir] = listing
	return listing, nil
}
