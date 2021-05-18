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

package metrics

import (
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"

	"github.com/elastic/beats/v7/libbeat/common"
)

func Get(_ string) (CPUMetrics, error) {
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
		sys:  uint64(raw.System),
		user: uint64(raw.User),
		idle: uint64(raw.Idle),
		nice: uint64(raw.Nice),
	}
	return totalCPU
}

// fillTicks is the Darwin implementation of fillTicks
func (self CPU) fillTicks(event *common.MapStr) {
	event.Put("user.ticks", self.user)
	event.Put("system.ticks", self.sys)
	event.Put("idle.ticks", self.idle)
	event.Put("nice.ticks", self.nice)
}

// fillCPUMetrics is the Darwin implementation of fillCPUTicks
func fillCPUMetrics(event *common.MapStr, current, prev CPU, numCPU int, timeDelta uint64, pathPostfix string) {
	idleTime := cpuMetricTimeDelta(prev.idle, current.idle, timeDelta, numCPU)
	totalPct := common.Round(float64(numCPU)-idleTime, common.DefaultDecimalPlacesCount)

	event.Put("total"+pathPostfix, totalPct)
	event.Put("user"+pathPostfix, cpuMetricTimeDelta(prev.user, current.user, timeDelta, numCPU))
	event.Put("system"+pathPostfix, cpuMetricTimeDelta(prev.sys, current.sys, timeDelta, numCPU))
	event.Put("idle"+pathPostfix, cpuMetricTimeDelta(prev.idle, current.idle, timeDelta, numCPU))
	event.Put("nice"+pathPostfix, cpuMetricTimeDelta(prev.nice, current.nice, timeDelta, numCPU))
}
