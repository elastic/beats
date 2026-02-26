// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func shouldExtract(name string, files ...string) bool {
	if files == nil {
		return true
	}

	// Clean the file path/name from the tar.gz archive
	// In the osquery 4.9.0 version the paths started to be prefixed with "./"
	// which caused the osqueryd binary not found/extracted from the archive.
	name = filepath.Clean(name)
	for _, f := range files {
		if strings.HasPrefix(name, f) ||
			strings.HasPrefix(f, name) {
			return true
		}
	}
	return false
}

// UnzipFile is a wrapper for Unzip
func UnzipFile(fp string, destinationDir string, files ...string) error {
	r, err := zip.OpenReader(fp)
	if err != nil {
		return err
	}
	defer r.Close()

	return Unzip(r, destinationDir, files...)
}

// Unzip extracts certain files from a zip to a destination directory
func Unzip(r *zip.ReadCloser, destinationDir string, files ...string) error {

	for _, file := range r.File {

		shouldExtract := shouldExtract(file.Name, files...)
		if !shouldExtract {
			continue
		}

		//nolint:gosec // file path is checked below
		path := filepath.Join(destinationDir, file.Name)
		// Check for directory traversal vulnerabilities.
		if !strings.HasPrefix(path, filepath.Clean(destinationDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", path)
		}

		if file.FileInfo().IsDir() {
			// It's a directory
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return err
			}
		} else {
			// It's a file
			outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return err
			}
			defer outFile.Close()

			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			//nolint:gosec // used during build only, check sums are validated beforehand, the size of distro is predicatable
			if _, err := io.Copy(outFile, rc); err != nil {
				return err
			}
		}
	}
	return nil
}
