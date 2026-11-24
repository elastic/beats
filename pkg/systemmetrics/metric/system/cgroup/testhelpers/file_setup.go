// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package testhelpers

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MainTestWrapper unpacks the given zipped test fixtures before running the tests.
// The extracted directories are left in place; they are gitignored, and keeping them
// lets sibling packages share them instead of racing to delete each other's copies.
func MainTestWrapper(m *testing.M, testFiles []string) int {
	for _, testCase := range testFiles {
		if err := extractTestData(testCase); err != nil {
			fmt.Fprintf(os.Stderr, "error extracting %s: %s\n", testCase, err)
			return 1
		}
	}
	return m.Run()
}

// extractTestData unpacks a zip file next to itself, so testdata/docker.zip becomes
// testdata/docker. Each archive holds a single top-level directory named after the
// archive.
//
// Several packages under metric/system/cgroup share one testdata directory and `go test`
// runs them as concurrent processes, so the tree is built in a temporary directory and
// published with a single rename. Readers therefore only ever see a complete tree, and
// a process that loses the race simply keeps the copy that was published first.
func extractTestData(path string) error {
	dest := filepath.Dir(path)
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	target := filepath.Join(dest, name)

	if found, err := exists(target); err != nil || found {
		return err
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	tmpDir, err := os.MkdirTemp(dest, ".extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	extractAndWriteFile := func(zipFile *zip.File) error {
		path := filepath.Join(tmpDir, zipFile.Name) //nolint:gosec // test with controlled input

		if zipFile.FileInfo().IsDir() {
			return os.MkdirAll(path, 0o755)
		}

		rc, err := zipFile.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// Archives are not guaranteed to list a directory before its contents.
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}

		destFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, rc); err != nil { //nolint:gosec // test with controlled input
			return err
		}

		return os.Chmod(path, zipFile.Mode())
	}

	for _, f := range r.File {
		if err := extractAndWriteFile(f); err != nil {
			return err
		}
	}

	if err := os.Rename(filepath.Join(tmpDir, name), target); err != nil {
		if found, existsErr := exists(target); existsErr != nil || !found {
			return err
		}
	}

	return nil
}

// exists returns whether the given file or directory exists or not
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
