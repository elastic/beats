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

//go:build freebsd || linux
// +build freebsd linux

package cpu

import (
	"bufio"
	"os"
	"strconv"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/metric/system/resolve"
)

// Get returns a metrics object for CPU data
func Get(procfs resolve.Resolver) (CPUMetrics, error) {
	path := procfs.ResolveHostFS("/proc/stat")
	fd, err := os.Open(path)
	defer fd.Close()
	if err != nil {
		return CPUMetrics{}, errors.Wrapf(err, "error opening file %s", path)
	}

	return scanStatFile(bufio.NewScanner(fd))

}

// statScanner iterates through a /proc/stat entry, reading both the global lines and per-CPU lines, each time calling lineReader, which implements the OS-specific code for parsing individual lines
func statScanner(scanner *bufio.Scanner, lineReader func(string) (CPU, error)) (CPUMetrics, error) {
	cpuData := CPUMetrics{}
	var err error

	for scanner.Scan() {
		text := scanner.Text()
		// Check to see if this is the global CPU line
		if isCPUGlobalLine(text) {
			cpuData.totals, err = lineReader(text)
			if err != nil {
				return CPUMetrics{}, errors.Wrap(err, "error parsing global CPU line")
			}
		}
		if isCPULine(text) {
			perCPU, err := lineReader(text)
			if err != nil {
				return CPUMetrics{}, errors.Wrap(err, "error parsing CPU line")
			}
			cpuData.list = append(cpuData.list, perCPU)

		}
	}
	return cpuData, nil
}

func isCPUGlobalLine(line string) bool {
	if len(line) > 4 && line[0:4] == "cpu " {
		return true
	}
	return false
}

func isCPULine(line string) bool {
	if len(line) > 3 && line[0:3] == "cpu" && line[3] != ' ' {
		return true
	}
	return false
}

func touint(val string) (uint64, error) {
	return strconv.ParseUint(val, 10, 64)
}
