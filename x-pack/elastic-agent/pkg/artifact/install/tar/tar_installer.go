// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tar

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
)

// Installer or tar packages
type Installer struct {
	config *artifact.Config
}

// NewInstaller creates an installer able to install tar packages
func NewInstaller(config *artifact.Config) (*Installer, error) {
	return &Installer{
		config: config,
	}, nil
}

// Install performs installation of program in a specific version.
// It expects package to be already downloaded.
func (i *Installer) Install(_ context.Context, spec program.Spec, version, installDir string) error {
	artifactPath, err := artifact.GetArtifactPath(spec, version, i.config.OS(), i.config.Arch(), i.config.TargetDirectory)
	if err != nil {
		return err
	}

	f, err := os.Open(artifactPath)
	if err != nil {
		return errors.New(fmt.Sprintf("artifact for '%s' version '%s' could not be found at '%s'", spec.Name, version, artifactPath), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, artifactPath))
	}
	defer f.Close()

	// cleanup install directory before unpack
	_, err = os.Stat(installDir)
	if err == nil || os.IsExist(err) {
		os.RemoveAll(installDir)
	}

	// unpack must occur in directory that holds the installation directory
	// or the extraction will be double nested
	return unpack(f, filepath.Dir(installDir))
}

func unpack(r io.Reader, dir string) error {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return errors.New("requires gzip-compressed body", err, errors.TypeFilesystem)
	}

	tr := tar.NewReader(zr)
	var rootDir string

	for {
		f, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !validFileName(f.Name) {
			return errors.New("tar contained invalid filename: %q", f.Name, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, f.Name))
		}
		rel := filepath.FromSlash(f.Name)
		abs := filepath.Join(dir, rel)

		// find the root dir
		if currentDir := filepath.Dir(abs); rootDir == "" || len(filepath.Dir(rootDir)) > len(currentDir) {
			rootDir = currentDir
		}

		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsRegular():
			// just to be sure, it should already be created by Dir type
			if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
				return errors.New(err, "TarInstaller: creating directory for file "+abs, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, abs))
			}

			wf, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return errors.New(err, "TarInstaller: creating file "+abs, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, abs))
			}

			_, err = io.Copy(wf, tr)

			if err == nil {
				// sometimes we try executing binary too fast and run into text file busy after unpacking
				// syncing prevents this
				if syncErr := wf.Sync(); syncErr != nil {
					err = syncErr
				}
			}

			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}

			if err != nil {
				return fmt.Errorf("TarInstaller: error writing to %s: %v", abs, err)
			}
		case mode.IsDir():
			if err := os.MkdirAll(abs, 0755); err != nil {
				return errors.New(err, "TarInstaller: creating directory for file "+abs, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, abs))
			}
		default:
			return errors.New(fmt.Sprintf("tar file entry %s contained unsupported file type %v", f.Name, mode), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, f.Name))
		}
	}

	return nil
}

func validFileName(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
