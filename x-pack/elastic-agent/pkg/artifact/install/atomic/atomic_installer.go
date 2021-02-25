// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package atomic

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/hashicorp/go-multierror"
)

type embeddedInstaller interface {
	Install(ctx context.Context, spec program.Spec, version, installDir string) error
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
func (i *Installer) Install(ctx context.Context, spec program.Spec, version, installDir string) error {
	// tar installer uses Dir of installDir to determine location of unpack
	tempDir, err := ioutil.TempDir(paths.TempDir(), "elastic-agent-install")
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

	if err := i.installer.Install(ctx, spec, version, tempInstallDir); err != nil {
		// cleanup unfinished install
		if rerr := os.RemoveAll(tempInstallDir); rerr != nil {
			err = multierror.Append(err, rerr)
		}
		return err
	}

	if err := os.Rename(tempInstallDir, installDir); err != nil {
		if rerr := os.RemoveAll(installDir); rerr != nil {
			err = multierror.Append(err, rerr)
		}
		if rerr := os.RemoveAll(tempInstallDir); rerr != nil {
			err = multierror.Append(err, rerr)
		}
		return err
	}

	// on windows rename is not atomic, let's force it to flush the cache
	if runtime.GOOS == "windows" {
		if f, err := os.OpenFile(installDir, os.O_SYNC|os.O_RDWR, 0755); err == nil {
			f.Sync()
		}
	}

	return nil
}
