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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/elastic/elastic-agent-libs/dev-tools/mage/gotool"
)

const (
	// GoImportsImportPath controls the import path used to install goimports.
	GoImportsImportPath = "golang.org/x/tools/cmd/goimports"

	// GoImportsLocalPrefix is a string prefix matching imports that should be
	// grouped after third-party packages.
	GoImportsLocalPrefix = "github.com/elastic"
)

// Linter contains targets related to linting the Go code
type GoImports mg.Namespace

// Run executes goimports against all .go files in and below the CWD.
func (GoImports) Run() error {
	mg.Deps(GoImports.Install)
	goFiles, err := FindFilesRecursive(func(path string, _ os.FileInfo) bool {
		return filepath.Ext(path) == ".go"
	})
	if err != nil {
		return err
	}
	if len(goFiles) == 0 {
		return nil
	}

	fmt.Println(">> fmt - goimports: Formatting Go code") //nolint:forbidigo // it's a mage target
	args := append(
		[]string{"-local", GoImportsLocalPrefix, "-l", "-w"},
		goFiles...,
	)

	return sh.RunV("goimports", args...)
}

func (GoImports) Install() error {
	err := gotool.Install(gotool.Install.Package(filepath.Join(GoImportsImportPath)))
	if err != nil {
		return fmt.Errorf("cannot install GoImports: %w", err)
	}

	return nil
}

// FindFilesRecursive recursively traverses from the CWD and invokes the given
// match function on each regular file to determine if the given path should be
// returned as a match. It ignores files in .git directories.
func FindFilesRecursive(match func(path string, info os.FileInfo) bool) ([]string, error) {
	var matches []string
	walkDir := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't look for files in git directories
		if d.IsDir() && filepath.Base(path) == ".git" {
			return filepath.SkipDir
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("canot get FileInfo: %w", err)
		}
		if !info.Mode().IsRegular() {
			// continue
			return nil
		}

		if match(filepath.ToSlash(path), info) {
			matches = append(matches, path)
		}
		return nil
	}

	err := filepath.WalkDir(".", fs.WalkDirFunc(walkDir))
	return matches, err
}
