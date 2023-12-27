// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package provider
package provider

import (
	"errors"
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	// ErrNotFound indicates an error where a provider was not found.
	ErrNotFound = errors.New("provider not found")
	// ErrExists indicates an error where a provider has already been registered.
	ErrExists = errors.New("provider already registered")
)

// Provider defines an interface TODO
type Provider interface {
	v2.InputManager // TODO
}

// FactoryFunc defines a factory function for creating a new Provider.
type FactoryFunc func(logger *logp.Logger) (Provider, error)

var (
	registry   = map[string]FactoryFunc{}
	registryMu sync.RWMutex
)

// Register will register the Provider with name and its factory function. An
// error is returned if the name has already been registered.
func Register(name string, factoryFn FactoryFunc) error {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[name]; exists {
		return ErrExists
	}

	registry[name] = factoryFn

	return nil
}

// Get returns the factory function for the Provider with name. Returns an error
// if the name hasn't been registered.
func Get(name string) (FactoryFunc, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	factoryFn, ok := registry[name]
	if !ok {
		return nil, ErrNotFound
	}

	return factoryFn, nil
}

// Has returns true if Provider with name has been registered.
func Has(name string) bool {
	registryMu.RLock()
	_, exists := registry[name]
	registryMu.RUnlock()

	return exists
}
