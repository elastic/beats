// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/uninstall"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

// operationUninstall uninstalls a artifact from predefined location
type operationUninstall struct {
	logger      *logger.Logger
	program     Descriptor
	uninstaller uninstall.Uninstaller
}

func newOperationUninstall(
	logger *logger.Logger,
	program Descriptor,
	uninstaller uninstall.Uninstaller) *operationUninstall {

	return &operationUninstall{
		logger:      logger,
		program:     program,
		uninstaller: uninstaller,
	}
}

// Name is human readable name identifying an operation
func (o *operationUninstall) Name() string {
	return "operation-uninstall"
}

// Check checks whether uninstall needs to be ran.
//
// Always true.
func (o *operationUninstall) Check(_ context.Context, _ Application) (bool, error) {
	return true, nil
}

// Run runs the operation
func (o *operationUninstall) Run(ctx context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			application.SetState(state.Failed, err.Error(), nil)
		}
	}()

	return o.uninstaller.Uninstall(ctx, o.program.BinaryName(), o.program.Version(), o.program.Directory())
}
