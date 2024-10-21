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

package runner

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func createRepoZipArchive(ctx context.Context, dir string, dest string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path to %s: %w", dir, err)
	}

	projectFilesOutput, err := cmdBufferedOutput(exec.Command("git", "ls-files", "-z"), dir)
	if err != nil {
		return err
	}

	// Add files that are not yet tracked in git. Prevents a footcannon where someone writes code to a new file, then tests it before they add to git
	untrackedOutput, err := cmdBufferedOutput(exec.Command("git", "ls-files", "--exclude-standard", "-o", "-z"), dir)
	if err != nil {
		return err
	}

	_, err = io.Copy(&projectFilesOutput, &untrackedOutput)
	if err != nil {
		return fmt.Errorf("failed to read stdout of git ls-files -o: %w", err)
	}

	archive, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", dest, err)
	}
	defer archive.Close()

	zw := zip.NewWriter(archive)
	defer zw.Close()

	s := bufio.NewScanner(&projectFilesOutput)
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if i := strings.IndexRune(string(data), '\x00'); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if !atEOF {
			return 0, nil, nil
		}
		return len(data), data, bufio.ErrFinalToken
	})
	for s.Scan() {
		if ctx.Err() != nil {
			// incomplete close and delete
			_ = archive.Close()
			_ = os.Remove(dest)
			return ctx.Err()
		}
		err := func(line string) error {
			if line == "" {
				return nil
			}
			fullPath := filepath.Join(absDir, line)
			s, err := os.Stat(fullPath)
			if err != nil {
				return fmt.Errorf("failed to stat file %s: %w", fullPath, err)
			}
			if s.IsDir() {
				// skip directories
				return nil
			}
			f, err := os.Open(fullPath)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", fullPath, err)
			}
			defer f.Close()
			w, err := zw.Create(line)
			if err != nil {
				return fmt.Errorf("failed to create zip entry %s: %w", line, err)
			}
			_, err = io.Copy(w, f)
			if err != nil {
				return fmt.Errorf("failed to copy zip entry %s: %w", line, err)
			}
			return nil
		}(s.Text())
		if err != nil {
			return fmt.Errorf("error adding files: %w", err)
		}
	}
	return nil
}

func cmdBufferedOutput(cmd *exec.Cmd, workDir string) (bytes.Buffer, error) {
	var stdoutBuf bytes.Buffer
	cmd.Dir = workDir
	cmd.Stdout = &stdoutBuf
	err := cmd.Run()
	if err != nil {
		return *bytes.NewBufferString(""), fmt.Errorf("failed to run cmd %s: %w", strings.Join(cmd.Args, " "), err)
	}
	return stdoutBuf, nil
}
