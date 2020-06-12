// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/state"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/process"
)

// operationStart start installed process
// skips if process is already running
type operationStart struct {
	logger         *logger.Logger
	program        Descriptor
	operatorConfig *config.Config
	cfg            map[string]interface{}
	eventProcessor callbackHooks

	pi *process.Info
}

func newOperationStart(
	logger *logger.Logger,
	program Descriptor,
	operatorConfig *config.Config,
	cfg map[string]interface{},
	eventProcessor callbackHooks) *operationStart {
	// TODO: make configurable

	return &operationStart{
		logger:         logger,
		program:        program,
		operatorConfig: operatorConfig,
		cfg:            cfg,
		eventProcessor: eventProcessor,
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
	o.eventProcessor.OnStarting(ctx, application.Name())
	defer func() {
		if err != nil {
			// kill the process if something failed
			err = errors.New(err,
				o.Name(),
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, application.Name()))
			o.eventProcessor.OnFailing(ctx, application.Name(), err)
		} else {
			o.eventProcessor.OnRunning(ctx, application.Name())
		}
	}()

	return application.Start(ctx, o.program, o.cfg)
}
