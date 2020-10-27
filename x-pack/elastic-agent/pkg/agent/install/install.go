// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// Install installs Elastic Agent persistently on the system including creating and starting its service.
func Install() error {
	dir, err := findDirectory()
	if err != nil {
		return errors.New(err, "failed to discover the source directory for installation", errors.TypeFilesystem)
	}

	// uninstall current installation
	err = Uninstall()
	if err != nil {
		return err
	}

	// ensure parent directory exists, copy source into install path
	err = os.MkdirAll(filepath.Dir(InstallPath), 0755)
	if err != nil {
		return errors.New(
			err,
			fmt.Sprintf("failed to create installation parent directory (%s)", filepath.Dir(InstallPath)),
			errors.M("directory", filepath.Dir(InstallPath)))
	}
	err = copy.Copy(dir, InstallPath, copy.Options{
		OnSymlink: func(_ string) copy.SymlinkAction {
			return copy.Shallow
		},
		Sync: true,
	})
	if err != nil {
		return errors.New(
			err,
			fmt.Sprintf("failed to copy source directory (%s) to destination (%s)", dir, InstallPath),
			errors.M("source", dir), errors.M("destination", InstallPath))
	}

	// place shell wrapper, if present on platform
	if ShellWrapperPath != "" {
		err = os.MkdirAll(filepath.Dir(ShellWrapperPath), 0755)
		if err == nil {
			err = ioutil.WriteFile(ShellWrapperPath, []byte(ShellWrapper), 0755)
		}
		if err != nil {
			return errors.New(
				err,
				fmt.Sprintf("failed to write shell wrapper (%s)", ShellWrapperPath),
				errors.M("destination", ShellWrapperPath))
		}
	}

	// post install (per platform)
	err = postInstall()
	if err != nil {
		return err
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
			fmt.Sprintf("failed to install service (%s)", ServiceName),
			errors.M("service", ServiceName))
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
			fmt.Sprintf("failed to start service (%s)", ServiceName),
			errors.M("service", ServiceName))
	}
	return nil
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
	if insideData(sourceDir) {
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
	_, err := os.Stat(filepath.Join(dir, BinaryName))
	if os.IsNotExist(err) {
		return fmt.Errorf("missing %s", BinaryName)
	}
	return nil
}

// insideData returns true when the exePath is inside of the current Agents data path.
func insideData(exePath string) bool {
	expectedPath := filepath.Join("data", fmt.Sprintf("elastic-agent-%s", release.ShortCommit()))
	return strings.HasSuffix(exePath, expectedPath)
}
