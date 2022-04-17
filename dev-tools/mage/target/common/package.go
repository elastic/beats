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

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
)

// PackageSystemTests packages the python system tests results
func PackageSystemTests() error {
	excludeds := []string{".ci", ".git", ".github", "vendor", "dev-tools"}

	// include run and docker-logs as they are the directories we want to compress
	systemTestsDir := filepath.Join("build", "system-tests")
	systemTestsRunDir := filepath.Join(systemTestsDir, "run")
	systemTestsLogDir := filepath.Join(systemTestsDir, "docker-logs")
	files, err := devtools.FindFilesRecursive(func(path string, _ os.FileInfo) bool {
		base := filepath.Base(path)
		for _, excluded := range excludeds {
			if strings.HasPrefix(base, excluded) {
				return false
			}
		}

		return strings.HasPrefix(path, systemTestsRunDir) || strings.HasPrefix(path, systemTestsLogDir)
	})
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Printf(">> there are no system test files under %s", systemTestsDir)
		return nil
	}

	// create a plain directory layout for all beats
	beat := devtools.MustExpand("{{ repo.SubDir }}")
	beat = strings.ReplaceAll(beat, string(os.PathSeparator), "-")

	targetFile := devtools.MustExpand("{{ elastic_beats_dir }}/build/system-tests-" + beat + ".tar.gz")
	parent := filepath.Dir(targetFile)
	if !fileExists(parent) {
		fmt.Printf(">> creating parent dir: %s", parent)
		os.Mkdir(parent, 0750)
	}

	err = devtools.Tar(systemTestsDir, targetFile)
	if err != nil {
		fmt.Printf(">> %s", err)
		return err
	}

	return nil
}

// fileExists returns true if the specified file exists.
func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
