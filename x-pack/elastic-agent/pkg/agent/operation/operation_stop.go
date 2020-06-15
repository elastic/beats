// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

// operationStop stops the running process
// skips if process is already skipped
type operationStop struct {
	logger         *logger.Logger
	operatorConfig *config.Config
}

func newOperationStop(
	logger *logger.Logger,
	operatorConfig *config.Config) *operationStop {
	return &operationStop{
		logger:         logger,
		operatorConfig: operatorConfig,
	}
}

// Name is human readable name identifying an operation
func (o *operationStop) Name() string {
	return "operation-stop"
}

// Check checks whether application needs to be stopped.
//
// If the application state is not stopped then stop should be performed.
func (o *operationStop) Check(application Application) (bool, error) {
	if application.State().Status != state.Stopped {
		return true, nil
	}
	return false, nil
}

// Run runs the operation
func (o *operationStop) Run(ctx context.Context, application Application) (err error) {
	application.Stop()
	return nil
}
