// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hooks

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
)

type embeddedUninstaller interface {
	Uninstall(ctx context.Context, spec program.Spec, version, installDir string) error
}

// Uninstaller that executes PreUninstallSteps
type Uninstaller struct {
	uninstaller embeddedUninstaller
}

// NewUninstaller creates an uninstaller that executes PreUninstallSteps
func NewUninstaller(i embeddedUninstaller) (*Uninstaller, error) {
	return &Uninstaller{
		uninstaller: i,
	}, nil
}

// Uninstall performs the execution of the PreUninstallSteps
func (i *Uninstaller) Uninstall(ctx context.Context, spec program.Spec, version, installDir string) error {
	if spec.PreUninstallSteps != nil {
		return spec.PreUninstallSteps.Execute(ctx, installDir)
	}
	return i.uninstaller.Uninstall(ctx, spec, version, installDir)
}
