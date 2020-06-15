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
)

// operationStop stops the running process
// skips if process is already skipped
type operationStop struct {
	logger         *logger.Logger
	operatorConfig *config.Config
	eventProcessor callbackHooks
}

func newOperationStop(
	logger *logger.Logger,
	operatorConfig *config.Config,
	eventProcessor callbackHooks) *operationStop {
	return &operationStop{
		logger:         logger,
		operatorConfig: operatorConfig,
		eventProcessor: eventProcessor,
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
	o.eventProcessor.OnStopping(ctx, application.Name())
	defer func() {
		if err != nil {
			err = errors.New(err,
				o.Name(),
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, application.Name()))
			o.eventProcessor.OnFailing(ctx, application.Name(), err)
		} else {
			o.eventProcessor.OnStopped(ctx, application.Name())
		}
	}()

	application.Stop()
	return nil
}
