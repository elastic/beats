// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/fileutil"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install"
)

func installOsquery(ctx context.Context) error {
	exefp, err := os.Executable()
	if err != nil {
		return err
	}
	exedir := filepath.Dir(exefp)

	// Install osqueryd if needed
	return installOsqueryWithDir(ctx, exedir)
}

func installOsqueryWithDir(ctx context.Context, dir string) error {
	log := logp.NewLogger("osqueryd_install").With("dir", dir)
	log.Info("Check if osqueryd needs to be installed")

	fn := distro.OsquerydDistroFilename()
	var installFunc func(context.Context, string, string, bool) error

	if runtime.GOOS == "darwin" {
		installFunc = install.InstallFromPkg
	}

	installing := false
	ilog := log.With("file", fn)
	if installFunc != nil {
		exists, err := fileutil.FileExists(fn)
		if err != nil {
			ilog.Errorf("Failed to access the install package file, error: %v", err)
			return err
		}
		if exists {
			ilog.Info("Found install package file, installing")
			err = installFunc(ctx, fn, dir, true)
			if err != nil {
				ilog.Errorf("Failed to extract from install package, error: %v", err)
				return err
			}
			installing = true
		} else {
			ilog.Info("Install package doesn't exists, nothing to install")
		}
	}

	if installing {
		// Check that osqueryd file is now installed
		osqfn := distro.OsquerydFilename()
		if runtime.GOOS == "darwin" {
			osqfn = distro.OsquerydDarwinApp()
		}
		flog := log.With("file", osqfn)
		exists, err := fileutil.FileExists(osqfn)
		if err != nil {
			flog.Errorf("Failed to access the file, error: %v", err)
			return err
		}
		if exists {
			flog.Info("File found")
		} else {
			flog.Error("File is not found after install")
			return os.ErrNotExist
		}

		if derr := os.Remove(fn); derr != nil {
			ilog.Warn("Failed to delete install package after install")
		}
		log.Info("Successfully installed osqueryd")
	}
	return nil
}
