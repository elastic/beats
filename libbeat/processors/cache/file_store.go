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
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

var fileStores = fileStoreSet{stores: map[string]*fileStore{}}

// fileStoreSet is a collection of shared fileStore caches.
type fileStoreSet struct {
	mu     sync.Mutex
	stores map[string]*fileStore
}

// get returns a fileStore cache with the provided ID based on the config.
// If a fileStore with the ID already exist, its configuration is adjusted
// and its reference count is increased. The returned context.CancelFunc
// reduces the reference count and deletes the fileStore from the set if the
// count reaches zero.
func (s *fileStoreSet) get(id string, cfg config, log *logp.Logger) (*fileStore, context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, ok := s.stores[id]
	if !ok {
		store = newFileStore(cfg, id, pathFromConfig(cfg, log), log)
		s.stores[store.id] = store
	}
	store.add(cfg)

	return store, func() {
		store.dropFrom(s)
	}
}

// pathFromConfig returns the mapping form a config to a file-system path.
func pathFromConfig(cfg config, log *logp.Logger) string {
	path := filepath.Join(paths.Resolve(paths.Data, "cache_processor"), cleanFilename(cfg.Store.File.ID))
	log.Infow("mapping file-backed cache processor config to file path", "id", cfg.Store.File.ID, "path", path)
	return path
}

// cleanFilename replaces illegal printable characters (and space or dot) in
// filenames, with underscore.
func cleanFilename(s string) string {
	return pathCleaner.Replace(s)
}

var pathCleaner = strings.NewReplacer(
	"/", "_",
	"<", "_",
	">", "_",
	":", "_",
	`"`, "_",
	"/", "_",
	`\`, "_",
	"|", "_",
	"?", "_",
	"*", "_",
	".", "_",
	" ", "_",
)

// free removes the fileStore with the given ID from the set. free is safe
// for concurrent use.
func (s *fileStoreSet) free(id string) {
	s.mu.Lock()
	delete(s.stores, id)
	s.mu.Unlock()
}

// fileStore is a file-backed cache store.
type fileStore struct {
	memStore

	path string
	// cancel stops periodic write out operations.
	// Write out operations are protected by the
	// memStore's mutex.
	cancel context.CancelFunc

	log *logp.Logger
}

// newFileStore returns a new fileStore configured to apply the give TTL duration.
// The fileStore is guaranteed not to grow larger than cap elements. id is the
// look-up into the global cache store the fileStore is held in.
func newFileStore(cfg config, id, path string, log *logp.Logger) *fileStore {
	s := fileStore{
		path: path,
		log:  log,
		memStore: memStore{
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
		},
	}
	s.cancel = noop
	if cfg.Store.File.WriteOutEvery > 0 {
		var ctx context.Context
		ctx, s.cancel = context.WithCancel(context.Background())
		go s.periodicWriteOut(ctx, cfg.Store.File.WriteOutEvery)
	}
	s.readState()
	return &s
}

func (c *fileStore) String() string { return "file:" + c.id }

// dropFrom decreases the reference count for the fileStore and removes it from
// the stores map if the count is zero. dropFrom is safe for concurrent use.
func (c *fileStore) dropFrom(stores *fileStoreSet) {
	c.mu.Lock()
	c.refs--
	if c.refs < 0 {
		panic("invalid reference count")
	}
	if c.refs == 0 {
		// Stop periodic writes
		c.cancel()
		// and do a final write out.
		c.writeState(true)

		stores.free(c.id)
		// GC assists.
		c.cache = nil
		c.expiries = nil
	}
	c.mu.Unlock()
}

func (c *fileStore) readState() {
	f, err := os.Open(c.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			c.log.Debugw("no state on file system", "error", err)
		} else {
			c.log.Errorw("failed to open file to read state", "error", err)
		}
		return
	}
	defer f.Close()

	// It would be nice to be able at this stage to determine
	// whether the file is stale past the TTL of the cache, but
	// we do not have this information yet. So we must read
	// through all the elements. If any survive the filter, we
	// were alive, otherwise delete the file.

	c.log.Infow("reading state from file", "id", c.id, "path", c.path)
	dec := json.NewDecoder(f)
	for {
		var e CacheEntry
		err = dec.Decode(&e)
		if err != nil {
			if err != io.EOF {
				switch err := err.(type) {
				case *json.SyntaxError:
					c.log.Errorw("failed to read state element", "error", err, "path", c.path, "offset", err.Offset)
				default:
					c.log.Errorw("failed to read state element", "error", err, "path", c.path)
				}
			}
			break
		}
		if e.Expires.Before(time.Now()) {
			// Don't retain expired elements.
			c.dirty = true // The cache now does not reflect the file.
			continue
		}
		c.cache[e.Key] = &e
		heap.Push(&c.expiries, &e)
	}

	c.log.Infow("got state from file", "id", c.id, "entries", len(c.cache))
	if len(c.cache) != 0 {
		return
	}
	// We had no live entries, so delete the file.
	err = os.Remove(c.path)
	if err != nil {
		c.log.Errorw("failed to delete stale cache file", "error", err)
	}
}

// periodicWriteOut writes the cache contents to the backing file at the
// specified interval until the context is cancelled. periodicWriteOut is
// safe for concurrent use.
func (c *fileStore) periodicWriteOut(ctx context.Context, every time.Duration) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			c.mu.Lock()
			c.writeState(false)
			c.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// writeState writes the current cache state to the backing file.
// If final is true and the cache is empty, the file will be deleted.
func (c *fileStore) writeState(final bool) {
	if !c.dirty {
		return
	}
	c.log.Infow("write state to file", "id", c.id, "path", c.path)
	if len(c.cache) == 0 && final {
		err := os.Remove(c.path)
		if err != nil {
			c.log.Errorw("failed to delete write state when empty", "error", err)
		}
		return
	}
	f, err := os.CreateTemp(filepath.Dir(c.path), filepath.Base(c.path)+"-*.tmp")
	if err != nil {
		c.log.Errorw("failed to open file to write state", "error", err)
		return
	}
	// Try to make sure we are private.
	err = os.Chmod(f.Name(), 0o600)
	if err != nil {
		c.log.Errorw("failed to set state file mode", "error", err)
		return
	}
	tmp := f.Name()
	defer func() {
		err = f.Sync()
		if err != nil {
			c.log.Errorw("failed to sync file after writing state", "error", err)
			return
		}
		err = f.Close()
		if err != nil {
			c.log.Errorw("failed to close file after writing state", "error", err)
			return
		}
		// Try to be atomic.
		err = os.Rename(tmp, c.path)
		if err != nil {
			c.log.Errorw("failed to finalize writing state", "error", err)
		}
		c.log.Infow("write state to file sync and replace succeeded", "id", c.id, "path", c.path)
	}()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	now := time.Now()
	for c.expiries.Len() != 0 {
		e := c.expiries.pop()
		if e.Expires.Before(now) {
			// Don't write expired elements.
			continue
		}
		err = enc.Encode(e)
		if err != nil {
			c.log.Errorw("failed to write state element", "error", err)
			return
		}
	}
	// Only mark as not dirty if we succeeded in the write.
	c.dirty = false
	c.log.Infow("write state to file succeeded", "id", c.id, "path", c.path)
}
