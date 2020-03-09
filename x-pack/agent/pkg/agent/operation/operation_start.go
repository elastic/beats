// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/process"
)

// operationStart start installed process
// skips if process is already running
type operationStart struct {
	program        app.Descriptor
	logger         *logger.Logger
	operatorConfig *config.Config
	cfg            map[string]interface{}
	eventProcessor callbackHooks

	pi *process.Info
}

func newOperationStart(
	logger *logger.Logger,
	operatorConfig *config.Config,
	cfg map[string]interface{},
	eventProcessor callbackHooks) *operationStart {
	// TODO: make configurable

	return &operationStart{
		logger:         logger,
		operatorConfig: operatorConfig,
		cfg:            cfg,
		eventProcessor: eventProcessor,
	}
}

// Name is human readable name identifying an operation
func (o *operationStart) Name() string {
	return "operation-start"
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationStart) Check() (bool, error) {
	// TODO: get running processes and compare hashes

	return true, nil
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

	return application.Start(ctx, o.cfg)
}
