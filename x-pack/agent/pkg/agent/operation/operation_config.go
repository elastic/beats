// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

var (
	// ErrClientNotFound is an error when client is not found
	ErrClientNotFound = errors.New("client not found, check if process is running")
	// ErrClientNotConfigurable happens when stored client does not implement Config func
	ErrClientNotConfigurable = errors.New("client does not provide configuration")
)

// Configures running process by sending a configuration to its
// grpc endpoint
type operationConfig struct {
	logger         *logger.Logger
	operatorConfig *config.Config
	cfg            map[string]interface{}
	eventProcessor callbackHooks
}

func newOperationConfig(
	logger *logger.Logger,
	operatorConfig *config.Config,
	cfg map[string]interface{},
	eventProcessor callbackHooks) *operationConfig {
	return &operationConfig{
		logger:         logger,
		operatorConfig: operatorConfig,
		cfg:            cfg,
		eventProcessor: eventProcessor,
	}
}

// Name is human readable name identifying an operation
func (o *operationConfig) Name() string {
	return "operation-config"
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationConfig) Check() (bool, error) { return true, nil }

// Run runs the operation
func (o *operationConfig) Run(ctx context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			err = errors.New(err,
				o.Name(),
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, application.Name()))
			o.eventProcessor.OnFailing(ctx, application.Name(), err)
		}
	}()
	return application.Configure(ctx, o.cfg)
}
