// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package zip

import (
	"archive/zip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
)

const (
	// powershellCmdTemplate uses elevated execution policy to avoid failure in case script execution is disabled on the system
	powershellCmdTemplate = `set-executionpolicy unrestricted; cd %s; .\install-service-%s.ps1`
)

// Installer or zip packages
type Installer struct {
	config *artifact.Config
}

// NewInstaller creates an installer able to install zip packages
func NewInstaller(config *artifact.Config) (*Installer, error) {
	return &Installer{
		config: config,
	}, nil
}

// Install performs installation of program in a specific version.
// It expects package to be already downloaded.
func (i *Installer) Install(programName, version, installDir string) error {
	artifactPath, err := artifact.GetArtifactPath(programName, version, i.config.OS(), i.config.Arch(), i.config.TargetDirectory)
	if err != nil {
		return err
	}

	if err := i.unzip(artifactPath, programName, version); err != nil {
		return err
	}

	rootDir, err := i.getRootDir(artifactPath)
	if err != nil {
		return err
	}

	// if root directory is not the same as desired directory rename
	// e.g contains `-windows-` or  `-SNAPSHOT-`
	if rootDir != installDir {
		if err := os.Rename(rootDir, installDir); err != nil {
			return errors.New(err, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, installDir))
		}
	}

	return i.runInstall(programName, version, installDir)
}

func (i *Installer) unzip(artifactPath, programName, version string) error {
	if _, err := os.Stat(artifactPath); err != nil {
		return errors.New(fmt.Sprintf("artifact for '%s' version '%s' could not be found at '%s'", programName, version, artifactPath), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, artifactPath))
	}

	powershellArg := fmt.Sprintf("Expand-Archive -LiteralPath \"%s\" -DestinationPath \"%s\"", artifactPath, i.config.InstallPath)
	installCmd := exec.Command("powershell", "-command", powershellArg)
	return installCmd.Run()
}

func (i *Installer) runInstall(programName, version, installPath string) error {
	powershellCmd := fmt.Sprintf(powershellCmdTemplate, installPath, programName)
	installCmd := exec.Command("powershell", "-command", powershellCmd)

	return installCmd.Run()
}

// retrieves root directory from zip archive
func (i *Installer) getRootDir(zipPath string) (dir string, err error) {
	defer func() {
		if dir != "" {
			dir = filepath.Join(i.config.InstallPath, dir)
		}
	}()

	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer zipReader.Close()

	var rootDir string
	for _, f := range zipReader.File {
		if filepath.Base(f.Name) == filepath.Dir(f.Name) {
			return f.Name, nil
		}

		if currentDir := filepath.Dir(f.Name); rootDir == "" || len(currentDir) < len(rootDir) {
			rootDir = currentDir
		}
	}

	return rootDir, nil
}
