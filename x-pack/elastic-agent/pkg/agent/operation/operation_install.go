// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"os"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/state"
)

// operationInstall installs a artifact from predefined location
// skips if artifact is already installed
type operationInstall struct {
	logger         *logger.Logger
	program        Descriptor
	operatorConfig *config.Config
	installer      install.Installer
}

func newOperationInstall(
	logger *logger.Logger,
	program Descriptor,
	operatorConfig *config.Config,
	installer install.Installer) *operationInstall {

	return &operationInstall{
		logger:         logger,
		program:        program,
		operatorConfig: operatorConfig,
		installer:      installer,
	}
}

// Name is human readable name identifying an operation
func (o *operationInstall) Name() string {
	return "operation-install"
}

// Check checks whether install needs to be ran.
//
// If the installation directory already exists then it will not be ran.
func (o *operationInstall) Check(_ Application) (bool, error) {
	installDir := o.program.Directory()
	_, err := os.Stat(installDir)
	return os.IsNotExist(err), nil
}

// Run runs the operation
func (o *operationInstall) Run(ctx context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			application.SetState(state.Failed, err.Error())
		}
	}()

	return o.installer.Install(o.program.BinaryName(), o.program.Version(), o.program.Directory())
}
