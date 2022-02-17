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
	linterInstallFilename = "./intall-golang-ci.sh"
	linterBinaryFilename  = "./bin/golangci-lint"
	linterVersion         = "v1.44.0"
)

// InstallLinter installs golangci-lint (https://golangci-lint.run) to `./bin`
// using the official installation script downloaded from GitHub.
// If the linter binary already exists does nothing.
func InstallLinter() error {
	_, err := os.Stat(linterBinaryFilename)
	if err == nil {
		log.Println("already installed, exiting...")
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	log.Println("preparing the installation script file...")
	installScript, err := os.OpenFile(linterInstallFilename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	defer installScript.Close()

	log.Println("downloading the linter installation script...")
	resp, err := http.Get("https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	lr := io.LimitReader(resp.Body, 1024*100) // not more than 100 KB
	_, err = io.Copy(installScript, lr)
	if err != nil {
		return err
	}
	return sh.Run(linterInstallFilename, linterVersion)
}

// LintAll runs the linter against the entire codebase
func LintAll() error {
	mg.Deps(InstallLinter)
	return sh.Run(linterBinaryFilename, "-v", "run", "./...")
}

func Check() error {
	mg.Deps(CheckNoBeatsDependency, CheckModuleTidy, LintAll)
	return nil
}

// CheckNoBeatsDependency is required to make sure we are not introducing
// dependency on elastic/beats.
func CheckNoBeatsDependency() error {
	goModPath, err := filepath.Abs("go.mod")
	if err != nil {
		return err
	}
	goModFile, err := os.Open(goModPath)
	if err != nil {
		return fmt.Errorf("failed to open module file: %+v", err)
	}
	beatsImport := []byte("github.com/elastic/beats")
	scanner := bufio.NewScanner(goModFile)
	lineCount := 1
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.Contains(line, beatsImport) {
			return fmt.Errorf("line %d is beats dependency: '%s'\nPlease, make sure you are not adding anything that depends on %s", lineCount, line, beatsImport)
		}
		lineCount++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// CheckModuleTidy checks if `go mod tidy` was run before.
func CheckModuleTidy() error {
	err := sh.Run("go", "mod", "tidy")
	if err != nil {
		return err
	}
	err = sh.Run("git", "diff", "--exit-code")
	if err != nil {
		return fmt.Errorf("`go mod tidy` was not called before committing: %w", err)
	}

	return nil
}
