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

// DynamicProviderComm is the interface that an dynamic provider uses to communicate back to Elastic Agent.
type DynamicProviderComm interface {
	context.Context

	// AddOrUpdate updates a mapping with given ID with latest mapping and processors.
	AddOrUpdate(id string, mapping map[string]interface{}, processors []map[string]interface{}) error
	// Remove removes a mapping by given ID.
	Remove(id string)
}

// DynamicProvider is the interface that a dynamic provider must implement.
type DynamicProvider interface {
	// Run runs the inventory provider.
	Run(DynamicProviderComm) error
}

// DynamicProviderBuilder creates a new dynamic provider based on the given config and returns it.
type DynamicProviderBuilder func(config *config.Config) (DynamicProvider, error)

// AddDynamicProvider adds a new DynamicProviderBuilder
func (r *providerRegistry) AddDynamicProvider(name string, builder DynamicProviderBuilder) error {
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

	r.dynamicProviders[name] = builder
	r.logger.Debugf("Registered provider: %s", name)
	return nil
}

// GetDynamicProvider returns the dynamic provider with the giving name, nil if it doesn't exist
func (r *providerRegistry) GetDynamicProvider(name string) (DynamicProviderBuilder, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	b, ok := r.dynamicProviders[name]
	return b, ok
}
