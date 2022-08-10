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

package numcpu

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// getCPU implements NumCPU on linux
// see https://www.kernel.org/doc/Documentation/admin-guide/cputopology.rst
func getCPU() (int, bool, error) {

	// These are the files that LSCPU looks for
	// This will report online CPUs, which are are the logical CPUS
	// that are currently online and scheduleable by the system.
	// Some users may expect a "present" count, which reflects what
	// CPUs are available to the OS, online or off.
	// These two values will only differ in cases where CPU hotplugging is in affect.
	// This env var swaps between them.
	_, isPresent := os.LookupEnv("LINUX_CPU_COUNT_PRESENT")
	var cpuPath = "/sys/devices/system/cpu/online"
	if isPresent {
		cpuPath = "/sys/devices/system/cpu/present"
	}

	rawFile, err := ioutil.ReadFile(cpuPath)
	// if the file doesn't exist, assume it's a support issue and not a bug
	if errors.Is(err, os.ErrNotExist) {
		return -1, false, nil
	}
	if err != nil {
		return -1, false, errors.Wrapf(err, "error reading file %s", cpuPath)
	}

	cpuCount, err := parseCPUList(string(rawFile))
	if err != nil {
		return -1, false, errors.Wrapf(err, "error parsing file %s", cpuPath)
	}
	return cpuCount, true, nil
}

// parse the weird list files we get from sysfs
func parseCPUList(raw string) (int, error) {

	listPart := strings.Split(raw, ",")
	count := 0
	for _, v := range listPart {
		if strings.Contains(v, "-") {
			rangeC, err := parseCPURange(v)
			if err != nil {
				return 0, errors.Wrapf(err, "error parsing line %s", v)
			}
			count = count + rangeC
		} else {
			count++
		}
	}
	return count, nil
}

func parseCPURange(cpuRange string) (int, error) {
	var first, last int
	_, err := fmt.Sscanf(cpuRange, "%d-%d", &first, &last)
	if err != nil {
		return 0, errors.Wrapf(err, "error reading from range %s", cpuRange)
	}

	return (last - first) + 1, nil
}
