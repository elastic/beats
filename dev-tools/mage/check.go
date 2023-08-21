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

package mage

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/elastic/elastic-agent-libs/processors/dissect"
)

func CheckNoChanges() error {
	changes, err := GitDiffIndex()
	if err != nil {
		return fmt.Errorf("failed to diff the git index: %w", err)
	}

	if len(changes) > 0 {
		if mg.Verbose() {
			err = GitDiff()
			if err != nil {
				return fmt.Errorf("failed to run git diff: %w", err)
			}
		}

		return fmt.Errorf("some files are not up-to-date. "+
			"Usually running 'mage update' or 'mage addLicenseHeaders' "+
			"fixes the issues. Fix the issues, review and commit the changes. "+
			"Modified: %v", changes)
	}
	return nil
}

// GitDiffIndex returns a list of files that differ from what is committed.
// These could file that were created, deleted, modified, or moved.
func GitDiffIndex() ([]string, error) {
	// Ensure the index is updated so that diff-index gives accurate results.
	if err := sh.Run("git", "update-index", "-q", "--refresh"); err != nil {
		return nil, err
	}

	// git diff-index provides a list of modified files.
	// https://www.git-scm.com/docs/git-diff-index
	out, err := sh.Output("git", "diff-index", "HEAD", "--", ".")
	if err != nil {
		return nil, err
	}

	// Example formats.
	// :100644 100644 bcd1234... 0123456... M file0
	// :100644 100644 abcd123... 1234567... R86 file1 file3
	d, err := dissect.New(":%{src_mode} %{dst_mode} %{src_sha1} %{dst_sha1} %{status}\t%{paths}")
	if err != nil {
		return nil, err
	}

	// Parse lines.
	var modified []string
	s := bufio.NewScanner(bytes.NewBufferString(out))
	for s.Scan() {
		m, err := d.Dissect(s.Text())
		if err != nil {
			return nil, fmt.Errorf("failed to dissect git diff-index output: %w", err)
		}

		paths := strings.Split(m["paths"], "\t")
		if len(paths) > 1 {
			modified = append(modified, paths[1])
		} else {
			modified = append(modified, paths[0])
		}
	}
	if err = s.Err(); err != nil {
		return nil, err
	}

	return modified, nil
}

// GitDiff runs 'git diff' and writes the output to stdout.
func GitDiff() error {
	c := exec.Command("git", "--no-pager", "diff", "--minimal")
	c.Stdin = nil
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	log.Println("exec:", strings.Join(c.Args, " "))
	err := c.Run()
	return err
}
