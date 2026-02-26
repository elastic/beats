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
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup/cgcommon"
)

// CPUSubsystem contains metrics and limits from the "cpu" subsystem.
// in cgroupsV2, this merges both the 'cpu' and 'cpuacct' controllers.
type CPUSubsystem struct {
	ID   string `json:"id,omitempty"`   // ID of the cgroup.
	Path string `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.
	// Shows pressure stall information for CPU.
	Pressure map[string]cgcommon.Pressure `json:"pressure,omitempty" struct:"pressure,omitempty"`
	// Stats shows overall counters for the CPU controller
	Stats CPUStats `json:"stats" struct:"stats"`
	// CFS contains CPU bandwidth control settings.
	CFS CFS `json:"cfs,omitzero" struct:"cfs,omitempty"`
}

// CFS contains CPU bandwidth control settings from cgroup v2 (cpu.max and cpu.weight).
// This is equivalent to the CFS struct in cgroups v1, but uses Weight instead of Shares.
type CFS struct {
	// Period in microseconds for how regularly the cgroup's access to CPU resources is reallocated.
	Period UsOpt `json:"period,omitzero" struct:"period,omitempty"`
	// Quota in microseconds for which all tasks in the cgroup can run during one period.
	// A value of 0 indicates unlimited (cpu.max "max").
	Quota UsOpt `json:"quota,omitzero" struct:"quota,omitempty"`
	// Relative CPU weight (1-10000, default 100). This replaces cpu.shares from cgroups v1.
	Weight opt.Uint `json:"weight,omitzero" struct:"weight,omitempty"`
}

// UsOpt wraps opt.Uint for optional microsecond values.
// Analogous to opt.BytesOpt; when unset, serializes as nil/omitted rather than 0.
type UsOpt struct {
	Us opt.Uint `json:"us" struct:"us"`
}

// IsZero returns true when the value has not been set.
func (u UsOpt) IsZero() bool {
	return u.Us.IsZero()
}

// CPUStats carries the information from the cpu.stat cgroup file
type CPUStats struct {
	//The following three metrics are only available when the controller is enabled.
	Throttled ThrottledField    `json:"throttled,omitzero" struct:"throttled,omitempty"`
	Periods   opt.Uint          `json:"periods,omitzero" struct:"periods,omitempty"`
	Usage     cgcommon.CPUUsage `json:"usage" struct:"usage"`
	User      cgcommon.CPUUsage `json:"user" struct:"user"`
	System    cgcommon.CPUUsage `json:"system" struct:"system"`
}

// ThrottledField contains the `throttled` information for the CPU stats
type ThrottledField struct {
	Us      opt.Uint `json:"us,omitzero" struct:"us,omitempty"`
	Periods opt.Uint `json:"periods,omitzero" struct:"periods,omitempty"`
}

// IsZero implements the IsZero interface for ThrottledField
func (t ThrottledField) IsZero() bool {
	return t.Us.IsZero() && t.Periods.IsZero()
}

// Get fetches CPU subsystem metrics for V2 cgroups
func (cpu *CPUSubsystem) Get(path string) error {

	var err error
	cpu.Pressure, err = cgcommon.GetPressure(filepath.Join(path, "cpu.pressure"))
	// Not all systems have pressure stats. Treat this as a soft error.
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		return fmt.Errorf("error fetching Pressure data: %w", err)
	}

	cpu.Stats, err = getStats(path)
	if err != nil {
		return fmt.Errorf("error fetching CPU stat data: %w", err)
	}

	cpu.CFS, err = getCFS(path)
	if err != nil {
		return fmt.Errorf("error fetching CFS data: %w", err)
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
		return CPUStats{}, fmt.Errorf("error reading cpu.stat: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	data := CPUStats{}
	for sc.Scan() {
		key, val, err := cgcommon.ParseCgroupParamKeyValue(sc.Text())
		if err != nil {
			return data, fmt.Errorf("error parsing cpu.stat file: %w", err)
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

// parseCPUMax parses "quota period" from cpu.max.
// quota may be "max" (unlimited) or an integer in microseconds; "max" maps to 0.
func parseCPUMax(path string) (quota, period uint64, err error) {
	contents, err := os.ReadFile(filepath.Join(path, "cpu.max"))
	if err != nil {
		return 0, 0, err
	}

	fields := strings.Fields(strings.TrimSpace(string(contents)))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected format in cpu.max: expected 2 fields, got %d", len(fields))
	}

	// quota can be "max" (unlimited) or an integer
	if fields[0] == "max" {
		quota = 0 // 0 indicates unlimited
	} else {
		quota, err = strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("error parsing quota from cpu.max: %w", err)
		}
	}

	// period is always an integer
	period, err = strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing period from cpu.max: %w", err)
	}

	return quota, period, nil
}

// getCFS reads cpu.max/cpu.weight into CFS. Missing files are treated as soft errors.
func getCFS(path string) (CFS, error) {
	cfs := CFS{}

	// Parse cpu.max for quota and period
	quota, period, err := parseCPUMax(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return cfs, fmt.Errorf("error reading cpu.max: %w", err)
		}
		// File doesn't exist - continue without it
	} else {
		cfs.Quota = UsOpt{Us: opt.UintWith(quota)}
		cfs.Period = UsOpt{Us: opt.UintWith(period)}
	}

	// Parse cpu.weight
	weightPath := filepath.Join(path, "cpu.weight")
	if _, statErr := os.Stat(weightPath); statErr == nil {
		weight, err := cgcommon.ParseUintFromFile(weightPath)
		if err != nil {
			return cfs, fmt.Errorf("error reading cpu.weight: %w", err)
		}
		cfs.Weight = opt.UintWith(weight)
	} else if !os.IsNotExist(statErr) {
		return cfs, fmt.Errorf("error reading cpu.weight: %w", statErr)
	}

	return cfs, nil
}
