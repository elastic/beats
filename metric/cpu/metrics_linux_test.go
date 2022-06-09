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

package cpu

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestScanCPUInfoFile tests the parsing of `/proc/cpuinfo` for different
// system/CPU configurations. The lscpu GitHub contains a nice set of
// test files: https://github.com/util-linux/util-linux/tree/master/tests/ts/lscpu/dumps
func TestScanCPUInfoFile(t *testing.T) {
	testCases := []string{
		"cpuinfo",
		"cpuinfo-quad-socket",
		// Source: https://github.com/util-linux/util-linux/blob/master/tests/ts/lscpu/dumps/armv7.tar.gz
		"cpuinfo-armv7",
	}
	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			sourceFd, err := os.Open(filepath.Join("testdata", tc))
			if err != nil {
				t.Fatalf("cannot open test file: %s", err)
			}
			defer sourceFd.Close()

			scanner := bufio.NewScanner(sourceFd)
			cpuInfo, err := scanCPUInfoFile(scanner)
			if err != nil {
				t.Fatalf("scanCPUInfoFile error: %s", err)
			}

			// Ignoring the error, because if there is any parsing error, generateGoldenFile
			// will be false, making the test to run as expected
			if generateGoldenFile, _ := strconv.ParseBool(os.Getenv("GENERATE")); generateGoldenFile {
				t.Logf("generating golden files for test: %s", t.Name())
				scanCPUInfoFileGenGoldenFile(t, cpuInfo, tc)
				return
			}

			expectedFd, err := os.Open(filepath.Join("testdata", tc+".expected.json"))
			if err != nil {
				t.Fatalf("cannot open test expectation file: %s", err)
			}
			defer expectedFd.Close()

			expected := []CPUInfo{}
			if err := json.NewDecoder(expectedFd).Decode(&expected); err != nil {
				t.Fatalf("cannot decode goldenfile data: %s", err)
			}

			assert.Equal(t, expected, cpuInfo)
		})
	}
}

func scanCPUInfoFileGenGoldenFile(t *testing.T, data []CPUInfo, name string) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("cannot marshal data into JSON: %s", err)
	}

	// Add a line break at the end
	jsonData = append(jsonData, '\n')

	expectedFd, err := os.Create(filepath.Join("testdata", name+".expected.json"))
	if err != nil {
		t.Fatalf("cannot open/create test expectation file: %s", err)
	}
	defer expectedFd.Close()

	if _, err := expectedFd.Write(jsonData); err != nil {
		t.Fatalf("cannot write data to goldenfile: %s", err)
	}
}
