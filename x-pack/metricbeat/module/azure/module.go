// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

// Module extends mb.Module with shared resources for the azure module.
// MetricSets that need cursor state should type-assert base.Module() to this
// interface to access the shared statestore registry.
type Module interface {
	mb.Module
	// GetCursorRegistry returns the shared statestore registry for cursor
	// persistence. Created lazily on first call — no disk I/O if lookback
	// is disabled. Returns an error if the registry cannot be created.
	GetCursorRegistry() (*statestore.Registry, error)
}

// sharedRegistryState holds the lazily-created statestore registry shared across
// all azure module instances produced by the same ModuleBuilder closure.
//
// Path-aware caching: in production, paths.Resolve(paths.Data, "azure-cursor")
// never changes, so the registry is created once per process. In tests, each
// test overrides paths.Paths.Data to a unique t.TempDir(), causing the path to
// change and a new registry to be created — giving test isolation without leakage.
type sharedRegistryState struct {
	mu       sync.Mutex
	registry *statestore.Registry
	err      error
	dataPath string
}

type module struct {
	mb.BaseModule
	shared *sharedRegistryState
}

// ModuleBuilder returns a ModuleFactory that shares a single statestore.Registry
// across all azure module instances. The registry is created lazily on the first
// call to GetCursorRegistry, so no disk I/O occurs for non-cursor configurations.
func ModuleBuilder() mb.ModuleFactory {
	shared := &sharedRegistryState{}
	return func(base mb.BaseModule) (mb.Module, error) {
		return &module{
			BaseModule: base,
			shared:     shared,
		}, nil
	}
}

// GetCursorRegistry returns the shared statestore.Registry for cursor persistence.
// Thread-safe; created lazily on first call.
func (m *module) GetCursorRegistry() (*statestore.Registry, error) {
	s := m.shared
	s.mu.Lock()
	defer s.mu.Unlock()

	dataPath := paths.Resolve(paths.Data, "azure-cursor")

	if s.registry != nil && s.dataPath == dataPath {
		return s.registry, s.err
	}

	logger := logp.NewLogger("azure.cursor")
	reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
		Root:     dataPath,
		FileMode: 0o600,
	})
	if err != nil {
		s.err = fmt.Errorf("failed to create azure cursor registry: %w", err)
		s.registry = nil
		s.dataPath = dataPath
		return nil, s.err
	}

	s.registry = statestore.NewRegistry(reg)
	s.err = nil
	s.dataPath = dataPath
	return s.registry, nil
}
