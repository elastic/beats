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

package cgv2

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup/cgcommon"
)

// IOSubsystem is the replacement for the bulkio controller in cgroupsV1
type IOSubsystem struct {
	ID   string `json:"id,omitempty"`   // ID of the cgroup.
	Path string `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.

	Stats    map[string]IOStat            `json:"stats" struct:"stats"`
	Pressure map[string]cgcommon.Pressure `json:"pressure,omitempty" struct:"pressure,omitempty"`
}

// IOStat carries io.Stat data for the controllers
// This data is broken down per-device, based on the maj-minor device ID
type IOStat struct {
	Read      IOMetric `json:"read" struct:"read"`
	Write     IOMetric `json:"write" struct:"write"`
	Discarded IOMetric `json:"discarded" struct:"discarded"`
}

// IOMetric groups together the common IO sub-metrics by bytes and IOOps count
type IOMetric struct {
	Bytes uint64 `json:"bytes" struct:"bytes"`
	IOs   uint64 `json:"ios" struct:"ios"`
}

// Get fetches metrics for the IO subsystem
// resolveDevIDs determines if Get will try to resolve the major-minor ID pairs reported by io.stat
// are resolved to a device name
func (io *IOSubsystem) Get(path string, resolveDevIDs bool) error {
	var err error
	io.Stats, err = getIOStats(path, resolveDevIDs)
	if err != nil {
		return fmt.Errorf("error getting io.stats for path %s: %w", path, err)
	}

	//Pressure doesn't exist on certain V2 implementations.
	_, err = os.Stat(filepath.Join(path, "io.pressure"))
	if errors.Is(err, os.ErrNotExist) {
		logp.L().Debugf("io.pressure does not exist. Skipping.")
		return nil
	}

	io.Pressure, err = cgcommon.GetPressure(filepath.Join(path, "io.pressure"))
	if err != nil {
		return fmt.Errorf("error fetching io.pressure for path %s: %w", path, err)
	}

	return nil
}

// getIOStats fetches and formats the io.stats object
func getIOStats(path string, resolveDevIDs bool) (map[string]IOStat, error) {
	stats := make(map[string]IOStat)
	file := filepath.Join(path, "io.stat")
	f, err := os.Open(file)
	if err != nil {
		return stats, fmt.Errorf("error reading cpu.stat: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		devices, metrics, foundMetrics, err := parseStatLine(sc.Text(), resolveDevIDs)
		if err != nil {
			return nil, fmt.Errorf("error parsing line in file: %w", err)
		}
		if !foundMetrics {
			continue
		}
		for _, dev := range devices {
			stats[dev] = metrics
		}
	}

	return stats, nil
}

// parses a single line in io.stat; a bit complicated, since these files are more complex then they look.
// returns a list of device names associated with the metrics, the metric set, and a bool indicating if metrics were found
func parseStatLine(line string, resolveDevIDs bool) ([]string, IOStat, bool, error) {
	devIds := []string{}
	stats := IOStat{}
	foundMetrics := false
	// cautiously iterate over a line to find the components
	// under certain conditions, the stat.io will combine different loopback devices onto a single line,
	// 7:7 7:6 7:5 7:4 rbytes=556032 wbytes=0 rios=78 wios=0 dbytes=0 dios=0
	// we can also get lines without metrics, like
	//  7:7 7:6 7:5 7:4
	for _, component := range strings.Split(line, " ") {
		if strings.Contains(component, ":") {
			var major, minor uint64
			_, err := fmt.Sscanf(component, "%d:%d", &major, &minor)
			if err != nil {
				return nil, IOStat{}, false, fmt.Errorf("could not read device ID: %s: %w", component, err)
			}

			var found bool
			var devName string
			// try to find the device name associated with the major/minor pair
			// This isn't guaranteed to work, for a number of reasons, so we'll need to fall back
			if resolveDevIDs {
				found, devName, _ = fetchDeviceName(major, minor)
			}

			if found {
				devIds = append(devIds, devName)
			} else {
				devIds = append(devIds, component)
			}
		} else if strings.Contains(component, "=") {
			foundMetrics = true
			counterSplit := strings.Split(component, "=")
			if len(counterSplit) < 2 {
				continue
			}
			name := counterSplit[0]
			counter, err := strconv.ParseUint(counterSplit[1], 10, 64)
			if err != nil {
				return nil, IOStat{}, false, fmt.Errorf("error parsing counter '%s' in stat: %w", counterSplit[1], err)
			}
			switch name {
			case "rbytes":
				stats.Read.Bytes = counter
			case "wbytes":
				stats.Write.Bytes = counter
			case "rios":
				stats.Read.IOs = counter
			case "wios":
				stats.Write.IOs = counter
			case "dbytes":
				stats.Discarded.Bytes = counter
			case "dios":
				stats.Discarded.IOs = counter
			}

		}

	}
	return devIds, stats, foundMetrics, nil
}
