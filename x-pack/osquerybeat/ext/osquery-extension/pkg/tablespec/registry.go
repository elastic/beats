// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespec

import (
	"fmt"
	"sync"
)

var (
	once           sync.Once
	globalRegistry *registry
)

// TableRegistry manages the registration and discovery of extension tables.
type TableRegistry interface {
	// Register adds a table spec to the registry
	Register(spec *TableSpec) error

	// Get retrieves a table spec by name
	Get(name string) (*TableSpec, bool)

	// List returns all registered table specs
	List() []*TableSpec

	// ListByPlatform returns table specs that support the given platform
	ListByPlatform(platform string) []*TableSpec
}

// registry is a thread-safe implementation of TableRegistry
type registry struct {
	mu     sync.RWMutex
	tables map[string]*TableSpec
}

// NewRegistry creates a new TableRegistry instance
func newRegistry() *registry {
	return &registry{
		tables: make(map[string]*TableSpec),
	}
}

// MustRegister registers a table spec and panics if registration fails.
// This is intended for use during package initialization.
func MustRegister(spec *TableSpec) {
	if err := GetGlobalRegistry().Register(spec); err != nil {
		panic(err)
	}
}

func GetGlobalRegistry() TableRegistry {
	once.Do(func() {
		globalRegistry = newRegistry()
	})
	return globalRegistry
}

// Register adds a table spec to the registry
func (r *registry) Register(spec *TableSpec) error {
	if spec == nil {
		return fmt.Errorf("cannot register nil table spec")
	}

	name := spec.Name()
	if name == "" {
		return fmt.Errorf("table spec must have a non-empty name")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tables[name]; exists {
		return fmt.Errorf("table %q is already registered", name)
	}

	r.tables[name] = spec
	return nil
}

// Get retrieves a table spec by name
func (r *registry) Get(name string) (*TableSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	spec, ok := r.tables[name]
	return spec, ok
}

// List returns all registered table specs
func (r *registry) List() []*TableSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()

	specs := make([]*TableSpec, 0, len(r.tables))
	for _, spec := range r.tables {
		specs = append(specs, spec)
	}
	return specs
}

// ListByPlatform returns table specs that support the given platform
func (r *registry) ListByPlatform(platform string) []*TableSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()

	specs := make([]*TableSpec, 0)
	for _, spec := range r.tables {
		for _, p := range spec.Platforms() {
			if p == platform {
				specs = append(specs, spec)
				break
			}
		}
	}
	return specs
}
