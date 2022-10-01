// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package dns

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
)

// ImplFactory is a factory method for DNS monitoring implementations.
type ImplFactory func(mb.BaseMetricSet, *logp.Logger) (Sniffer, error)

type implRegistry struct {
	byName map[string]ImplFactory
}

// Registry contains the registry of dns monitoring implementations
var Registry implRegistry

// Register registers a new DNS monitoring implementation.
func (r *implRegistry) Register(name string, factory ImplFactory) error {
	if _, found := r.byName[name]; found {
		return fmt.Errorf("dns monitoring implementation '%s' already registered", name)
	}
	if r.byName == nil {
		r.byName = make(map[string]ImplFactory)
	}
	r.byName[name] = factory
	return nil
}

// MustRegister registers a new implementation and panics in case of an error.
func (r *implRegistry) MustRegister(name string, factory ImplFactory) {
	if err := r.Register(name, factory); err != nil {
		panic(err)
	}
}

// Get returns a dns monitoring implementation by name.
func (r *implRegistry) Get(name string) (ImplFactory, error) {
	factory, found := r.byName[name]
	if !found {
		return nil, fmt.Errorf("no such dns monitoring implementation: '%s'", name)
	}
	return factory, nil
}
