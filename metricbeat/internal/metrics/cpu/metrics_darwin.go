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
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v3/cpu"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
)

// Get is the Darwin implementation of Get
func Get(_ resolve.Resolver) (CPUMetrics, error) {
	// We're using the gopsutil library here.
	// The code used by both gosigar and go-sysinfo appears to be
	// the same code as gopsutil, including copy-pasted comments.
	// For the sake of just reducing complexity,
	sum, err := cpu.Times(false)
	if err != nil {
		return CPUMetrics{}, errors.Wrap(err, "error fetching CPU summary data")
	}
	perCPU, err := cpu.Times(true)
	if err != nil {
		return CPUMetrics{}, errors.Wrap(err, "error fetching per-CPU data")
	}

	cpulist := []CPU{}
	for _, cpu := range perCPU {
		cpulist = append(cpulist, fillCPU(cpu))
	}
	return CPUMetrics{totals: fillCPU(sum[0]), list: cpulist}, nil
}

func fillCPU(raw cpu.TimesStat) CPU {
	totalCPU := CPU{
		Sys:  opt.UintWith(uint64(raw.System)),
		User: opt.UintWith(uint64(raw.User)),
		Idle: opt.UintWith(uint64(raw.Idle)),
		Nice: opt.UintWith(uint64(raw.Nice)),
	}
	return totalCPU
}
