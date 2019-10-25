// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/app"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/process"
)

var (
	// ErrUpdateClientFailed happens when call to a client vault returns an error.
	ErrUpdateClientFailed = errors.New("updating clientvault failed")
	// ErrStoringReattachInfoFailed happens when call to reattach collection fails
	// might be related to filesystem. Check logs for more information.
	ErrStoringReattachInfoFailed = errors.New("backing up reattach information failed")
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
func (o *operationStart) Run(application Application) (err error) {
	o.eventProcessor.OnStarting(application.Name())
	defer func() {
		if err != nil {
			// kill the process if something failed
			err = errors.Wrap(err, o.Name())
			o.eventProcessor.OnFailing(application.Name(), err)
		} else {
			o.eventProcessor.OnRunning(application.Name())
		}
	}()

	return application.Start(o.cfg)
}
