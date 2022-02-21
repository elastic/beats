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

//go:build mage
// +build mage

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	linterInstallURL      = "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
	linterInstallFilename = "./build/intall-golang-ci.sh"
	linterBinaryFilename  = "./build/golangci-lint"
	linterVersion         = "v1.44.0"
)

// Aliases are shortcuts to long target names.
// nolint: deadcode // it's used by `mage`.
var Aliases = map[string]interface{}{
	"llc":  Linter.LastChange,
	"lint": Linter.All,
}

// Linter contains targets related to linting the Go code
type Linter mg.Namespace

// Install installs golangci-lint (https://golangci-lint.run) to `./build`
// using the official installation script downloaded from GitHub.
// If the linter binary already exists does nothing.
func (Linter) Install() error {
	dirPath := filepath.Dir(linterBinaryFilename)
	err := os.MkdirAll(dirPath, 0700)
	if err != nil {
		return fmt.Errorf("failed to create path %q: %w", dirPath, err)
	}

	_, err = os.Stat(linterBinaryFilename)
	if err == nil {
		log.Println("the linter has been already installed, skipping...")
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed check if file %q exists: %w", linterBinaryFilename, err)
	}

	log.Println("preparing the installation script file...")

	installScript, err := os.OpenFile(linterInstallFilename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", linterInstallFilename, err)
	}
	defer installScript.Close()

	log.Println("downloading the linter installation script...")
	// nolint: noctx // valid use since there is no context
	resp, err := http.Get(linterInstallURL)
	if err != nil {
		return fmt.Errorf("cannot download the linter installation script from %q: %w", linterInstallURL, err)
	}
	defer resp.Body.Close()

	lr := io.LimitReader(resp.Body, 1024*100) // not more than 100 KB, just to be safe
	_, err = io.Copy(installScript, lr)
	if err != nil {
		return fmt.Errorf("failed to finish downloading the linter installation script: %w", err)
	}

	err = installScript.Close() // otherwise we cannot run the script
	if err != nil {
		return fmt.Errorf("failed to close file %q: %w", linterInstallFilename, err)
	}

	binaryDir := filepath.Dir(linterBinaryFilename)
	err = os.MkdirAll(binaryDir, 0700)
	if err != nil {
		return fmt.Errorf("cannot create path %q: %w", binaryDir, err)
	}

	// there must be no space after `-b`, otherwise the script does not work correctly ¯\_(ツ)_/¯
	return sh.Run(linterInstallFilename, "-b"+binaryDir, linterVersion)
}

// All runs the linter against the entire codebase
func (l Linter) All() error {
	mg.Deps(l.Install)
	return runLinter()
}

// LastChange runs the linter against all files changed since the fork point from `main`.
// If the current branch is `main` then runs against the files changed in the last commit.
func (l Linter) LastChange() error {
	mg.Deps(l.Install)

	branch, err := sh.Output("git", "branch", "--show-current")
	if err != nil {
		return fmt.Errorf("failed to get the current branch: %w", err)
	}

	// the linter is supposed to support linting changed diffs only but,
	// for some reason, it simply does not work - does not output any
	// results without linting the whole files, so we have to use `--whole-files`
	// which can lead to some frustration from developers who would like to
	// fix a single line in an existing codebase and the linter would force them
	// into fixing all linting issues in the whole file instead

	if branch == "main" {
		// files changed in the last commit
		return runLinter("--new-from-rev=HEAD~", "--whole-files")
	}

	return runLinter("--new-from-rev=origin/main", "--whole-files")
}

// Check runs all the checks
// nolint: deadcode,unparam // it's used as a `mage` target and requires returning an error
func Check() error {
	mg.Deps(Deps.CheckNoBeats, Deps.CheckModuleTidy, Linter.LastChange)
	return nil
}

// Deps contains targets related to checking dependencies
type Deps mg.Namespace

// CheckNoBeats is required to make sure we are not introducing
// dependency on elastic/beats.
func (Deps) CheckNoBeats() error {
	goModPath, err := filepath.Abs("go.mod")
	if err != nil {
		return err
	}
	goModFile, err := os.Open(goModPath)
	if err != nil {
		return fmt.Errorf("failed to open module file: %w", err)
	}
	beatsImport := []byte("github.com/elastic/beats")
	scanner := bufio.NewScanner(goModFile)
	lineCount := 1
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.Contains(line, beatsImport) {
			return fmt.Errorf("line %d is a beats dependency: '%s'\nPlease, make sure you are not adding anything that depends on %s", lineCount, line, beatsImport)
		}
		lineCount++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// CheckModuleTidy checks if `go mod tidy` was run before the last commit.
func (Deps) CheckModuleTidy() error {
	err := sh.Run("go", "mod", "tidy")
	if err != nil {
		return err
	}
	err = sh.Run("git", "diff", "--exit-code", "go.mod")
	if err != nil {
		return fmt.Errorf("`go mod tidy` was not called before the last commit: %w", err)
	}

	return nil
}

// runWithStdErr runs a command redirecting its stderr to the console instead of discarding it
func runWithStdErr(command string, args ...string) error {
	_, err := sh.Exec(nil, os.Stdout, os.Stderr, command, args...)
	return err
}

// runLinter runs the linter passing the `mage -v` (verbose mode) and given arguments.
// Also redirects linter's output to the `stderr` instead of discarding it.
func runLinter(runFlags ...string) error {
	var args []string

	if mg.Verbose() {
		args = append(args, "-v")
	}

	args = append(args, "run")
	args = append(args, runFlags...)
	args = append(args, "./...")

	return runWithStdErr(linterBinaryFilename, args...)
}
