// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package pkg

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// InstallReceiptSource represents the "source" object in Homebrew's INSTALL_RECEIPT.json.
type InstallReceiptSource struct {
	Path string
}

// InstallReceipt represents the JSON object in Homebrew's INSTALL_RECEIPT.json.
type InstallReceipt struct {
	Source InstallReceiptSource
}

func listBrewPackages() ([]*Package, error) {
	packageDirs, err := ioutil.ReadDir(homebrewCellarPath)
	if err != nil {
		return nil, err
	}

	var packages []*Package
	for _, packageDir := range packageDirs {
		if !packageDir.IsDir() {
			continue
		}
		pkgPath := path.Join(homebrewCellarPath, packageDir.Name())
		versions, err := ioutil.ReadDir(pkgPath)
		if err != nil {
			return nil, fmt.Errorf("error reading directory: %s: %w", pkgPath, err)
		}

		for _, version := range versions {
			if !version.IsDir() {
				continue
			}
			pkg := &Package{
				Name:        packageDir.Name(),
				Version:     version.Name(),
				InstallTime: version.ModTime(),
				Type:        "brew",
			}

			// Read formula
			var formulaPath string
			installReceiptPath := path.Join(homebrewCellarPath, pkg.Name, pkg.Version, "INSTALL_RECEIPT.json")
			contents, err := ioutil.ReadFile(installReceiptPath)
			if err != nil {
				pkg.error = fmt.Errorf("error reading %v: %w", installReceiptPath, err)
			} else {
				var installReceipt InstallReceipt
				err = json.Unmarshal(contents, &installReceipt)
				if err != nil {
					pkg.error = fmt.Errorf("error unmarshalling JSON in %v: %w", installReceiptPath, err)
				} else {
					formulaPath = installReceipt.Source.Path
				}
			}

			if formulaPath == "" {
				// Fallback to /usr/local/Cellar/{pkg.Name}/{pkg.Version}/.brew/{pkg.Name}.rb
				formulaPath = path.Join(homebrewCellarPath, pkg.Name, pkg.Version, ".brew", pkg.Name+".rb")
			}

			file, err := os.Open(formulaPath)
			if err != nil {
				pkg.error = fmt.Errorf("error reading %v: %w", formulaPath, err)
			} else {
				defer file.Close()

				scanner := bufio.NewScanner(file)
				count := 15 // only look into the first few lines of the formula
				for scanner.Scan() {
					count--
					if count == 0 {
						break
					}
					line := scanner.Text()
					if strings.HasPrefix(line, "  desc ") {
						pkg.Summary = strings.Trim(line[7:], " \"")
					} else if strings.HasPrefix(line, "  homepage ") {
						pkg.URL = strings.Trim(line[11:], " \"")
					}
				}
				if err = scanner.Err(); err != nil {
					pkg.error = fmt.Errorf("error parsing %v: %w", formulaPath, err)
				}
			}

			packages = append(packages, pkg)
		}
	}
	return packages, nil
}
