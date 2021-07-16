// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

// Install installs Elastic Agent persistently on the system including creating and starting its service.
func Install(cfgFile string) error {
	dir, err := findDirectory()
	if err != nil {
		return errors.New(err, "failed to discover the source directory for installation", errors.TypeFilesystem)
	}

	// uninstall current installation
	err = Uninstall(cfgFile)
	if err != nil {
		return err
	}

	// ensure parent directory exists, copy source into install path
	err = os.MkdirAll(filepath.Dir(paths.InstallPath), 0755)
	if err != nil {
		return errors.New(
			err,
			fmt.Sprintf("failed to create installation parent directory (%s)", filepath.Dir(paths.InstallPath)),
			errors.M("directory", filepath.Dir(paths.InstallPath)))
	}
	err = copy.Copy(dir, paths.InstallPath, copy.Options{
		OnSymlink: func(_ string) copy.SymlinkAction {
			return copy.Shallow
		},
		Sync: true,
	})
	if err != nil {
		return errors.New(
			err,
			fmt.Sprintf("failed to copy source directory (%s) to destination (%s)", dir, paths.InstallPath),
			errors.M("source", dir), errors.M("destination", paths.InstallPath))
	}

	// place shell wrapper, if present on platform
	if paths.ShellWrapperPath != "" {
		err = os.MkdirAll(filepath.Dir(paths.ShellWrapperPath), 0755)
		if err == nil {
			err = ioutil.WriteFile(paths.ShellWrapperPath, []byte(paths.ShellWrapper), 0755)
		}
		if err != nil {
			return errors.New(
				err,
				fmt.Sprintf("failed to write shell wrapper (%s)", paths.ShellWrapperPath),
				errors.M("destination", paths.ShellWrapperPath))
		}
	}

	// post install (per platform)
	err = postInstall()
	if err != nil {
		return err
	}

	// fix permissions
	err = FixPermissions()
	if err != nil {
		return errors.New(
			err,
			"failed to perform permission changes",
			errors.M("destination", paths.InstallPath))
	}

	// install service
	svc, err := newService()
	if err != nil {
		return err
	}
	err = svc.Install()
	if err != nil {
		return errors.New(
			err,
			fmt.Sprintf("failed to install service (%s)", paths.ServiceName),
			errors.M("service", paths.ServiceName))
	}
	return nil
}

// StartService starts the installed service.
//
// This should only be called after Install is successful.
func StartService() error {
	svc, err := newService()
	if err != nil {
		return err
	}
	err = svc.Start()
	if err != nil {
		return errors.New(
			err,
			fmt.Sprintf("failed to start service (%s)", paths.ServiceName),
			errors.M("service", paths.ServiceName))
	}
	return nil
}

// StopService stops the installed service.
func StopService() error {
	svc, err := newService()
	if err != nil {
		return err
	}
	err = svc.Stop()
	if err != nil {
		return errors.New(
			err,
			fmt.Sprintf("failed to stop service (%s)", paths.ServiceName),
			errors.M("service", paths.ServiceName))
	}
	return nil
}

// FixPermissions fixes the permissions on the installed system.
func FixPermissions() error {
	return fixPermissions()
}

// findDirectory returns the directory to copy into the installation location.
//
// This also verifies that the discovered directory is a valid directory for installation.
func findDirectory() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return "", err
	}
	sourceDir := filepath.Dir(execPath)
	if info.IsInsideData(sourceDir) {
		// executable path is being reported as being down inside of data path
		// move up to directories to perform the copy
		sourceDir = filepath.Dir(filepath.Dir(sourceDir))
	}
	err = verifyDirectory(sourceDir)
	if err != nil {
		return "", err
	}
	return sourceDir, nil
}

// verifyDirectory ensures that the directory includes the executable.
func verifyDirectory(dir string) error {
	_, err := os.Stat(filepath.Join(dir, paths.BinaryName))
	if os.IsNotExist(err) {
		return fmt.Errorf("missing %s", paths.BinaryName)
	}
	return nil
}
