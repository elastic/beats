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

package filestream

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var (
	excludedFileName = "excluded_file"
	includedFileName = "included_file"
	directoryPath    = "unharvestable_dir"
)

func TestFileScanner(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "fswatch_test_file_scanner")
	if err != nil {
		t.Fatalf("cannot create temporary test dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	setupFilesForScannerTest(t, tmpDir)

	excludedFilePath := filepath.Join(tmpDir, excludedFileName)
	includedFilePath := filepath.Join(tmpDir, includedFileName)

	testCases := map[string]struct {
		paths         []string
		excludedFiles []match.Matcher
		includedFiles []match.Matcher
		symlinks      bool
		expectedFiles []string
	}{
		"select all files": {
			paths:         []string{excludedFilePath, includedFilePath},
			expectedFiles: []string{excludedFilePath, includedFilePath},
		},
		"skip excluded files": {
			paths: []string{excludedFilePath, includedFilePath},
			excludedFiles: []match.Matcher{
				match.MustCompile(excludedFileName),
			},
			expectedFiles: []string{includedFilePath},
		},
		"only include included_files": {
			paths: []string{excludedFilePath, includedFilePath},
			includedFiles: []match.Matcher{
				match.MustCompile(includedFileName),
			},
			expectedFiles: []string{includedFilePath},
		},
		"skip directories": {
			paths:         []string{filepath.Join(tmpDir, directoryPath)},
			expectedFiles: []string{},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			cfg := fileScannerConfig{
				ExcludedFiles: test.excludedFiles,
				IncludedFiles: test.includedFiles,
				Symlinks:      test.symlinks,
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
			assert.ElementsMatch(t, paths, test.expectedFiles)
		})
	}
}

func setupFilesForScannerTest(t *testing.T, tmpDir string) {
	err := os.Mkdir(filepath.Join(tmpDir, directoryPath), 0750)
	if err != nil {
		t.Fatalf("cannot create non harvestable directory: %v", err)
	}
	for _, path := range []string{excludedFileName, includedFileName} {
		f, err := os.Create(filepath.Join(tmpDir, path))
		if err != nil {
			t.Fatalf("file %s, error %v", path, err)
		}
		f.Close()
	}
}

func TestFileWatchNewDeleteModified(t *testing.T) {
	oldTs := time.Now()
	newTs := oldTs.Add(5 * time.Second)
	testCases := map[string]struct {
		prevFiles      map[string]os.FileInfo
		nextFiles      map[string]os.FileInfo
		expectedEvents []loginp.FSEvent
	}{
		"one new file": {
			prevFiles: map[string]os.FileInfo{},
			nextFiles: map[string]os.FileInfo{
				"new_path": testFileInfo{"new_path", 5, oldTs, nil},
			},
			expectedEvents: []loginp.FSEvent{
				{Op: loginp.OpCreate, OldPath: "", NewPath: "new_path", Info: testFileInfo{"new_path", 5, oldTs, nil}},
			},
		},
		"one deleted file": {
			prevFiles: map[string]os.FileInfo{
				"old_path": testFileInfo{"old_path", 5, oldTs, nil},
			},
			nextFiles: map[string]os.FileInfo{},
			expectedEvents: []loginp.FSEvent{
				{Op: loginp.OpDelete, OldPath: "old_path", NewPath: "", Info: testFileInfo{"old_path", 5, oldTs, nil}},
			},
		},
		"one modified file": {
			prevFiles: map[string]os.FileInfo{
				"path": testFileInfo{"path", 5, oldTs, nil},
			},
			nextFiles: map[string]os.FileInfo{
				"path": testFileInfo{"path", 10, newTs, nil},
			},
			expectedEvents: []loginp.FSEvent{
				{Op: loginp.OpWrite, OldPath: "path", NewPath: "path", Info: testFileInfo{"path", 10, newTs, nil}},
			},
		},
		"two modified files": {
			prevFiles: map[string]os.FileInfo{
				"path1": testFileInfo{"path1", 5, oldTs, nil},
				"path2": testFileInfo{"path2", 5, oldTs, nil},
			},
			nextFiles: map[string]os.FileInfo{
				"path1": testFileInfo{"path1", 10, newTs, nil},
				"path2": testFileInfo{"path2", 10, newTs, nil},
			},
			expectedEvents: []loginp.FSEvent{
				{Op: loginp.OpWrite, OldPath: "path1", NewPath: "path1", Info: testFileInfo{"path1", 10, newTs, nil}},
				{Op: loginp.OpWrite, OldPath: "path2", NewPath: "path2", Info: testFileInfo{"path2", 10, newTs, nil}},
			},
		},
		"one modified file, one new file": {
			prevFiles: map[string]os.FileInfo{
				"path1": testFileInfo{"path1", 5, oldTs, nil},
			},
			nextFiles: map[string]os.FileInfo{
				"path1": testFileInfo{"path1", 10, newTs, nil},
				"path2": testFileInfo{"path2", 10, newTs, nil},
			},
			expectedEvents: []loginp.FSEvent{
				{Op: loginp.OpWrite, OldPath: "path1", NewPath: "path1", Info: testFileInfo{"path1", 10, newTs, nil}},
				{Op: loginp.OpCreate, OldPath: "", NewPath: "path2", Info: testFileInfo{"path2", 10, newTs, nil}},
			},
		},
		"one new file, one deleted file": {
			prevFiles: map[string]os.FileInfo{
				"path_deleted": testFileInfo{"path_deleted", 5, oldTs, nil},
			},
			nextFiles: map[string]os.FileInfo{
				"path_new": testFileInfo{"path_new", 10, newTs, nil},
			},
			expectedEvents: []loginp.FSEvent{
				{Op: loginp.OpDelete, OldPath: "path_deleted", NewPath: "", Info: testFileInfo{"path_deleted", 5, oldTs, nil}},
				{Op: loginp.OpCreate, OldPath: "", NewPath: "path_new", Info: testFileInfo{"path_new", 10, newTs, nil}},
			},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			w := fileWatcher{
				log:          logp.L(),
				prev:         test.prevFiles,
				scanner:      &mockScanner{test.nextFiles},
				events:       make(chan loginp.FSEvent),
				sameFileFunc: testSameFile,
			}

			go w.watch(context.Background())

			count := len(test.expectedEvents)
			actual := make([]loginp.FSEvent, count)
			for i := 0; i < count; i++ {
				actual[i] = w.Event()
			}

			assert.ElementsMatch(t, actual, test.expectedEvents)
		})
	}
}

type mockScanner struct {
	files map[string]os.FileInfo
}

func (m *mockScanner) GetFiles() map[string]os.FileInfo {
	return m.files
}

type testFileInfo struct {
	path string
	size int64
	time time.Time
	sys  interface{}
}

func (t testFileInfo) Name() string       { return t.path }
func (t testFileInfo) Size() int64        { return t.size }
func (t testFileInfo) Mode() os.FileMode  { return 0 }
func (t testFileInfo) ModTime() time.Time { return t.time }
func (t testFileInfo) IsDir() bool        { return false }
func (t testFileInfo) Sys() interface{}   { return t.sys }

func testSameFile(fi1, fi2 os.FileInfo) bool {
	return fi1.Name() == fi2.Name()
}
