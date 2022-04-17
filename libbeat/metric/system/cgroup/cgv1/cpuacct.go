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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/metric/system/cgroup/cgcommon"
	"github.com/menderesk/gosigar/sys/linux"
)

var clockTicks = uint64(linux.GetClockTicks())

// CPUAccountingSubsystem contains metrics from the "cpuacct" subsystem.
// Note that percentage values are not taken from cgroup metrics, but derived via FillPercentages()
type CPUAccountingSubsystem struct {
	ID          string            `json:"id,omitempty"`   // ID of the cgroup.
	Path        string            `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.
	Total       cgcommon.CPUUsage `json:"total_nanos"`
	UsagePerCPU map[string]uint64 `json:"percpu" struct:"percpu"`
	// CPU time statistics for tasks in this cgroup.
	Stats CPUAccountingStats `json:"stats,omitempty"`
}

// CPUAccountingStats contains the stats reported from the cpuacct subsystem.
type CPUAccountingStats struct {
	User   cgcommon.CPUUsage `json:"user" struct:"user"`
	System cgcommon.CPUUsage `json:"system" struct:"system"`
}

// Get reads metrics from the "cpuacct" subsystem. path is the filepath to the
// cgroup hierarchy to read.
func (cpuacct *CPUAccountingSubsystem) Get(path string) error {
	cpuacct.UsagePerCPU = make(map[string]uint64)
	if err := cpuacctStat(path, cpuacct); err != nil {
		return errors.Wrap(err, "error fetching cpuacct stats")
	}

	if err := cpuacctUsage(path, cpuacct); err != nil {
		return errors.Wrap(err, "error fetching cpuacct usage")
	}

	if err := cpuacctUsagePerCPU(path, cpuacct); err != nil {
		return errors.Wrap(err, "error fetching per_cpu data")
	}

	return nil
}

func cpuacctStat(path string, cpuacct *CPUAccountingSubsystem) error {
	f, err := os.Open(filepath.Join(path, "cpuacct.stat"))
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
		case "user":
			cpuacct.Stats.User.NS = convertJiffiesToNanos(v)
		case "system":
			cpuacct.Stats.System.NS = convertJiffiesToNanos(v)
		}
	}

	return sc.Err()
}

func cpuacctUsage(path string, cpuacct *CPUAccountingSubsystem) error {
	var err error
	cpuacct.Total.NS, err = cgcommon.ParseUintFromFile(path, "cpuacct.usage")
	if err != nil {
		return err
	}

	return nil
}

func cpuacctUsagePerCPU(path string, cpuacct *CPUAccountingSubsystem) error {
	contents, err := ioutil.ReadFile(filepath.Join(path, "cpuacct.usage_percpu"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	values := make(map[string]uint64)
	usages := bytes.Fields(contents)
	for cpu, usage := range usages {
		value, err := cgcommon.ParseUint(usage)
		if err != nil {
			return err
		}

		// For backwards compatibility, We start the CPU count at 1
		cpuStr := fmt.Sprintf("%d", cpu+1)
		values[cpuStr] = value
	}
	cpuacct.UsagePerCPU = values

	return nil
}

func convertJiffiesToNanos(j uint64) uint64 {
	return (j * uint64(time.Second)) / clockTicks
}
