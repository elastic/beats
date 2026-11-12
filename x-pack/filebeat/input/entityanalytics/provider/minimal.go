// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"net/http"
	"sync"
	"time"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/entcollect"
)

// MinimalStateProviderFactory creates a minimal-state provider from the
// input configuration.
type MinimalStateProviderFactory func(cfg *config.C, log *logp.Logger) (MinimalState, error)

// MinimalState is a minimal-state provider and the settings the input
// needs to run it.
type MinimalState struct {
	Provider entcollect.Provider

	// FullSync and IncrementalSync are the intervals between full and
	// incremental syncs.
	FullSync        time.Duration
	IncrementalSync time.Duration

	// Client is the HTTP client Provider uses, or nil if it does not
	// use HTTP. Before the first sync, the input wraps the client's
	// transport with the request tracer configured by Tracer, which
	// needs the input's ID and so cannot be set up here.
	Client *http.Client
	Tracer *TracerConfig

	// Untraced holds URL prefixes of requests that are never traced,
	// because their bodies carry credentials.
	Untraced []string
}

var (
	minimalStateProviderRegistry   = map[string]MinimalStateProviderFactory{}
	minimalStateProviderRegistryMu sync.RWMutex
)

// RegisterMinimalStateProvider registers a MinimalStateProviderFactory
// under name. Returns ErrExists if the name is already taken.
func RegisterMinimalStateProvider(name string, factory MinimalStateProviderFactory) error {
	minimalStateProviderRegistryMu.Lock()
	defer minimalStateProviderRegistryMu.Unlock()

	if _, exists := minimalStateProviderRegistry[name]; exists {
		return ErrExists
	}
	minimalStateProviderRegistry[name] = factory
	return nil
}

// GetMinimalStateProvider returns the MinimalStateProviderFactory for
// name. Returns ErrNotFound if the name has not been registered.
func GetMinimalStateProvider(name string) (MinimalStateProviderFactory, error) {
	minimalStateProviderRegistryMu.RLock()
	defer minimalStateProviderRegistryMu.RUnlock()

	factory, ok := minimalStateProviderRegistry[name]
	if !ok {
		return nil, ErrNotFound
	}
	return factory, nil
}

// HasMinimalStateProvider returns true if a MinimalStateProviderFactory
// is registered for name.
func HasMinimalStateProvider(name string) bool {
	minimalStateProviderRegistryMu.RLock()
	_, exists := minimalStateProviderRegistry[name]
	minimalStateProviderRegistryMu.RUnlock()
	return exists
}
