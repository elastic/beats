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
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Get returns a metrics object for CPU data
func Get(m *Monitor) (CPUMetrics, error) {
	procfs := m.Hostfs

	path := procfs.ResolveHostFS("/proc/stat")
	fd, err := os.Open(path)
	defer func() {
		_ = fd.Close()
	}()
	if err != nil {
		return CPUMetrics{}, fmt.Errorf("error opening file %s: %w", path, err)
	}

	metrics, err := scanStatFile(bufio.NewScanner(fd))
	if err != nil {
		return CPUMetrics{}, fmt.Errorf("scanning stat file: %w", err)
	}

	cpuInfoPath := procfs.ResolveHostFS("/proc/cpuinfo")
	cpuInfoFd, err := os.Open(cpuInfoPath)
	if err != nil {
		return CPUMetrics{}, fmt.Errorf("opening '%s': %w", cpuInfoPath, err)
	}
	defer cpuInfoFd.Close()

	cpuInfo, err := scanCPUInfoFile(bufio.NewScanner(cpuInfoFd))
	metrics.CPUInfo = cpuInfo

	return metrics, err
}

func cpuinfoScanner(scanner *bufio.Scanner) ([]CPUInfo, error) {
	cpuInfos := []CPUInfo{}
	current := CPUInfo{}
	// On my tests the order the cores appear on /proc/cpuinfo
	// is the same as on /proc/stats, this means it matches our
	// current 'system.core.id' metric. This information
	// is also the same as the 'processor' line on /proc/cpuinfo.
	coreID := 0
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, ":")
		if len(split) != 2 {
			// A blank line its a separation between CPUs
			// even the last CPU contains one blank line at the end
			cpuInfos = append(cpuInfos, current)
			current = CPUInfo{}
			coreID++

			continue
		}

		k, v := split[0], split[1]
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "model":
			current.ModelNumber = v
		case "model name":
			current.ModelName = v
		case "physical id":
			id, err := strconv.Atoi(v)
			if err != nil {
				return []CPUInfo{}, fmt.Errorf("parsing physical ID: %w", err)
			}
			current.PhysicalID = id
		case "core id":
			id, err := strconv.Atoi(v)
			if err != nil {
				return []CPUInfo{}, fmt.Errorf("parsing core ID: %w", err)
			}
			current.CoreID = id
		case "cpu MHz":
			mhz, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return []CPUInfo{}, fmt.Errorf("parsing CPU %d Mhz: %w", coreID, err)
			}
			current.Mhz = mhz
		}
	}

	return cpuInfos, nil
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
				return CPUMetrics{}, fmt.Errorf("error parsing global CPU line: %w", err)
			}
		}
		if isCPULine(text) {
			perCPU, err := lineReader(text)
			if err != nil {
				return CPUMetrics{}, fmt.Errorf("error parsing CPU line: %w", err)
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
