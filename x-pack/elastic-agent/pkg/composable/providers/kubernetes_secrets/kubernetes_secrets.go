// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes_secrets

import (
	"fmt"
	"time"

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
	composable.Providers.AddDynamicProvider("kubernetes_secrets", DynamicProviderBuilder)
}


type dynamicProviderSecrets struct {
	logger *logger.Logger
	config *Config
}

type eventWatcher struct {
	logger         *logger.Logger
	cleanupTimeout time.Duration
	comm           corecomp.DynamicProviderComm
}

// DynamicProviderBuilder builds the dynamic provider.
func DynamicProviderBuilder(logger *logger.Logger, c *config.Config) (corecomp.DynamicProvider, error) {
	var cfg Config
	if c == nil {
		c = config.New()
	}
	err := c.Unpack(&cfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack configuration")
	}
	return &dynamicProviderSecrets{logger, &cfg}, nil
}

func (p *dynamicProviderSecrets) Fetch(comm corecomp.DynamicProviderComm) string {
	fmt.Println("I FETCHEEED the secrets providerrrrr")
	return "someSecret"
}

// Run runs the environment context provider.
func (p *dynamicProviderSecrets) Run(comm corecomp.DynamicProviderComm) error {
	fmt.Println("I started the secrets providerrrrr")
	return nil
}
