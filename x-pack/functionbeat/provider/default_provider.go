// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/config"
	"github.com/elastic/beats/x-pack/functionbeat/core"
)

// DefaultProvider implements the minimal required to retrieve and start functions.
type DefaultProvider struct {
	rawConfig      *common.Config
	config         *config.ProviderConfig
	registry       *Registry
	name           string
	log            *logp.Logger
	managerFactory CLIManagerFactory
}

// NewDefaultProvider returns factory methods to handle generic provider.
func NewDefaultProvider(name string, manager CLIManagerFactory) func(*logp.Logger, *Registry, *common.Config) (Provider, error) {
	return func(log *logp.Logger, registry *Registry, cfg *common.Config) (Provider, error) {
		c := &config.ProviderConfig{}
		err := cfg.Unpack(c)
		if err != nil {
			return nil, err
		}

		if manager == nil {
			manager = NewNullCli
		}

		return &DefaultProvider{
			rawConfig:      cfg,
			config:         c,
			registry:       registry,
			name:           name,
			log:            log,
			managerFactory: manager,
		}, nil
	}
}

// Name returns the name of the provider.
func (d *DefaultProvider) Name() string {
	return d.name
}

// CreateFunctions takes factory method and returns runnable function.
func (d *DefaultProvider) CreateFunctions(clientFactory clientFactory, enabledFunctions []string) ([]core.Runner, error) {
	return CreateFunctions(d.registry, d, enabledFunctions, d.config.Functions, clientFactory)
}

// FindFunctionByName returns a function instance identified by a unique name or an error if not found.
func (d *DefaultProvider) FindFunctionByName(name string) (Function, error) {
	return FindFunctionByName(d.registry, d, d.config.Functions, name)
}

// CLIManager returns the type responsable of installing, updating and removing remote function
// for a specific provider.
func (d *DefaultProvider) CLIManager() (CLIManager, error) {
	return d.managerFactory(nil, d.rawConfig, d)
}

// nullCLI is used when a provider doesn't implement the CLI to manager functions on the service provider.
type nullCLI struct{}

// NewNullCli returns a NOOP CliManager.
func NewNullCli(_ *logp.Logger, _ *common.Config, _ Provider) (CLIManager, error) {
	return (*nullCLI)(nil), nil
}

func (*nullCLI) Deploy(_ string) error { return fmt.Errorf("deploy not implemented") }
func (*nullCLI) Update(_ string) error { return fmt.Errorf("update not implemented") }
func (*nullCLI) Remove(_ string) error { return fmt.Errorf("remove not implemented") }
