// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package zip

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"

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
func (i *Installer) Install(_ context.Context, programName, version, installDir string) error {
	artifactPath, err := artifact.GetArtifactPath(programName, version, i.config.OS(), i.config.Arch(), i.config.TargetDirectory)
	if err != nil {
		return err
	}

	// cleanup install directory before unzip
	_, err = os.Stat(installDir)
	if err == nil || os.IsExist(err) {
		os.RemoveAll(installDir)
	}

	if err := i.unzip(artifactPath); err != nil {
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

	return nil
}

func (i *Installer) unzip(artifactPath string) error {
	r, err := zip.OpenReader(artifactPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(i.config.InstallPath, 0755); err != nil && !os.IsExist(err) {
		// failed to create install dir
		return err
	}

	unpackFile := func(f *zip.File) (err error) {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if cerr := rc.Close(); cerr != nil {
				err = multierror.Append(err, cerr)
			}
		}()

		path := filepath.Join(i.config.InstallPath, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if cerr := f.Close(); cerr != nil {
					err = multierror.Append(err, cerr)
				}
			}()

			if _, err = io.Copy(f, rc); err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		if err := unpackFile(f); err != nil {
			return err
		}
	}

	return nil
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
