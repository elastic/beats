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

// +build !windows

package filestream

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/match"
)

func TestFileScannerSymlinks(t *testing.T) {
	testCases := map[string]struct {
		paths         []string
		excludedFiles []match.Matcher
		symlinks      bool
		expectedFiles []string
	}{
		// covers test_input.py/test_skip_symlinks
		"skip symlinks": {
			paths: []string{
				filepath.Join("testdata", "symlink_to_included_file"),
				filepath.Join("testdata", "included_file"),
			},
			symlinks: false,
			expectedFiles: []string{
				mustAbsPath(filepath.Join("testdata", "included_file")),
			},
		},
		"return a file once if symlinks are enabled": {
			paths: []string{
				filepath.Join("testdata", "symlink_to_included_file"),
				filepath.Join("testdata", "included_file"),
			},
			symlinks: true,
			expectedFiles: []string{
				mustAbsPath(filepath.Join("testdata", "included_file")),
			},
		},
	}

	err := os.Symlink(
		mustAbsPath(filepath.Join("testdata", "included_file")),
		mustAbsPath(filepath.Join("testdata", "symlink_to_included_file")),
	)
	if err != nil {
		t.Fatal(err)
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			cfg := fileScannerConfig{
				ExcludedFiles: test.excludedFiles,
				Symlinks:      true,
				RecursiveGlob: false,
			}
			fs, err := newFileScanner(test.paths, cfg)
			if err != nil {
				t.Fatal(err)
			}
			files := fs.GetFiles()
			paths := make([]string, 0)
			for p, _ := range files {
				paths = append(paths, p)
			}
			assert.Equal(t, test.expectedFiles, paths)
		})
	}
}
