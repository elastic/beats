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

// skipping tests on windows 32 bit versions, not supported
//go:build !windows && !386
// +build !windows,!386

package data

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/elastic/beats/v7/metricbeat/include"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestAll(t *testing.T) {
	configFiles, _ := filepath.Glob(getModulesPath() + "/*/*/_meta/testdata/config.yml")

	for _, f := range configFiles {
		// get module and metricset name from path
		s := strings.Split(f, string(os.PathSeparator))
		moduleName := s[4]
		metricSetName := s[5]

		t.Run(fmt.Sprintf("%s.%s", moduleName, metricSetName), func(t *testing.T) {

			if runtime.GOOS == "aix" && (moduleName == "docker" || moduleName == "kubernetes") {
				t.Skipf("%s module not available on AIX", moduleName)

			} else {
				config := mbtest.ReadDataConfig(t, f)
				mbtest.TestDataFilesWithConfig(t, moduleName, metricSetName, config)
			}
		})
	}
}

func getModulesPath() string {
	return "../../../module"
}
