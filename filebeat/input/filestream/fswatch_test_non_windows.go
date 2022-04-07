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

//go:build !windows
// +build !windows

package filestream

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	loginp "github.com/elastic/beats/v8/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v8/libbeat/common/match"
	"github.com/elastic/beats/v8/libbeat/logp"
)

func TestFileScannerSymlinks(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "fswatch_test_file_scanner")
	if err != nil {
		t.Fatalf("cannot create temporary test dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	setupFilesForScannerTest(t, tmpDir)

	testCases := map[string]struct {
		paths         []string
		excludedFiles []match.Matcher
		includedFiles []match.Matcher
		symlinks      bool
		expectedFiles []string
	}{
		// covers test_input.py/test_skip_symlinks
		"skip symlinks": {
			paths: []string{
				filepath.Join(tmpDir, "symlink_to_0"),
				filepath.Join(tmpDir, "included_file"),
			},
			symlinks: false,
			expectedFiles: []string{
				mustAbsPath(filepath.Join(tmpDir, "included_file")),
			},
		},
		"return a file once if symlinks are enabled": {
			paths: []string{
				filepath.Join(tmpDir, "symlink_to_0"),
				filepath.Join(tmpDir, "included_file"),
			},
			symlinks: true,
			expectedFiles: []string{
				mustAbsPath(filepath.Join(tmpDir, "included_file")),
			},
		},
		"do not return symlink if original file is not allowed": {
			paths: []string{
				filepath.Join(tmpDir, "symlink_to_1"),
				filepath.Join(tmpDir, "included_file"),
			},
			excludedFiles: []match.Matcher{
				match.MustCompile("original_" + excludedFileName),
			},
			symlinks: true,
			expectedFiles: []string{
				mustAbsPath(filepath.Join(tmpDir, "included_file")),
			},
		},
	}

	for i, filename := range []string{"included_file", "excluded_file"} {
		err := os.Symlink(
			mustAbsPath(filepath.Join(tmpDir, "original_"+filename)),
			mustAbsPath(filepath.Join(tmpDir, "symlink_to_"+strconv.Itoa(i))),
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			cfg := fileScannerConfig{
				ExcludedFiles: test.excludedFiles,
				IncludedFiles: test.includedFiles,
				Symlinks:      true,
				RecursiveGlob: false,
			}
			fs, err := newFileScanner(test.paths, cfg)
			if err != nil {
				t.Fatal(err)
			}
			files := fs.GetFiles()
			paths := make([]string, 0)
			for p := range files {
				paths = append(paths, p)
			}
			assert.ElementsMatch(t, test.expectedFiles, paths)
		})
	}
}

func TestFileWatcherRenamedFile(t *testing.T) {
	testPath := mustAbsPath("first_name")
	renamedPath := mustAbsPath("renamed")

	f, err := os.Create(testPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	fi, err := os.Stat(testPath)
	if err != nil {
		t.Fatal(err)
	}

	cfg := fileScannerConfig{
		ExcludedFiles: nil,
		Symlinks:      false,
		RecursiveGlob: false,
	}
	scanner, err := newFileScanner([]string{testPath, renamedPath}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	w := fileWatcher{
		log:     logp.L(),
		scanner: scanner,
		events:  make(chan loginp.FSEvent),
	}

	go w.watch(context.Background())
	assert.Equal(t, loginp.FSEvent{Op: loginp.OpCreate, OldPath: "", NewPath: testPath, Info: fi}, w.Event())

	err = os.Rename(testPath, renamedPath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(renamedPath)
	fi, err = os.Stat(renamedPath)
	if err != nil {
		t.Fatal(err)
	}

	go w.watch(context.Background())
	evt := w.Event()

	assert.Equal(t, loginp.OpRename, evt.Op)
	assert.Equal(t, testPath, evt.OldPath)
	assert.Equal(t, renamedPath, evt.NewPath)
}

func mustAbsPath(filename string) string {
	abspath, err := filepath.Abs(filename)
	if err != nil {
		panic(err)
	}
	return abspath
}
