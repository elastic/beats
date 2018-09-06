// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/beatless/core"
)

// Config is a generic configured used by providers.
type Config struct {
	Functions []*common.Config `config:"functions"`
}

// DefaultProvider implements the minimal required to retrieve and start functions.
type DefaultProvider struct {
	config   *Config
	registry *Registry
	name     string
	log      *logp.Logger
}

// NewDefaultProvider returns factory methods to handle generic provider.
func NewDefaultProvider(name string) func(*logp.Logger, *Registry, *common.Config) (Provider, error) {
	return func(log *logp.Logger, registry *Registry, config *common.Config) (Provider, error) {
		c := &Config{}
		err := config.Unpack(c)
		if err != nil {
			return nil, err
		}
		return &DefaultProvider{config: c, registry: registry, name: name, log: log}, nil
	}
}

// Name returns the name of the provider.
func (d *DefaultProvider) Name() string {
	return d.name
}

// CreateFunctions takes factory method and returns runnable function.
func (d *DefaultProvider) CreateFunctions(clientFactory clientFactory) ([]core.Runner, error) {
	return CreateFunctions(d.registry, d, d.config.Functions, clientFactory)
}
