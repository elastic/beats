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
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
)

var (
	// GoImportsImportPath controls the import path used to install goimports.
	GoImportsImportPath = "github.com/elastic/beats/vendor/golang.org/x/tools/cmd/goimports"

	// GoImportsLocalPrefix is a string prefix matching imports that should be
	// grouped after third-party packages.
	GoImportsLocalPrefix = "github.com/elastic"

	// GoLicenserImportPath controls the import path used to install go-licenser.
	GoLicenserImportPath = "github.com/elastic/go-licenser"
)

// Format adds license headers, formats .go files with goimports, and formats
// .py files with autopep8.
func Format() {
	// Don't run AddLicenseHeaders and GoImports concurrently because they
	// both can modify the same files.
	mg.Deps(AddLicenseHeaders)
	mg.Deps(GoImports, PythonAutopep8)
}

// GoImports executes goimports against all .go files in and below the CWD. It
// ignores vendor/ directories.
func GoImports() error {
	goFiles, err := FindFilesRecursive(func(path string, _ os.FileInfo) bool {
		return filepath.Ext(path) == ".go" && !strings.Contains(path, "vendor/")
	})
	if err != nil {
		return err
	}
	if len(goFiles) == 0 {
		return nil
	}

	fmt.Println(">> fmt - goimports: Formatting Go code")
	if err := sh.Run("go", "get", GoImportsImportPath); err != nil {
		return err
	}

	args := append(
		[]string{"-local", GoImportsLocalPrefix, "-l", "-w"},
		goFiles...,
	)

	return sh.RunV("goimports", args...)
}

// PythonAutopep8 executes autopep8 on all .py files in and below the CWD. It
// ignores build/ directories.
func PythonAutopep8() error {
	pyFiles, err := FindFilesRecursive(func(path string, _ os.FileInfo) bool {
		return filepath.Ext(path) == ".py" &&
			!strings.Contains(path, "build/") &&
			!strings.Contains(path, "vendor/")
	})
	if err != nil {
		return err
	}
	if len(pyFiles) == 0 {
		return nil
	}

	fmt.Println(">> fmt - autopep8: Formatting Python code")
	ve, err := PythonVirtualenv()
	if err != nil {
		return err
	}

	autopep8, err := LookVirtualenvPath(ve, "autopep8")
	if err != nil {
		return err
	}

	args := append(
		[]string{"--in-place", "--max-line-length", "120"},
		pyFiles...,
	)

	return sh.RunV(autopep8, args...)
}

// AddLicenseHeaders adds license headers to .go files. It applies the
// appropriate license header based on the value of mage.BeatLicense.
func AddLicenseHeaders() error {
	fmt.Println(">> fmt - go-licenser: Adding missing headers")

	if err := sh.Run("go", "get", GoLicenserImportPath); err != nil {
		return err
	}

	var license string
	switch BeatLicense {
	case "ASL2", "ASL 2.0":
		license = "ASL2"
	case "Elastic", "Elastic License":
		license = "Elastic"
	default:
		return errors.Errorf("unknown license type %v", BeatLicense)
	}

	return sh.RunV("go-licenser", "-license", license)
}
