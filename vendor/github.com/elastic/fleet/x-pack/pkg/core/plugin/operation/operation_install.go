// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"os"

	"github.com/elastic/fleet/x-pack/pkg/artifact/install"
	"github.com/pkg/errors"
	"github.com/urso/ecslog"
)

// operationInstall installs a artifact from predefined location
// skips if artifact is already installed
type operationInstall struct {
	logger         *ecslog.Logger
	program        Program
	operatorConfig *Config
	installer      install.Installer
}

func newOperationInstall(
	logger *ecslog.Logger,
	program Program,
	operatorConfig *Config,
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

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationInstall) Check() (bool, error) {
	installDir := o.program.Directory(o.operatorConfig.DownloadConfig)
	_, err := os.Stat(installDir)
	return os.IsNotExist(err), nil
}

// Run runs the operation
func (o *operationInstall) Run() (err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, o.Name())
		}
	}()

	return o.installer.Install(o.program.BinaryName(), o.program.Version(), o.program.Directory(o.operatorConfig.DownloadConfig))
}
