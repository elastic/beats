// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tar

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func pathInDir(path, dir string) bool {
	cleanPath := filepath.Clean(path)
	cleanDir := filepath.Clean(dir)
	if cleanPath == cleanDir {
		return true
	}
	return strings.HasPrefix(cleanPath, cleanDir+string(os.PathSeparator))
}

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

func ExtractFile(fp string, destinationDir string, files ...string) error {
	return extractFile(fp, destinationDir, false, files...)
}

// ExtractFileSkipEscaping is like ExtractFile but silently skips symlink and
// hardlink entries whose targets escape the destination directory.
func ExtractFileSkipEscaping(fp string, destinationDir string, files ...string) error {
	return extractFile(fp, destinationDir, true, files...)
}

func extractFile(fp string, destinationDir string, skipEscaping bool, files ...string) error {
	f, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	return extract(zr, destinationDir, skipEscaping, files...)
}

// Extract extracts entries from a tar reader, rejecting any entries whose
// paths or link targets escape the destination directory.
func Extract(r io.Reader, destinationDir string, files ...string) error {
	return extract(r, destinationDir, false, files...)
}

// ExtractSkipEscaping extracts entries from a tar reader, silently skipping
// symlink and hardlink entries whose targets escape the destination directory.
// Regular file path traversal attempts still cause a hard error.
func ExtractSkipEscaping(r io.Reader, destinationDir string, files ...string) error {
	return extract(r, destinationDir, true, files...)
}

func extract(r io.Reader, destinationDir string, skipEscaping bool, files ...string) error {
	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		shouldExtract := shouldExtract(header.Name, files...)
		if !shouldExtract {
			continue
		}

		//nolint:gosec // file path is checked below
		path := filepath.Join(destinationDir, header.Name)
		if !pathInDir(path, destinationDir) {
			return fmt.Errorf("illegal file path in tar: %v", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(path, os.FileMode(header.Mode)&os.ModePerm); err != nil { //nolint:gosec // mode clamped to permission bits
				return err
			}
		case tar.TypeReg:
			if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
				return err
			}
			writer, err := os.Create(path)
			if err != nil {
				return err
			}

			//nolint:gosec // used during build only, check sums are validated beforehand, the size of distro is predicatable
			if _, err = io.Copy(writer, tarReader); err != nil {
				_ = writer.Close()
				return err
			}

			if err = os.Chmod(path, os.FileMode(header.Mode)&os.ModePerm); err != nil { //nolint:gosec // mode clamped to permission bits
				_ = writer.Close()
				return err
			}

			if err = writer.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
				return err
			}
			resolvedTarget := header.Linkname
			if !filepath.IsAbs(header.Linkname) {
				resolvedTarget = filepath.Join(filepath.Dir(path), header.Linkname) //nolint:gosec // validated by pathInDir below
			}
			if !pathInDir(resolvedTarget, destinationDir) {
				if skipEscaping {
					continue
				}
				return fmt.Errorf("illegal symlink target in tar: %v -> %v", header.Name, header.Linkname)
			}
			if err = os.Symlink(header.Linkname, path); err != nil {
				return err
			}
		case tar.TypeLink:
			if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
				return err
			}
			targetPath := filepath.Join(destinationDir, header.Linkname)
			if !pathInDir(targetPath, destinationDir) {
				if skipEscaping {
					continue
				}
				return fmt.Errorf("illegal hardlink target in tar: %v -> %v", header.Name, header.Linkname)
			}
			if err = os.Link(targetPath, path); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unable to untar type=%c in file=%s", header.Typeflag, path)
		}
	}
	return nil
}
