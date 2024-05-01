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

// allows us to use `defer` from TestMain, since the TestMain ideom is to use
// os.Exit, which does not respect `defer`
func MainTestWrapper(m *testing.M, testFiles []string) int {
	for _, testCase := range testFiles {
		err := extractTestData(testCase)
		defer generateTestdataCleanup(testCase)()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error extracting %s: %s", testCase, err)
			return 1
		}
	}
	return m.Run()
}

func generateTestdataCleanup(path string) func() {
	return func() {
		_ = os.RemoveAll(extractedPathNameFromZipName(path))
	}
}

// extractedPathNameFromZipName turns the .zip name into the name of the extracted path.
// used for cleanup.
func extractedPathNameFromZipName(path string) string {
	baseName := strings.Split(filepath.Base(path), ".")[0]
	return filepath.Join("testdata", baseName)
}

// extractTestData from zip file and write it in the same dir as the zip file.
func extractTestData(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	dest := filepath.Dir(path)

	extractAndWriteFile := func(zipFile *zip.File) error {
		rc, err := zipFile.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, zipFile.Name) //nolint: gosec // test with controlled input
		if found, err := exists(path); err != nil || found {
			return err
		}

		if zipFile.FileInfo().IsDir() {
			err = os.MkdirAll(path, zipFile.Mode())
			if err != nil {
				return err
			}
		} else {
			destFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(0700))
			if err != nil {
				return err
			}
			defer destFile.Close()

			_, err = io.Copy(destFile, rc) //nolint: gosec // test with controlled input
			if err != nil {
				return err
			}

			err = os.Chmod(path, zipFile.Mode())
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
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
