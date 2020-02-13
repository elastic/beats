// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"os"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/x-pack/agent/pkg/artifact/install"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

// operationInstall installs a artifact from predefined location
// skips if artifact is already installed
type operationInstall struct {
	logger         *logger.Logger
	program        Descriptor
	operatorConfig *config.Config
	installer      install.Installer
	eventProcessor callbackHooks
}

func newOperationInstall(
	logger *logger.Logger,
	program Descriptor,
	operatorConfig *config.Config,
	installer install.Installer,
	eventProcessor callbackHooks) *operationInstall {

	return &operationInstall{
		logger:         logger,
		program:        program,
		operatorConfig: operatorConfig,
		installer:      installer,
		eventProcessor: eventProcessor,
	}
}

// Name is human readable name identifying an operation
func (o *operationInstall) Name() string {
	return "operation-install"
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationInstall) Check() (bool, error) {
	installDir := o.program.Directory()
	_, err := os.Stat(installDir)
	return os.IsNotExist(err), nil
}

// Run runs the operation
func (o *operationInstall) Run(ctx context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			err = errors.New(err,
				o.Name(),
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, application.Name()))
			o.eventProcessor.OnFailing(ctx, application.Name(), err)
		}
	}()

	return o.installer.Install(o.program.BinaryName(), o.program.Version(), o.program.Directory())
}
