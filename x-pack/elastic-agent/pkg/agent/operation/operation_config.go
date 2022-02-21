// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

var (
	// ErrClientNotFound is an error when client is not found
	ErrClientNotFound = errors.New("client not found, check if process is running")
	// ErrClientNotConfigurable happens when stored client does not implement Config func
	ErrClientNotConfigurable = errors.New("client does not provide configuration")
)

// perhaps here
// Configures running process by sending a configuration to its
// grpc endpoint
type operationConfig struct {
	logger         *logger.Logger
	operatorConfig *configuration.SettingsConfig
	cfg            map[string]interface{}
}

func newOperationConfig(
	logger *logger.Logger,
	operatorConfig *configuration.SettingsConfig,
	cfg map[string]interface{}) *operationConfig {
	return &operationConfig{
		logger:         logger,
		operatorConfig: operatorConfig,
		cfg:            cfg,
	}
}

// Name is human readable name identifying an operation
func (o *operationConfig) Name() string {
	return "operation-config"
}

// Check checks whether config needs to be run.
//
// Always returns true.
func (o *operationConfig) Check(_ context.Context, _ Application) (bool, error) { return true, nil }

// Run runs the operation
func (o *operationConfig) Run(ctx context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			// application failed to apply config but is running.
			s := state.Degraded
			if errors.Is(err, process.ErrAppNotRunning) {
				s = state.Failed
			}

			application.SetState(s, err.Error(), nil)
		}
	}()
	return application.Configure(ctx, o.cfg)
}
