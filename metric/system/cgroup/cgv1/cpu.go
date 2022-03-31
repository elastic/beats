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

package cgv1

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup/cgcommon"
	"github.com/elastic/beats/v7/libbeat/opt"
)

// CPUSubsystem contains metrics and limits from the "cpu" subsystem. This
// subsystem is used to guarantee a minimum number of cpu shares to the cgroup
// when the system is busy. This subsystem does not track CPU usage, for that
// information see the "cpuacct" subsystem.
type CPUSubsystem struct {
	ID   string `json:"id,omitempty"`   // ID of the cgroup.
	Path string `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.
	// Completely Fair Scheduler (CFS) settings.
	CFS CFS `json:"cfs,omitempty"`
	// Real-time (RT) Scheduler settings.
	RT RT `json:"rt,omitempty"`
	// CPU time statistics for tasks in this cgroup.
	Stats CPUStats `json:"stats,omitempty"`
}

// RT contains the tunable parameters for the real-time scheduler.
type RT struct {
	// Period of time in microseconds for how regularly the cgroup's access to
	// CPU resources should be reallocated.
	Period opt.Us `json:"period" struct:"period"`
	// Period of time in microseconds for the longest continuous period in which
	// the tasks in the cgroup have access to CPU resources.
	Runtime opt.Us `json:"runtime" struct:"runtime"`
}

// CFS contains the tunable parameters for the completely fair scheduler.
type CFS struct {
	// Period of time in microseconds for how regularly the cgroup's access to
	// CPU resources should be reallocated.
	PeriodMicros opt.Us `json:"period" struct:"period"`
	// Total amount of time in microseconds for which all tasks in the cgroup
	// can run during one period.
	QuotaMicros opt.Us `json:"quota" struct:"quota"`
	// Relative share of CPU time available to tasks the cgroup. The value is
	// an integer greater than or equal to 2.
	Shares uint64 `json:"shares"`
}

// CPUStats contains stats that indicate the extent to which this cgroup's
// CPU usage was throttled.
type CPUStats struct {
	// Number of periods with throttling active.
	Periods   uint64         `json:"periods,omitempty"`
	Throttled ThrottledField `json:"throttled" struct:"throttled"`
}

// ThrottledField contains the `throttled` information for the CPU stats
type ThrottledField struct {
	Us      uint64 `json:"us" struct:"us"`
	Periods uint64 `json:"periods" struct:"periods"`
}

// Get reads metrics from the "cpu" subsystem. path is the filepath to the
// cgroup hierarchy to read.
func (cpu *CPUSubsystem) Get(path string) error {
	if err := cpuCFS(path, cpu); err != nil {
		return errors.Wrap(err, "error fetching CFS data")
	}

	if err := cpuRT(path, cpu); err != nil {
		return errors.Wrap(err, "error fetching RT data")
	}

	if err := cpuStat(path, cpu); err != nil {
		return errors.Wrap(err, "error fetching CPU stats")
	}

	return nil
}

func cpuStat(path string, cpu *CPUSubsystem) error {
	f, err := os.Open(filepath.Join(path, "cpu.stat"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t, v, err := cgcommon.ParseCgroupParamKeyValue(sc.Text())
		if err != nil {
			return err
		}
		switch t {
		case "nr_periods":
			cpu.Stats.Periods = v

		case "nr_throttled":
			cpu.Stats.Throttled.Periods = v

		case "throttled_time":
			cpu.Stats.Throttled.Us = v
		}
	}

	return sc.Err()
}

func cpuCFS(path string, cpu *CPUSubsystem) error {
	var err error
	cpu.CFS.PeriodMicros.Us, err = cgcommon.ParseUintFromFile(path, "cpu.cfs_period_us")
	if err != nil {
		return err
	}

	cpu.CFS.QuotaMicros.Us, err = cgcommon.ParseUintFromFile(path, "cpu.cfs_quota_us")
	if err != nil {
		return err
	}

	cpu.CFS.Shares, err = cgcommon.ParseUintFromFile(path, "cpu.shares")
	if err != nil {
		return err
	}

	return nil
}

func cpuRT(path string, cpu *CPUSubsystem) error {
	var err error
	cpu.RT.Period.Us, err = cgcommon.ParseUintFromFile(path, "cpu.rt_period_us")
	if err != nil {
		return err
	}

	cpu.RT.Runtime.Us, err = cgcommon.ParseUintFromFile(path, "cpu.rt_runtime_us")
	if err != nil {
		return err
	}

	return nil
}
