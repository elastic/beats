// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package local

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

func init() {
	composable.Providers.AddContextProvider("local", ContextProviderBuilder)
}

type contextProvider struct {
	Mapping map[string]interface{} `config:"vars"`
}

// Run runs the environment context provider.
func (c *contextProvider) Run(comm composable.ContextProviderComm) error {
	err := comm.Set(c.Mapping)
	if err != nil {
		return errors.New(err, "failed to set mapping", errors.TypeUnexpected)
	}
	return nil
}

// ContextProviderBuilder builds the context provider.
func ContextProviderBuilder(c *config.Config) (composable.ContextProvider, error) {
	p := &contextProvider{}
	if c != nil {
		err := c.Unpack(p)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack vars: %s", err)
		}
	}
	if p.Mapping == nil {
		p.Mapping = map[string]interface{}{}
	}
	return p, nil
}
