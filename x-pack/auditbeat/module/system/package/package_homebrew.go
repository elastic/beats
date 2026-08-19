// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package pkg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// InstallReceiptSource represents the "source" object in Homebrew's INSTALL_RECEIPT.json.
type InstallReceiptSource struct {
	Path string
}

// InstallReceipt represents the JSON object in Homebrew's INSTALL_RECEIPT.json.
type InstallReceipt struct {
	Source InstallReceiptSource
}

func listBrewPackages(brewCellarPath string) ([]*Package, error) {
	packageDirs, err := os.ReadDir(brewCellarPath)
	if err != nil {
		return nil, err
	}

	var packages []*Package
	for _, packageDir := range packageDirs {
		if !packageDir.IsDir() {
			continue
		}
		pkgPath := path.Join(brewCellarPath, packageDir.Name())
		versions, err := os.ReadDir(pkgPath)
		if err != nil {
			return nil, fmt.Errorf("error reading directory: %s: %w", pkgPath, err)
		}

		for _, version := range versions {
			if !version.IsDir() {
				continue
			}

			pkg := &Package{
				Name:    packageDir.Name(),
				Version: version.Name(),
				Type:    "brew",
			}
			packages = append(packages, pkg)

			if info, err := version.Info(); err == nil {
				pkg.InstallTime = info.ModTime()
			}

			// Read formula
			var formulaPath string
			installReceiptPath := path.Join(brewCellarPath, pkg.Name, pkg.Version, "INSTALL_RECEIPT.json")
			contents, err := os.ReadFile(installReceiptPath)
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
				formulaPath = path.Join(brewCellarPath, pkg.Name, pkg.Version, ".brew", pkg.Name+".rb")
			}

			if filepath.Ext(formulaPath) == ".rb" {
				readFormula(pkg, formulaPath)
			}
		}
	}
	return packages, nil
}

// readFormula reads the desc and homepage fields from the first few lines of a
// Homebrew Ruby formula file and stores them in dst.
func readFormula(dst *Package, path string) {
	file, err := os.Open(path)
	if err != nil {
		dst.error = fmt.Errorf("error reading %v: %w", path, err)
		return
	}
	defer file.Close()

	const (
		desc     = "  desc "
		homepage = "  homepage "
	)
	scanner := bufio.NewScanner(file)
	count := 15 // only look into the first few lines of the formula
	for scanner.Scan() {
		count--
		if count == 0 {
			break
		}
		line := scanner.Bytes()
		if after, ok := bytes.CutPrefix(line, []byte(desc)); ok {
			dst.Summary = string(bytes.Trim(after, ` "`))
		} else if after, ok := bytes.CutPrefix(line, []byte(homepage)); ok {
			dst.URL = string(bytes.Trim(after, ` "`))
		}
	}
	if err = scanner.Err(); err != nil {
		dst.error = fmt.Errorf("error parsing %v: %w", path, err)
	}
}
