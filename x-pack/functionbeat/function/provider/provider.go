// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/core"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/telemetry"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Create a new pipeline client based on the function configuration.
type clientFactory func(*conf.C) (pipeline.ISyncClient, error)

// Function is temporary
type Function interface {
	Run(context.Context, pipeline.ISyncClient, telemetry.T) error
	Name() string
}

// Provider providers the layer between functionbeat and cloud specific settings, its is responsable to
// return the function that need to be executed.
type Provider interface {
	CreateFunctions(clientFactory, []string) ([]core.Runner, error)
	FindFunctionByName(string) (Function, error)
	EnabledFunctions() ([]string, error)
	CLIManager() (CLIManager, error)
	TemplateBuilder() (TemplateBuilder, error)
	Name() string
}

// Runnable is the unit of work managed by the coordinator, anything related to the life of a function
// is encapsulated into the runnable.
type Runnable struct {
	config     *conf.C
	function   Function
	makeClient clientFactory
}

// Run call the the function's Run method, the method is a specific goroutine, it will block until
// beats shutdown or an error happen.
func (r *Runnable) Run(ctx context.Context, t telemetry.T) error {
	client, err := r.makeClient(r.config)
	if err != nil {
		return errors.Wrap(err, "could not create a client for the function")
	}
	defer client.Close()
	return r.function.Run(ctx, client, t)
}

func (r *Runnable) String() string {
	return r.function.Name()
}

// NewProvider return the provider specified in the configuration or an error.
func NewProvider(name string, cfg *conf.C) (Provider, error) {
	// Configure the provider, the provider will take care of the configuration for the
	// functions.
	registry := NewRegistry(feature.GlobalRegistry())
	providerFunc, err := registry.Lookup(name)
	if err != nil {
		return nil, fmt.Errorf("error finding the provider '%s', error: %v", name, err)
	}

	provider, err := providerFunc(logp.NewLogger("provider"), registry, cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating the provider '%s', error: %v", name, err)
	}

	return provider, nil
}

// IsAvailable checks if a cloud provider is available in the binary.
func IsAvailable(name string) (bool, error) {
	registry := NewRegistry(feature.GlobalRegistry())

	availableProviders, err := registry.AvailableProviders()
	if err != nil {
		return false, err
	}

	for _, p := range availableProviders {
		if p == name {
			return true, nil
		}
	}
	return false, nil
}

// ListFunctions returns the list of enabled function names.
func ListFunctions(provider string) ([]string, error) {
	functions, err := feature.GlobalRegistry().LookupAll(getNamespace(provider))
	if err != nil {
		return nil, err
	}

	names := make([]string, len(functions))
	for i, f := range functions {
		names[i] = f.Name()
	}
	return names, nil
}

// Create returns the provider from a configuration.
func Create(cfg *conf.C) (Provider, error) {
	providers, err := List()
	if err != nil {
		return nil, err
	}
	if len(providers) != 1 {
		return nil, fmt.Errorf("too many providers are available, expected one, got: %s", providers)
	}

	providerCfg, err := cfg.Child(providers[0], -1)
	if err != nil {
		return nil, err
	}

	return NewProvider(providers[0], providerCfg)
}

// List returns the list of available providers.
func List() ([]string, error) {
	registry := NewRegistry(feature.GlobalRegistry())
	return registry.AvailableProviders()
}
