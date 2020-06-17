// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hooks

import (
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
)

type embeddedUninstaller interface {
	Uninstall(programName, version, installDir string) error
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
func (i *Uninstaller) Uninstall(programName, version, installDir string) error {
	// pre uninstall hooks
	nameLower := strings.ToLower(programName)
	_, isSupported := program.SupportedMap[nameLower]
	if !isSupported {
		return nil
	}

	for _, spec := range program.Supported {
		if strings.ToLower(spec.Name) != nameLower {
			continue
		}

		if spec.PreUninstallSteps != nil {
			return spec.PreUninstallSteps.Execute(installDir)
		}

		// only one spec for type
		break
	}

	if err := i.uninstaller.Uninstall(programName, version, installDir); err != nil {
		return err
	}

	return nil
}
