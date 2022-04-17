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
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/metric/system/cgroup/cgcommon"
)

//IOSubsystem is the replacement for the bulkio controller in cgroupsV1
type IOSubsystem struct {
	ID   string `json:"id,omitempty"`   // ID of the cgroup.
	Path string `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.

	Stats    map[string]IOStat            `json:"stats" struct:"stats"`
	Pressure map[string]cgcommon.Pressure `json:"pressure" struct:"pressure"`
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
		return errors.Wrapf(err, "error getting io.stats for path %s", path)
	}

	//Pressure doesn't exist on certain V2 implementations.
	_, err = os.Stat(filepath.Join(path, "io.pressure"))
	if errors.Is(err, os.ErrNotExist) {
		logp.L().Debugf("io.pressure does not exist. Skipping.")
		return nil
	}

	io.Pressure, err = cgcommon.GetPressure(filepath.Join(path, "io.pressure"))
	if err != nil {
		return errors.Wrapf(err, "error fetching io.pressure for path %s:", path)
	}

	return nil
}

// getIOStats fetches and formats the io.stats object
func getIOStats(path string, resolveDevIDs bool) (map[string]IOStat, error) {
	stats := make(map[string]IOStat)
	file := filepath.Join(path, "io.stat")
	f, err := os.Open(file)
	if err != nil {
		return stats, errors.Wrap(err, "error reading cpu.stat")
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		devMetric := IOStat{}
		var major, minor uint64
		_, err := fmt.Sscanf(sc.Text(), "%d:%d rbytes=%d wbytes=%d rios=%d wios=%d dbytes=%d dios=%d", &major, &minor, &devMetric.Read.Bytes, &devMetric.Write.Bytes, &devMetric.Read.IOs, &devMetric.Write.IOs, &devMetric.Discarded.Bytes, &devMetric.Discarded.IOs)
		if err != nil {
			return stats, errors.Wrapf(err, "error scanning file: %s", file)
		}

		// try to find the device name associated with the major/minor pair
		// This isn't guarenteed to work, for a number of reasons, so we'll need to fall back
		var found bool
		var devName string
		if resolveDevIDs {
			found, devName, err = fetchDeviceName(major, minor)
			if err != nil {
				return nil, errors.Wrapf(err, "error looking up device ID %d:%d", major, minor)
			}
		}

		if found {
			stats[devName] = devMetric
		} else {
			idKey := fmt.Sprintf("%d:%d", major, minor)
			stats[idKey] = devMetric
		}
	}

	return stats, nil
}
