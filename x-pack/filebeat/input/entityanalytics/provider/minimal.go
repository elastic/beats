// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/entcollect"
)

// MinimalStateProviderFactory creates an entcollect.Provider from the
// input configuration, returning the provider and its full/incremental
// sync intervals. Sessions 2-5 register factories here.
type MinimalStateProviderFactory func(cfg *config.C, log *logp.Logger) (p entcollect.Provider, fullSync, incrementalSync time.Duration, err error)

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
