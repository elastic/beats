// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package snapshot

import (
	"fmt"
)

const snapshotIdentifier = "-SNAPSHOT"

// embeddedInstaller is an interface allowing installation of an artifact
type embeddedInstaller interface {
	// Install installs an artifact and returns
	// location of the installed program
	// error if something went wrong
	Install(programName, version, installDir string) error
}

// Installer or zip packages
type Installer struct {
	installer embeddedInstaller
}

// NewInstaller creates an installer able to install zip packages
func NewInstaller(installer embeddedInstaller) (*Installer, error) {
	return &Installer{
		installer: installer,
	}, nil
}

// Install performs installation of program in a specific version.
// It expects package to be already downloaded.
func (i *Installer) Install(programName, version, installDir string) error {
	version = fmt.Sprintf("%s-SNAPSHOT", version)
	return i.installer.Install(programName, version, installDir)
}
