// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composable

import "context"

// FetchContextProvider is the interface that a context provider uses so as to be able to be called
// explicitely on demand by vars framework in order to fetch specific target values like a k8s secret.
type FetchContextProvider interface {
	ContextProvider
	// Run runs the inventory provider.
	Fetch(string) (string, bool)
}

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
