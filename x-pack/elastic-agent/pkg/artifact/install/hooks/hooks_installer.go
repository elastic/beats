// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hooks

import (
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
)

type embeddedInstaller interface {
	Install(programName, version, installDir string) error
}

type embeddedChecker interface {
	Check(programName, version, installDir string) error
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
func (i *InstallerChecker) Install(programName, version, installDir string) error {
	if err := i.installer.Install(programName, version, installDir); err != nil {
		return err
	}

	// post install hooks
	nameLower := strings.ToLower(programName)
	_, isSupported := program.SupportedMap[nameLower]
	if !isSupported {
		return nil
	}

	for _, spec := range program.Supported {
		if strings.ToLower(spec.Name) != nameLower {
			continue
		}

		if spec.PostInstallSteps != nil {
			return spec.PostInstallSteps.Execute(installDir)
		}

		// only one spec for type
		break
	}

	return nil
}

// Check performs installation check of program to ensure that it is already installed, then
// runs the InstallerCheckSteps to ensure that the installation is valid.
func (i *InstallerChecker) Check(programName, version, installDir string) error {
	err := i.checker.Check(programName, version, installDir)
	if err != nil {
		return err
	}

	// installer check steps
	nameLower := strings.ToLower(programName)
	_, isSupported := program.SupportedMap[nameLower]
	if !isSupported {
		return nil
	}

	for _, spec := range program.Supported {
		if strings.ToLower(spec.Name) != nameLower {
			continue
		}

		if spec.InstallCheckSteps != nil {
			return spec.InstallCheckSteps.Execute(installDir)
		}

		// only one spec for type
		break
	}

	return nil
}
