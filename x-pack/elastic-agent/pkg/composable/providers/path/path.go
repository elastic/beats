// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package path

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

func init() {
	composable.Providers.AddContextProvider("path", ContextProviderBuilder)
}

type contextProvider struct{}

// Run runs the Agent context provider.
func (*contextProvider) Run(comm composable.ContextProviderComm) error {
	err := comm.Set(map[string]interface{}{
		"home":   paths.Home(),
		"data":   paths.Data(),
		"config": paths.Config(),
		"logs":   paths.Logs(),
	})
	if err != nil {
		return errors.New(err, "failed to set mapping", errors.TypeUnexpected)
	}
	return nil
}

// ContextProviderBuilder builds the context provider.
func ContextProviderBuilder(_ *config.Config) (composable.ContextProvider, error) {
	return &contextProvider{}, nil
}
