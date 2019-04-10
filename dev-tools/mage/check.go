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
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/processors/dissect"
)

// Check looks for created/modified/deleted/renamed files and returns an error
// if it finds any modifications. If executed in in verbose mode it will write
// the results of 'git diff' to stdout to indicate what changes have been made.
//
// It checks the file permissions of nosetests test cases and YAML files.
// It checks .go source files using 'go vet'.
func Check() error {
	fmt.Println(">> check: Checking source code for common problems")

	mg.Deps(GoVet, CheckNosetestsNotExecutable, CheckYAMLNotExecutable)

	changes, err := GitDiffIndex()
	if err != nil {
		return errors.Wrap(err, "failed to diff the git index")
	}

	if len(changes) > 0 {
		if mg.Verbose() {
			GitDiff()
		}

		return errors.Errorf("some files are not up-to-date. "+
			"Run 'mage fmt update' then review and commit the changes. "+
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
			return nil, errors.Wrap(err, "failed to dissect git diff-index output")
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

// CheckNosetestsNotExecutable checks that none of the nosetests files are
// executable. Nosetests silently skips executable .py files and we don't want
// this to happen.
func CheckNosetestsNotExecutable() error {
	if runtime.GOOS == "windows" {
		// Skip windows because it doesn't have POSIX permissions.
		return nil
	}

	tests, err := FindFiles(nosetestsTestFiles...)
	if err != nil {
		return err
	}

	var executableTestFiles []string
	for _, file := range tests {
		info, err := os.Stat(file)
		if err != nil {
			return err
		}

		if info.Mode().Perm()&0111 > 0 {
			executableTestFiles = append(executableTestFiles, file)
		}
	}

	if len(executableTestFiles) > 0 {
		return errors.Errorf("nosetests files cannot be executable because "+
			"they will be skipped. Fix permissions of %v", executableTestFiles)
	}
	return nil
}

// CheckYAMLNotExecutable checks that no .yml or .yaml files are executable.
func CheckYAMLNotExecutable() error {
	if runtime.GOOS == "windows" {
		// Skip windows because it doesn't have POSIX permissions.
		return nil
	}

	executableYAMLFiles, err := FindFilesRecursive(func(path string, info os.FileInfo) bool {
		switch filepath.Ext(path) {
		default:
			return false
		case ".yml", ".yaml":
			return info.Mode().Perm()&0111 > 0
		}
	})
	if err != nil {
		return errors.Wrap(err, "failed search for YAML files")
	}

	if len(executableYAMLFiles) > 0 {
		return errors.Errorf("YAML files cannot be executable. Fix "+
			"permissions of %v", executableYAMLFiles)

	}
	return nil
}

// GoVet vets the .go source code using 'go vet'.
func GoVet() error {
	err := sh.RunV("go", "vet", "./...")
	return errors.Wrap(err, "failed running go vet, please fix the issues reported")
}
