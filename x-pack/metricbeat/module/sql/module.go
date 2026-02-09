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
// created once (lazily on first call) and lives for the duration of the Beat process.
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

type module struct {
	mb.BaseModule

	// Shared across all module instances via the ModuleBuilder closure.
	registryOnce *sync.Once
	registry     **statestore.Registry
	registryErr  *error
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
// Closure Pattern: This function creates a closure containing the shared variables
// (once, registry, registryErr) that persist across all SQL module instances.
// Each module instance receives pointers to these shared variables, ensuring that:
//   - sync.Once guarantees single initialization (thread-safe)
//   - All modules access the exact same registry pointer
//   - No file conflicts occur from multiple independent registries
//
// This function is called ONCE during init() and registered in mb.Registry.
// The returned factory function is called MULTIPLE TIMES (once per module instance).
func ModuleBuilder() mb.ModuleFactory {
	var (
		once        sync.Once
		registry    *statestore.Registry
		registryErr error
	)
	return func(base mb.BaseModule) (mb.Module, error) {
		return &module{
			BaseModule:   base,
			registryOnce: &once,
			registry:     &registry,
			registryErr:  &registryErr,
		}, nil
	}
}

// GetCursorRegistry returns the shared statestore registry for cursor persistence.
// The registry and its underlying memlog backend are created once (on the first
// call across all SQL module instances) and then reused. Each caller should use
// registry.Get("cursor-state") to obtain a ref-counted Store handle.
//
// The registry itself is never closed (lives with the Beat process). This is by
// design — all Store handles must be closed (via mb.Closer), but the registry
// persists to allow new MetricSets to be created dynamically. Registry cleanup
// happens automatically when the Beat process exits.
//
// Thread-safety: This method uses sync.Once to ensure thread-safe initialization
// even when called concurrently from multiple MetricSet instances.
func (m *module) GetCursorRegistry() (*statestore.Registry, error) {
	m.registryOnce.Do(func() {
		logger := logp.NewLogger("sql.cursor")
		dataPath := paths.Resolve(paths.Data, "sql-cursor")

		reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
			Root:     dataPath,
			FileMode: 0o600,
		})
		if err != nil {
			*m.registryErr = fmt.Errorf("failed to create memlog registry: %w", err)
			return
		}

		*m.registry = statestore.NewRegistry(reg)
		logger.Debugf("Created shared SQL cursor registry at %p", *m.registry)
	})
	return *m.registry, *m.registryErr
}
