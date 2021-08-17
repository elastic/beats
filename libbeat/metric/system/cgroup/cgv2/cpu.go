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
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup/cgcommon"
	"github.com/elastic/beats/v7/libbeat/opt"
)

// CPUSubsystem contains metrics and limits from the "cpu" subsystem.
// in cgroupsV2, this merges both the 'cpu' and 'cpuacct' controllers.
type CPUSubsystem struct {
	ID   string `json:"id,omitempty"`   // ID of the cgroup.
	Path string `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.
	// Shows pressure stall information for CPU.
	Pressure map[string]cgcommon.Pressure `json:"pressure,omitempty" struct:"pressure,omitempty"`
	// Stats shows overall counters for the CPU controller
	Stats CPUStats
}

// CPUStats carries the information from the cpu.stat cgroup file
type CPUStats struct {
	//The following three metrics are only available when the controller is enabled.
	Throttled ThrottledField    `json:"throttled,omitempty" struct:"throttled,omitempty"`
	Periods   opt.Uint          `json:"periods,omitempty" struct:"periods,omitempty"`
	Usage     cgcommon.CPUUsage `json:"usage" struct:"usage"`
	User      cgcommon.CPUUsage `json:"user" struct:"user"`
	System    cgcommon.CPUUsage `json:"system" struct:"system"`
}

// ThrottledField contains the `throttled` information for the CPU stats
type ThrottledField struct {
	Us      opt.Uint `json:"us,omitempty" struct:"us,omitempty"`
	Periods opt.Uint `json:"periods,omitempty" struct:"periods,omitempty"`
}

// IsZero implements the IsZero interface for ThrottledField
func (t ThrottledField) IsZero() bool {
	return t.Us.IsZero() && t.Periods.IsZero()
}

// Get fetches memory subsystem metrics for V2 cgroups
func (cpu *CPUSubsystem) Get(path string) error {

	var err error
	cpu.Pressure, err = cgcommon.GetPressure(filepath.Join(path, "cpu.pressure"))
	// Not all systems have pressure stats. Treat this as a soft error.
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "error fetching Pressure data")
	}

	cpu.Stats, err = getStats(path)
	if err != nil {
		return errors.Wrap(err, "error fetching CPU stat data")
	}

	return nil
}

// getStats returns the cpu.stats data
func getStats(path string) (CPUStats, error) {
	f, err := os.Open(filepath.Join(path, "cpu.stat"))
	if err != nil {
		if os.IsNotExist(err) {
			return CPUStats{}, nil
		}
		return CPUStats{}, errors.Wrap(err, "error reading cpu.stat")
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	data := CPUStats{}
	for sc.Scan() {
		key, val, err := cgcommon.ParseCgroupParamKeyValue(sc.Text())
		if err != nil {
			return data, errors.Wrap(err, "error parsing cpu.stat file")
		}
		switch key {
		case "usage_usec":
			data.Usage.NS = val
		case "user_usec":
			data.User.NS = val
		case "system_usec":
			data.System.NS = val
		case "nr_periods":
			data.Periods = opt.UintWith(val)
		case "nr_throttled":
			data.Throttled.Periods = opt.UintWith(val)
		case "throttled_usec":
			data.Throttled.Us = opt.UintWith(val)
		}
	}

	return data, nil
}
