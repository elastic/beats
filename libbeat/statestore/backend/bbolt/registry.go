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
//
// This file was contributed to by generative AI

package bbolt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.etcd.io/bbolt"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
)

const defaultFileMode os.FileMode = 0o600
const defaultDBTimeout = 1 * time.Second

var errRegistryClosed = errors.New("bbolt registry has been closed")

// Registry implements backend.Registry and manages a set of per-store bbolt databases.
type Registry struct {
	logger *logp.Logger

	mu     sync.Mutex
	active bool

	settings Settings
	stores   map[string]*store

	gcDone chan struct{}
	gcWg   sync.WaitGroup
}

// Settings configures a new Registry.
type Settings struct {
	// Root directory for all store files.
	Root string

	// FileMode is used as the file mode for newly created DB files.
	// File mode 0600 will be used if not set.
	FileMode os.FileMode

	// DiskTTL is the inactivity duration after which entries are considered stale.
	// If 0, disk GC is disabled.
	DiskTTL time.Duration

	// Timeout sets the bbolt open timeout.
	Timeout time.Duration

	// NoGrowSync disables the grow sync behavior in bbolt.
	NoGrowSync bool

	// NoFreelistSync disables freelist syncing in bbolt.
	NoFreelistSync bool
}

// New creates a new bbolt Registry.
func New(logger *logp.Logger, settings Settings) (*Registry, error) {
	if logger == nil {
		return nil, errors.New("bbolt registry requires a logger")
	}
	if settings.Root == "" {
		return nil, errors.New("bbolt registry root path is empty")
	}
	if settings.FileMode == 0 {
		settings.FileMode = defaultFileMode
	}
	if settings.Timeout == 0 {
		settings.Timeout = defaultDBTimeout
	}

	root, err := filepath.Abs(settings.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve bbolt registry root: %w", err)
	}
	settings.Root = root

	// Create root directory. Directory permissions are intentionally strict.
	if err := os.MkdirAll(settings.Root, 0o700); err != nil {
		return nil, fmt.Errorf("create bbolt registry root directory: %w", err)
	}

	r := &Registry{
		logger:   logger,
		active:   true,
		settings: settings,
		stores:   map[string]*store{},
		gcDone:   make(chan struct{}),
	}

	if settings.DiskTTL > 0 {
		r.gcWg.Add(1)
		go func() {
			defer r.gcWg.Done()
			r.runDiskGC()
		}()
	}

	return r, nil
}

// Access returns a store by name, creating it if needed.
func (r *Registry) Access(name string) (backend.Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.active {
		return nil, errRegistryClosed
	}
	if name == "" {
		return nil, errors.New("bbolt store name is empty")
	}

	if existing := r.stores[name]; existing != nil && !existing.isClosed() {
		return existing, nil
	}

	logger := r.logger.With("store", name)
	dbPath := filepath.Join(r.settings.Root, name+".db")
	s, err := openStore(logger, dbPath, r.settings)
	if err != nil {
		return nil, err
	}

	r.stores[name] = s
	return s, nil
}

// GetDB returns the underlying bbolt database for a named store.
// Returns nil if store doesn't exist or is closed.
// This is intended for debugging/inspection tools only.
func (r *Registry) GetDB(name string) *bbolt.DB {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.active {
		return nil
	}

	s := r.stores[name]
	if s == nil {
		return nil
	}

	return s.DB()
}

// Close closes the registry and all open stores.
func (r *Registry) Close() error {
	r.mu.Lock()
	if !r.active {
		r.mu.Unlock()
		return nil
	}
	r.active = false

	// Stop GC goroutine first to avoid racing against store close.
	if r.gcDone != nil {
		close(r.gcDone)
		r.gcDone = nil
	}

	stores := make([]*store, 0, len(r.stores))
	for _, s := range r.stores {
		stores = append(stores, s)
	}
	r.stores = nil
	r.mu.Unlock()

	r.gcWg.Wait()

	var firstErr error
	for _, s := range stores {
		if s == nil {
			continue
		}
		if err := s.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (r *Registry) runDiskGC() {
	interval := r.settings.DiskTTL
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.gcDone:
			return
		case <-ticker.C:
			// NOTE: Phase 1-2 plan calls for a full scan GC in store-level code.
			// For now, registry.go only dispatches to stores that implement GC.
			r.mu.Lock()
			stores := make([]*store, 0, len(r.stores))
			for _, s := range r.stores {
				stores = append(stores, s)
			}
			r.mu.Unlock()

			for _, s := range stores {
				if s == nil {
					continue
				}
				if err := s.collectGarbage(); err != nil {
					s.logger.Errorf("disk GC failed: %v", err)
				}
			}
		}
	}
}
