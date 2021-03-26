// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes_secrets

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	corecomp "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	// PodPriority is the priority that pod mappings are added to the provider.
	PodPriority = 0
	// ContainerPriority is the priority that container mappings are added to the provider.
	ContainerPriority = 1
)

func init() {
	composable.Providers.AddContextProvider("kubernetes_secrets", ContextProviderBuilder)
}


type contextProviderK8sSecrets struct {
	logger *logger.Logger
	config *Config
}

// DynamicProviderBuilder builds the dynamic provider.
func ContextProviderBuilder(logger *logger.Logger, c *config.Config) (corecomp.ContextProvider, error) {
	var cfg Config
	if c == nil {
		c = config.New()
	}
	err := c.Unpack(&cfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack configuration")
	}
	return &contextProviderK8sSecrets{logger, &cfg}, nil
}

func (p *contextProviderK8sSecrets) Fetch(value string) (string, error) {
	// TODO: add actual call to k8s api here to get the secret
	return "someSecret42", nil
}

// Run runs the k8s secrets context provider.
func (p *contextProviderK8sSecrets) Run(comm corecomp.ContextProviderComm) error {
	return nil
}
