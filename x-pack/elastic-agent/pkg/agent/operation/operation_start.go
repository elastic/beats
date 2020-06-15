// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/state"
)

// operationStart start installed process
// skips if process is already running
type operationStart struct {
	logger         *logger.Logger
	program        Descriptor
	operatorConfig *config.Config
	cfg            map[string]interface{}

	pi *process.Info
}

func newOperationStart(
	logger *logger.Logger,
	program Descriptor,
	operatorConfig *config.Config,
	cfg map[string]interface{}) *operationStart {
	// TODO: make configurable

	return &operationStart{
		logger:         logger,
		program:        program,
		operatorConfig: operatorConfig,
		cfg:            cfg,
	}
}

// Name is human readable name identifying an operation
func (o *operationStart) Name() string {
	return "operation-start"
}

// Check checks whether application needs to be started.
//
// Only starts the application when in stopped state, any other state
// and the application is handled by the life cycle inside of the `Application`
// implementation.
func (o *operationStart) Check(application Application) (bool, error) {
	if application.State().Status == state.Stopped {
		return true, nil
	}
	return false, nil
}

// Run runs the operation
func (o *operationStart) Run(ctx context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			application.SetState(state.Failed, err.Error())
		}
	}()

	return application.Start(ctx, o.program, o.cfg)
}
