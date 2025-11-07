// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin || synthetics

package source

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func unzip(tf *os.File, targetDir string, folder string) error {
	rdr, err := zip.OpenReader(tf.Name())
	if err != nil {
		return err
	}
	defer rdr.Close()

	for _, f := range rdr.File {
		err = unzipFile(targetDir, folder, f)
		if err != nil {
			rmErr := os.RemoveAll(targetDir)
			if rmErr != nil {
				return fmt.Errorf("could not remove directory after encountering error unzipping file: %w, %s", rmErr, fmt.Sprintf(`(original unzip error: %s)`, err.Error()))
			}
			return err
		}
	}
	return nil
}

func sanitizeFilePath(filePath string, workdir string) (string, error) {
	destPath := filepath.Join(workdir, filePath)
	if !strings.HasPrefix(destPath, filepath.Clean(workdir)+string(os.PathSeparator)) {
		return filePath, fmt.Errorf("failed to extract illegal file path: %s", filePath)
	}
	return destPath, nil
}

// unzip file takes a given directory and a zipped file and extracts
// all the contents of the file based on the provided folder path,
// if the folder path is empty, it extracts the contents based on file
// tree structure
func unzipFile(workdir string, folder string, f *zip.File) error {
	var destPath string
	var err error
	if folder != "" {
		folderPaths := strings.Split(folder, string(filepath.Separator))
		var folderDepth = 1
		for _, path := range folderPaths {
			if path != "" {
				folderDepth++
			}
		}
		splitZipFileName := strings.Split(f.Name, string(filepath.Separator))
		root := splitZipFileName[0]

		prefix := filepath.Join(root, folder)
		if !strings.HasPrefix(f.Name, prefix) {
			return nil
		}

		sansFolder := splitZipFileName[folderDepth:]
		destPath = filepath.Join(workdir, filepath.Join(sansFolder...))
	} else {
		destPath, err = sanitizeFilePath(f.Name, workdir)
		if err != nil {
			return err
		}
	}

	// Never unpack node modules
	if strings.HasPrefix(destPath, "node_modules/") {
		return nil
	}

	if f.FileInfo().IsDir() {
		err := os.MkdirAll(destPath, 0755)
		if err != nil {
			return fmt.Errorf("could not make dest zip dir '%s': %w", destPath, err)
		}
		return nil
	}

	// In the case of project monitors, the destPath would be the direct
	// file path instead of directory, so we create the directory
	// if its not set up properly
	destDir := filepath.Dir(destPath)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		err = os.MkdirAll(destDir, defaultMod) // Create your file
		if err != nil {
			return fmt.Errorf("could not make dest zip dir '%s': %w", destDir, err)
		}
	}

	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("could not create dest file for zip '%s': %w", destPath, err)
	}
	err = os.Chmod(destPath, defaultMod)
	if err != nil {
		return fmt.Errorf("failed assigning default mode %s to %s: %w", defaultMod, destPath, err)
	}

	defer dest.Close()

	rdr, err := f.Open()
	if err != nil {
		return fmt.Errorf("could not open source zip file '%s': %w", f.Name, err)
	}
	defer rdr.Close()

	// Cap decompression to a max of 2GiB to prevent decompression bombs
	//nolint:gosec // zip bomb possibility, but user controls the zip, so it would only impact them
	_, err = io.Copy(dest, rdr)
	if err != nil {
		return err
	}

	return nil
}
