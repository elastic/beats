// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package zip

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
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
	if err := i.unzip(programName, version, installDir); err != nil {
		return err
	}

	oldPath := filepath.Join(installDir, fmt.Sprintf("%s-%s-windows", programName, version))
	newPath := filepath.Join(installDir, strings.Title(programName))
	if err := os.Rename(oldPath, newPath); err != nil {
		return errors.New(err, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, newPath))
	}

	return i.runInstall(programName, installDir)
}

func (i *Installer) unzip(programName, version, installPath string) error {
	artifactPath, err := artifact.GetArtifactPath(programName, version, i.config.OS(), i.config.Arch(), i.config.TargetDirectory)
	if err != nil {
		return err
	}

	if _, err := os.Stat(artifactPath); err != nil {
		return errors.New(fmt.Sprintf("artifact for '%s' version '%s' could not be found at '%s'", programName, version, artifactPath), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, artifactPath))
	}

	powershellArg := fmt.Sprintf("Expand-Archive -Path \"%s\" -DestinationPath \"%s\"", artifactPath, installPath)
	installCmd := exec.Command("powershell", "-command", powershellArg)
	return installCmd.Run()
}

func (i *Installer) runInstall(programName, installPath string) error {
	powershellCmd := fmt.Sprintf(powershellCmdTemplate, installPath, programName)

	installCmd := exec.Command("powershell", "-command", powershellCmd)
	return installCmd.Run()
}
