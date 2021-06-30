// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hooks

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
)

type embeddedInstaller interface {
	Install(ctx context.Context, spec program.Spec, version, installDir string) error
}

type embeddedChecker interface {
	Check(ctx context.Context, spec program.Spec, version, installDir string) error
}

// InstallerChecker runs the PostInstallSteps after running the embedded installer
// and runs the InstallerCheckSteps after running the embedded installation checker.
type InstallerChecker struct {
	installer embeddedInstaller
	checker   embeddedChecker
}

// NewInstallerChecker creates a new InstallerChecker
func NewInstallerChecker(i embeddedInstaller, c embeddedChecker) (*InstallerChecker, error) {
	return &InstallerChecker{
		installer: i,
		checker:   c,
	}, nil
}

// Install performs installation of program in a specific version, then runs the
// PostInstallSteps for the program if defined.
func (i *InstallerChecker) Install(ctx context.Context, spec program.Spec, version, installDir string) error {
	if err := i.installer.Install(ctx, spec, version, installDir); err != nil {
		return err
	}
	if spec.PostInstallSteps != nil {
		return spec.PostInstallSteps.Execute(ctx, installDir)
	}
	return nil
}

// Check performs installation check of program to ensure that it is already installed, then
// runs the InstallerCheckSteps to ensure that the installation is valid.
func (i *InstallerChecker) Check(ctx context.Context, spec program.Spec, version, installDir string) error {
	err := i.checker.Check(ctx, spec, version, installDir)
	if err != nil {
		return err
	}
	if spec.CheckInstallSteps != nil {
		return spec.CheckInstallSteps.Execute(ctx, installDir)
	}
	return nil
}
