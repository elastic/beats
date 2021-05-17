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
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/gosigar/sys/windows"
)

// Get fetches Windows CPU system times
func Get(_ string) (CPUMetrics, error) {
	idle, kernel, user, err := windows.GetSystemTimes()
	if err != nil {
		return CPUMetrics{}, errors.Wrap(err, "GetSystemTimes failed")
	}

	metrics := CPUMetrics{}
	//convert from duration to ticks
	metrics.totals.idle = uint64(idle / time.Millisecond)
	metrics.totals.sys = uint64(kernel / time.Millisecond)
	metrics.totals.user = uint64(user / time.Millisecond)

	// get per-cpu data
	cpus, err := windows.NtQuerySystemProcessorPerformanceInformation()
	if err != nil {
		return CPUMetrics{}, errors.Wrap(err, "NtQuerySystemProcessorPerformanceInformation failed")
	}
	metrics.list = make([]CPU, 0, len(cpus))
	for _, cpu := range cpus {
		metrics.list = append(metrics.list, CPU{
			idle: uint64(cpu.IdleTime / time.Millisecond),
			sys:  uint64(cpu.KernelTime / time.Millisecond),
			user: uint64(cpu.UserTime / time.Millisecond),
		})
	}

	return metrics, nil
}

// fillTicks is the Windows implementation of FillTicks
func (self CPUMetrics) fillTicks(event *common.MapStr) {
	event.Put("user.ticks", self.totals.user)
	event.Put("system.ticks", self.totals.sys)
	event.Put("idle.ticks", self.totals.idle)
}

// fillCPUMetrics is the Windows implementation of fillCPUMetrics
func fillCPUMetrics(event *common.MapStr, current, prev CPUMetrics, numCPU int, timeDelta uint64, pathPostfix string) {
	idleTime := cpuMetricTimeDelta(prev.totals.idle, current.totals.idle, timeDelta, numCPU)
	totalPct := common.Round(float64(numCPU)-idleTime, common.DefaultDecimalPlacesCount)

	event.Put("total"+pathPostfix, totalPct)
	event.Put("user"+pathPostfix, cpuMetricTimeDelta(prev.totals.user, current.totals.user, timeDelta, numCPU))
	event.Put("system"+pathPostfix, cpuMetricTimeDelta(prev.totals.sys, current.totals.sys, timeDelta, numCPU))
	event.Put("idle"+pathPostfix, cpuMetricTimeDelta(prev.totals.idle, current.totals.idle, timeDelta, numCPU))
}
