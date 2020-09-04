// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composable

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

// ContextProviderComm is the interface that a context provider uses to communicate back to Elastic Agent.
type ContextProviderComm interface {
	context.Context

	// Set sets the current mapping for this context.
	Set(map[string]interface{}) error
}

// ContextProvider is the interface that a context provider must implement.
type ContextProvider interface {
	// Run runs the context provider.
	Run(ContextProviderComm) error
}

// ContextProviderBuilder creates a new context provider based on the given config and returns it.
type ContextProviderBuilder func(config *config.Config) (ContextProvider, error)

// AddContextProvider adds a new ContextProviderBuilder
func (r *providerRegistry) AddContextProvider(name string, builder ContextProviderBuilder) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if name == "" {
		return fmt.Errorf("provider name is required")
	}
	if strings.ToLower(name) != name {
		return fmt.Errorf("provider name must be lowercase")
	}
	_, contextExists := r.contextProviders[name]
	_, dynamicExists := r.dynamicProviders[name]
	if contextExists || dynamicExists {
		return fmt.Errorf("provider '%s' is already registered", name)
	}
	if builder == nil {
		return fmt.Errorf("provider '%s' cannot be registered with a nil factory", name)
	}

	r.contextProviders[name] = builder
	r.logger.Debugf("Registered provider: %s", name)
	return nil
}

// GetContextProvider returns the context provider with the giving name, nil if it doesn't exist
func (r *providerRegistry) GetContextProvider(name string) (ContextProviderBuilder, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	b, ok := r.contextProviders[name]
	return b, ok
}
