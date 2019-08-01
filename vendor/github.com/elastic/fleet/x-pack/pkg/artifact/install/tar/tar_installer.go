// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/fleet/x-pack/pkg/artifact"
	"github.com/pkg/errors"
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
func (i *Installer) Install(programName, version, _ string) error {
	artifactPath, err := artifact.GetArtifactPath(programName, version, i.config.OS(), i.config.Arch(), i.config.TargetDirectory)
	if err != nil {
		return err
	}

	f, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("artifact for '%s' version '%s' could not be found at '%s'", programName, version, artifactPath)
	}
	defer f.Close()

	return unpack(f, i.config.InstallPath)

}

func unpack(r io.Reader, dir string) error {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("requires gzip-compressed body: %v", err)
	}

	tr := tar.NewReader(zr)

	for {
		f, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !validFileName(f.Name) {
			return fmt.Errorf("tar contained invalid filename: %q", f.Name)
		}
		rel := filepath.FromSlash(f.Name)
		abs := filepath.Join(dir, rel)

		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsRegular():
			// just to be sure, it should already be created by Dir type
			if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
				return errors.Wrap(err, "TarInstaller: creating directory for file "+abs)
			}

			wf, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return errors.Wrap(err, "TarInstaller: creating file "+abs)
			}

			_, err = io.Copy(wf, tr)
			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				return fmt.Errorf("TarInstaller: error writing to %s: %v", abs, err)
			}
		case mode.IsDir():
			if err := os.MkdirAll(abs, 0755); err != nil {
				return errors.Wrap(err, "TarInstaller: creating directory for file "+abs)
			}
		default:
			return fmt.Errorf("tar file entry %s contained unsupported file type %v", f.Name, mode)
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
