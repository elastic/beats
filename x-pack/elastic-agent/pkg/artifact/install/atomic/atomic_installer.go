// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package atomic

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
)

type embeddedInstaller interface {
	Install(ctx context.Context, programName, version, installDir string) error
}

// Installer installs into temporary destination and moves to correct one after
// successful finish.
type Installer struct {
	installer embeddedInstaller
}

// NewInstaller creates a new AtomicInstaller
func NewInstaller(i embeddedInstaller) (*Installer, error) {
	return &Installer{
		installer: i,
	}, nil
}

// Install performs installation of program in a specific version.
func (i *Installer) Install(ctx context.Context, programName, version, installDir string) error {
	// tar installer uses Dir of installDir to determine location of unpack
	tempDir, err := ioutil.TempDir(os.TempDir(), "elastic-agent-install")
	if err != nil {
		return err
	}
	tempInstallDir := filepath.Join(tempDir, filepath.Base(installDir))

	// cleanup install directory before Install
	if _, err := os.Stat(installDir); err == nil || os.IsExist(err) {
		os.RemoveAll(installDir)
	}

	if _, err := os.Stat(tempInstallDir); err == nil || os.IsExist(err) {
		os.RemoveAll(tempInstallDir)
	}

	if err := i.installer.Install(ctx, programName, version, tempInstallDir); err != nil {
		// cleanup unfinished install
		os.RemoveAll(tempInstallDir)
		return err
	}

	if err := os.Rename(tempInstallDir, installDir); err != nil {
		os.RemoveAll(installDir)
		os.RemoveAll(tempInstallDir)
		return err
	}

	return nil
}
