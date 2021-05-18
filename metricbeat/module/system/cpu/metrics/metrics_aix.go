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

/*
#cgo LDFLAGS: -L/usr/lib -lperfstat

#include <libperfstat.h>
#include <procinfo.h>
#include <unistd.h>
#include <utmp.h>
#include <sys/mntctl.h>
#include <sys/proc.h>
#include <sys/types.h>
#include <sys/vmount.h>

*/
import "C"

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
)

func init() {
	// sysconf(_SC_CLK_TCK) returns the number of ticks by second.
	system.ticks = uint64(C.sysconf(C._SC_CLK_TCK))
	system.pagesize = uint64(os.Getpagesize())
}

var system struct {
	ticks    uint64
	btime    uint64
	pagesize uint64
}

func tick2msec(val uint64) uint64 {
	return val * 1000 / system.ticks
}

// Get returns a metrics object for CPU data
func Get(_ string) (CPUMetrics, error) {

	totals, err := getCPUTotals()
	if err != nil {
		return CPUMetrics{}, errors.Wrap(err, "error getting CPU totals")
	}

	list, err := getPerCPUMetrics()
	if err != nil {
		return CPUMetrics{}, errors.Wrap(err, "error getting per-cpu metrics")
	}

	return CPUMetrics{totals: totals, list: list}, nil

}

// fillTicks is the AIX implementation of FillTicks
func (self CPU) fillTicks(event *common.MapStr) {
	event.Put("user.ticks", self.user)
	event.Put("system.ticks", self.sys)
	event.Put("idle.ticks", self.idle)
	event.Put("wait.ticks", self.wait)
}

// fillCPUMetrics is the AIX implementation of *Percentages()
func fillCPUMetrics(event *common.MapStr, current, prev CPU, numCPU int, timeDelta uint64, pathPostfix string) {
	idleTime := cpuMetricTimeDelta(prev.idle, current.idle, timeDelta, numCPU) + cpuMetricTimeDelta(prev.wait, current.wait, timeDelta, numCPU)
	totalPct := common.Round(float64(numCPU)-idleTime, common.DefaultDecimalPlacesCount)

	event.Put("total"+pathPostfix, totalPct)
	event.Put("user"+pathPostfix, cpuMetricTimeDelta(prev.user, current.user, timeDelta, numCPU))
	event.Put("system"+pathPostfix, cpuMetricTimeDelta(prev.sys, current.sys, timeDelta, numCPU))
	event.Put("idle"+pathPostfix, cpuMetricTimeDelta(prev.idle, current.idle, timeDelta, numCPU))
	event.Put("wait"+pathPostfix, cpuMetricTimeDelta(prev.wait, current.wait, timeDelta, numCPU))
}

func getCPUTotals() (CPU, error) {
	cpudata := C.perfstat_cpu_total_t{}

	if _, err := C.perfstat_cpu_total(nil, &cpudata, C.sizeof_perfstat_cpu_total_t, 1); err != nil {
		return CPU{}, fmt.Errorf("perfstat_cpu_total: %s", err)
	}

	totals := CPU{}
	totals.user = tick2msec(uint64(cpudata.user))
	totals.sys = tick2msec(uint64(cpudata.sys))
	totals.idle = tick2msec(uint64(cpudata.idle))
	totals.wait = tick2msec(uint64(cpudata.wait))

	return totals, nil
}

func getPerCPUMetrics() ([]CPU, error) {
	cpudata := C.perfstat_cpu_t{}
	id := C.perfstat_id_t{}
	id.name[0] = 0

	// Retrieve the number of cpu using perfstat_cpu
	capacity, err := C.perfstat_cpu(nil, nil, C.sizeof_perfstat_cpu_t, 0)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving CPU number: %s", err)
	}
	list := make([]CPU, 0, capacity)

	for {
		if _, err := C.perfstat_cpu(&id, &cpudata, C.sizeof_perfstat_cpu_t, 1); err != nil {
			return nil, fmt.Errorf("perfstat_cpu: %s", err)
		}

		cpu := CPU{}
		cpu.user = tick2msec(uint64(cpudata.user))
		cpu.sys = tick2msec(uint64(cpudata.sys))
		cpu.idle = tick2msec(uint64(cpudata.idle))
		cpu.wait = tick2msec(uint64(cpudata.wait))

		list = append(list, cpu)

		if id.name[0] == 0 {
			break
		}
	}

	return list, nil

}
