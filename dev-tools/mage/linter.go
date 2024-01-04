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
	linterVersion    = "v1.55.2"
	linterInstallURL = "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
)

var (
	linterConfigFilename = filepath.Join(".", ".golangci.yml")
	linterInstallDir     = filepath.Join(".", "build")
	linterInstallFile    = filepath.Join(linterInstallDir, "install-golang-ci.sh")
	linterBinaryFile     = filepath.Join(linterInstallDir, linterVersion, "golangci-lint")
)

// Linter contains targets related to linting the Go code
type Linter mg.Namespace

// CheckConfig makes sure that the `.golangci.yml` does not have uncommitted changes
func (Linter) CheckConfig() error {
	err := assertUnchanged(linterConfigFilename)
	if err != nil {
		return fmt.Errorf("linter configuration has uncommitted changes: %w", err)
	}
	return nil
}

// Install installs golangci-lint (https://golangci-lint.run) to `./build`
// using the official installation script downloaded from GitHub.
// If the linter binary already exists does nothing.
func (Linter) Install() error {
	return install(false)
}

// ForceInstall force installs the linter regardless of whether it exists or not.
func (Linter) ForceInstall() error {
	return install(true)
}

func install(force bool) error {
	dirPath := filepath.Dir(linterBinaryFile)
	err := os.MkdirAll(dirPath, 0700)
	if err != nil {
		return fmt.Errorf("failed to create path %q: %w", dirPath, err)
	}

	_, err = os.Stat(linterBinaryFile)
	if !force && err == nil {
		log.Println("The linter has been already installed, skipping...")
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed check if file %q exists: %w", linterBinaryFile, err)
	}

	log.Println("Preparing the installation script file...")

	installScript, err := os.OpenFile(linterInstallFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", linterInstallFile, err)
	}
	defer installScript.Close()

	log.Println("Downloading the linter installation script...")
	//nolint:noctx // valid use since there is no context
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
		return fmt.Errorf("failed to close file %q: %w", linterInstallFile, err)
	}

	binaryDir := filepath.Dir(linterBinaryFile)
	err = os.MkdirAll(binaryDir, 0700)
	if err != nil {
		return fmt.Errorf("cannot create path %q: %w", binaryDir, err)
	}

	// there must be no space after `-b`, otherwise the script does not work correctly ¯\_(ツ)_/¯
	return sh.Run(linterInstallFile, "-b"+binaryDir, linterVersion)
}

// All runs the linter against the entire codebase
func (l Linter) All() error {
	mg.Deps(l.Install, l.CheckConfig)
	return runLinter()
}

// Prints the version of the linter in use.
func (l Linter) Version() error {
	mg.Deps(l.Install)
	return runLinter("--version")
}

// LastChange runs the linter against all files changed since the fork point from `main`.
// If the current branch is `main` then runs against the files changed in the last commit.
func (l Linter) LastChange() error {
	mg.Deps(l.Install, l.CheckConfig)

	// get current branch name
	branch, err := sh.Output("git", "rev-parse", "--abbrev-ref", "HEAD")
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

// runLinter runs the linter passing the `mage -v` (verbose mode) and given arguments.
// Also redirects linter's output to the `stderr` instead of discarding it.
func runLinter(runFlags ...string) error {
	var args []string

	if mg.Verbose() {
		args = append(args, "-v")
	}

	args = append(args, "run")
	args = append(args, runFlags...)
	args = append(args, "-c", linterConfigFilename)
	args = append(args, "./...")

	return runWithStdErr(linterBinaryFile, args...)
}
