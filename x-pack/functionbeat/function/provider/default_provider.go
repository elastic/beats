// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/functionbeat/config"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/core"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// DefaultProvider implements the minimal required to retrieve and start functions.
type DefaultProvider struct {
	rawConfig       *conf.C
	config          *config.ProviderConfig
	registry        *Registry
	name            string
	log             *logp.Logger
	managerFactory  CLIManagerFactory
	templateFactory TemplateBuilderFactory
}

// NewDefaultProvider returns factory methods to handle generic provider.
func NewDefaultProvider(
	name string,
	manager CLIManagerFactory,
	templater TemplateBuilderFactory,
) func(*logp.Logger, *Registry, *conf.C) (Provider, error) {
	return func(log *logp.Logger, registry *Registry, cfg *conf.C) (Provider, error) {
		c := &config.ProviderConfig{}
		err := cfg.Unpack(c)
		if err != nil {
			return nil, err
		}

		if manager == nil {
			manager = NewNullCli
		}

		return &DefaultProvider{
			rawConfig:       cfg,
			config:          c,
			registry:        registry,
			name:            name,
			log:             log,
			managerFactory:  manager,
			templateFactory: templater,
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

// TemplateBuilder returns a TemplateBuilder returns a the type responsible to generate templates.
func (d *DefaultProvider) TemplateBuilder() (TemplateBuilder, error) {
	return d.templateFactory(d.log, d.rawConfig, d)
}

// EnabledFunctions return the list of enabled funcionts.
func (d *DefaultProvider) EnabledFunctions() ([]string, error) {
	return EnabledFunctions(d.registry, d, d.config.Functions)
}

// nullCLI is used when a provider doesn't implement the CLI to manager functions on the service provider.
type nullCLI struct{}

// NewNullCli returns a NOOP CliManager.
func NewNullCli(_ *logp.Logger, _ *conf.C, _ Provider) (CLIManager, error) {
	return (*nullCLI)(nil), nil
}

func (*nullCLI) Deploy(_ string) error  { return fmt.Errorf("deploy not implemented") }
func (*nullCLI) Update(_ string) error  { return fmt.Errorf("update not implemented") }
func (*nullCLI) Remove(_ string) error  { return fmt.Errorf("remove not implemented") }
func (*nullCLI) Export(_ string) error  { return fmt.Errorf("export not implemented") }
func (*nullCLI) Package(_ string) error { return fmt.Errorf("package not implemented") }

// nullTemplateBuilder is used when a provider does not implement a template builder functionality.
type nullTemplateBuilder struct{}

// NewNullTemplateBuilder returns a NOOP TemplateBuilder.
func NewNullTemplateBuilder(_ *logp.Logger, _ *conf.C, _ Provider) (TemplateBuilder, error) {
	return (*nullTemplateBuilder)(nil), nil
}

// RawTemplate returns a empty string.
func (*nullTemplateBuilder) RawTemplate(_ string) (string, error) {
	return "", fmt.Errorf("raw temaplate not implemented")
}
