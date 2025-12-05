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

package fs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTempDir(t *testing.T) {
	t.Run("temp dir is created using os.TempDir", func(t *testing.T) {
		tempDir := TempDir(t)
		osTempDir := os.TempDir()

		baseDir := filepath.Dir(tempDir)
		if baseDir != osTempDir {
			t.Fatalf("expecting %q to be created on %q", tempDir, osTempDir)
		}
	})

	t.Run("temp dir is created in the specified path", func(t *testing.T) {
		path := []string{os.TempDir(), "foo", "bar"}
		rootDir := filepath.Join(path...)

		// Pass each element here so we ensure they're correctly joined
		tempDir := TempDir(t, path...)

		baseDir := filepath.Dir(tempDir)
		if baseDir != rootDir {
			t.Fatalf("expecting %q to be created on %q", tempDir, rootDir)
		}
	})
}

func TestTempDirIsKeptOnTestFailure(t *testing.T) {
	rootDirEnv := "TEMPDIR_ROOTDIR"
	tempFilename := "it-works"
	if os.Getenv("INNER_TEST") == "1" {
		rootDir := os.Getenv(rootDirEnv)
		// We're inside the subprocess:
		// 1. Read the root dir set as env var by the 'main' process
		// 2. Use it when calling TempDir
		// 3. Create a file just to ensure this actually run
		tmpDir := TempDir(t, rootDir)
		if err := os.WriteFile(filepath.Join(tmpDir, tempFilename), []byte("it works\n"), 0x666); err != nil {
			t.Fatalf("cannot write temp file: %s", err)
		}

		t.Fatal("keep the folder")
		return
	}

	rootDir := os.TempDir()
	//nolint:gosec // This is intentionally running a subprocess
	cmd := exec.CommandContext(
		t.Context(),
		os.Args[0],
		fmt.Sprintf("-test.run=^%s$",
			t.Name()),
		"-test.v")
	cmd.Env = append(
		cmd.Env,
		"INNER_TEST=1",
		rootDirEnv+"="+rootDir,
	)

	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		// The test ran by cmd will fail and retrun 1 as the exit code. So we only
		// print the error if the main test fails.
		defer func() {
			if t.Failed() {
				t.Errorf(
					"the test process returned an error (this is expected on a normal test execution): %s",
					cmdErr)
				t.Logf("Output of the subprocess:\n%s\n", string(out))
			}
		}()
	}

	var tempFolder string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		txt := sc.Text()
		// To extract the temp folder path we split txt in a way that the path
		// is the 2nd element.
		// The string we're using as reference:
		// fs.go:97: Temporary directory saved: /tmp/TestTempDirIsKeptOnTestFailure2385221663
		if strings.Contains(txt, "Temporary directory saved:") {
			split := strings.Split(txt, "Temporary directory saved: ")
			if len(split) != 2 {
				t.Fatalf("could not parse log file form test output, invalid format %q", txt)
			}
			tempFolder = split[1]
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("Temp folder: %q", tempFolder)
				}
			})
		}
	}

	stat, err := os.Stat(tempFolder)
	if err != nil {
		t.Fatalf("cannot stat created temp folder: %s", err)
	}

	if !stat.IsDir() {
		t.Errorf("%s must be a directory", tempFolder)
	}

	if _, err = os.Stat(filepath.Join(tempFolder, tempFilename)); err != nil {
		t.Fatalf("cannot stat file create by subprocess: %s", err)
	}

	// Be nice and cleanup
	if err := os.RemoveAll(tempFolder); err != nil {
		t.Fatalf("cannot remove created folders: %s", err)
	}
}
