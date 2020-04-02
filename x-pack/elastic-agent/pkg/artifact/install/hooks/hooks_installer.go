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

// Installer or zip packages
type Installer struct {
	installer embeddedInstaller
}

// NewInstaller creates an installer able to install zip packages
func NewInstaller(i embeddedInstaller) (*Installer, error) {
	return &Installer{
		installer: i,
	}, nil
}

// Install performs installation of program in a specific version.
// It expects package to be already downloaded.
func (i *Installer) Install(programName, version, installDir string) error {
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
