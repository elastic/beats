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
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
)

func Check() error {
	mg.Deps(CheckNoBeatsDependency)
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
