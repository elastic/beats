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

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

// TestLargeScaleFilestreamDelete tests Filestream's delete feature
// at different scales, this should not be running on CI, hence the
// actual test function is commented out.
// func TestLargeScaleFilestreamDelete(t *testing.T) {
// 	for _, nFiles := range []int{10, 500, 1000, 5000, 10000} {
// 		// Because Filstream, by default, only ingests files >= 1kb, the
// 		// number of lines cannot be too small. 30 has been pretty safe
// 		for _, lines := range []int{30, 100} {
// 			t.Run(fmt.Sprintf("%d files %d lines each", nFiles, lines), func(t *testing.T) {
// 				testLargeScaleFilestreamDelete(t, 2*time.Minute, nFiles, lines)
// 			})
// 		}
// 	}
// }

func testLargeScaleFilestreamDelete(t *testing.T, timeout time.Duration, nFiles, lines int) {
	fb := integration.NewRealBeat(t, "filebeat", "../../filebeat")
	start := time.Now()

	dir := generateLogFilesScale(t, fb.TempDir(), nFiles, lines)
	elapsed := time.Since(start)

	t.Logf("%d files with %d lines each generated in %s", nFiles, lines, elapsed)

	homePath := fb.TempDir()
	vars := map[string]any{
		"dir":      dir,
		"homePath": homePath,
	}

	cfg := getConfig(t, vars, "delete", "scale-test.yml")
	err := os.WriteFile(filepath.Join(homePath, "filebeat.yml"), []byte(cfg), 0666)
	if err != nil {
		t.Fatalf("cannot write config file: %s", err)
	}

	deletedCount := atomic.Uint64{}
	fileWatcher := integration.NewFileWatcher(t, dir)
	fileWatcher.SetEventCallback(func(event fsnotify.Event) {
		if event.Has(fsnotify.Remove) {
			deletedCount.Add(1)
		}
	})
	fileWatcher.Start()
	fbStarted := time.Now()
	fb.Start()

	buff := strings.Builder{}
	require.Eventuallyf(t, func() bool {
		buff.Reset()

		count := deletedCount.Load()
		fmt.Fprintf(&buff, "%d", count)

		return count == uint64(nFiles)
	}, timeout, time.Millisecond*100, "expecting %d deleted files, got: %s", nFiles, &buff)

	t.Logf("Filebeat took %s to remove %d files", time.Since(fbStarted), nFiles)
}

func generateLogFilesScale(t *testing.T, tempDir string, files, lines int) string {
	fmt.Println("Temp dir:", tempDir)
	basePath := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(basePath, 0777); err != nil {
		t.Fatalf("cannot create folder to store logs: %s", err)
	}

	for fCount := range files {
		path := filepath.Join(basePath, fmt.Sprintf("%06d.log", fCount))
		f, err := os.Create(path)
		if err != nil {
			t.Errorf("cannot create file: %s", err)
		}
		for range lines {
			f.WriteString(gofakeit.HackerPhrase() + "\n")
		}
		if err := f.Sync(); err != nil {
			t.Errorf("cannot sync %q: %s", path, err)
		}
		if err := f.Close(); err != nil {
			t.Errorf("cannot close %q: %s", path, err)
		}
	}

	return basePath
}
