// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sql

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

// Module extends mb.Module with shared resources for the SQL module.
// MetricSets that need cursor state should type-assert base.Module() to this
// interface to access the shared statestore registry.
//
// Registry Lifetime: The statestore.Registry returned by GetCursorRegistry() is
// created lazily on first call and lives for the duration of the Beat process.
// It is never explicitly closed — the registry cleanup happens automatically when
// the Beat process exits. This is safe because:
//  1. The registry uses memlog backend which flushes writes synchronously
//  2. Beat shutdown sequence stops all modules before process exit
//  3. Individual Store handles (from registry.Get) are closed by MetricSets via mb.Closer
//
// This design matches the Filebeat pattern and prevents multiple independent stores
// from operating on the same files, which would cause file lock conflicts.
type Module interface {
	mb.Module
	// GetCursorRegistry returns the shared statestore registry for cursor
	// persistence. The registry is created lazily on first call — no disk I/O
	// occurs until a MetricSet actually enables cursor. Returns an error if
	// the registry could not be created.
	GetCursorRegistry() (*statestore.Registry, error)
}

// sharedRegistryState holds the shared statestore registry and tracks which
// data path it was created for. All SQL module instances created by the same
// ModuleBuilder share a single sharedRegistryState via pointer.
//
// Path-Aware Caching: In production, paths.Resolve(paths.Data, "sql-cursor")
// never changes, so the registry is created once and reused for the entire
// lifetime of the Beat process — identical behaviour to sync.Once.
// In integration tests, each test overrides paths.Paths.Data to a unique
// t.TempDir(), causing the resolved path to change. When a path change is
// detected, a new registry is created at the new location, giving each test
// an isolated cursor store without any cross-test state leakage.
type sharedRegistryState struct {
	mu       sync.Mutex
	registry *statestore.Registry
	err      error
	dataPath string // resolved path the current registry was created for
}

type module struct {
	mb.BaseModule

	// Shared across all module instances via the ModuleBuilder closure.
	shared *sharedRegistryState
}

func init() {
	if err := mb.Registry.AddModule("sql", ModuleBuilder()); err != nil {
		panic(err)
	}
}

// ModuleBuilder returns a ModuleFactory that shares a single statestore.Registry
// across all SQL module instances. The registry is created lazily on first call
// to GetCursorRegistry, so no disk I/O occurs for non-cursor configurations.
//
// Closure Pattern: This function creates a closure containing a shared
// sharedRegistryState that persists across all SQL module instances.
// Each module instance receives a pointer to this shared state, ensuring that:
//   - sync.Mutex guarantees thread-safe initialization and access
//   - All modules sharing the same data path access the exact same registry
//   - No file conflicts occur from multiple independent registries
//
// This function is called ONCE during init() and registered in mb.Registry.
// The returned factory function is called MULTIPLE TIMES (once per module instance).
func ModuleBuilder() mb.ModuleFactory {
	shared := &sharedRegistryState{}
	return func(base mb.BaseModule) (mb.Module, error) {
		return &module{
			BaseModule: base,
			shared:     shared,
		}, nil
	}
}

// GetCursorRegistry returns the shared statestore registry for cursor persistence.
// The registry and its underlying memlog backend are created on the first call
// and then reused as long as the resolved data path has not changed. Each caller
// should use registry.Get("cursor-state") to obtain a ref-counted Store handle.
//
// Path-aware caching: The method resolves the current data path via
// paths.Resolve(paths.Data, "sql-cursor"). If the resolved path matches the
// path used to create the cached registry, the cached registry is returned
// immediately. If the path has changed (which only happens in tests), a new
// registry is created at the new location.
//
// The registry itself is never closed (lives with the Beat process). This is by
// design — all Store handles must be closed (via mb.Closer), but the registry
// persists to allow new MetricSets to be created dynamically. Registry cleanup
// happens automatically when the Beat process exits.
//
// Thread-safety: This method uses sync.Mutex to ensure thread-safe access
// even when called concurrently from multiple MetricSet instances.
func (m *module) GetCursorRegistry() (*statestore.Registry, error) {
	s := m.shared
	s.mu.Lock()
	defer s.mu.Unlock()

	dataPath := paths.Resolve(paths.Data, "sql-cursor")

	// Return the cached registry if it was created for the same data path.
	if s.registry != nil && s.dataPath == dataPath {
		return s.registry, s.err
	}

	logger := logp.NewLogger("sql.cursor")

	reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
		Root:     dataPath,
		FileMode: 0o600,
	})
	if err != nil {
		s.err = fmt.Errorf("failed to create memlog registry: %w", err)
		s.registry = nil
		s.dataPath = dataPath
		return nil, s.err
	}

	s.registry = statestore.NewRegistry(reg)
	s.err = nil
	s.dataPath = dataPath
	logger.Debugf("Created shared SQL cursor registry at %s (ptr=%p)", dataPath, s.registry)
	return s.registry, nil
}
