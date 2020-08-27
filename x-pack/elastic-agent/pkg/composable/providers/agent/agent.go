// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package agent

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

func init() {
	composable.Providers.AddContextProvider("agent", ContextProviderBuilder)
}

type contextProvider struct{}

// Run runs the Agent context provider.
func (*contextProvider) Run(comm composable.ContextProviderComm) error {
	a, err := info.NewAgentInfo()
	if err != nil {
		return err
	}
	err = comm.Set(map[string]interface{}{
		"id": a.AgentID(),
		"version": map[string]interface{}{
			"version":    release.Version(),
			"commit":     release.Commit(),
			"build_time": release.BuildTime().Format("2006-01-02 15:04:05 -0700 MST"),
			"snapshot":   release.Snapshot(),
		},
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
