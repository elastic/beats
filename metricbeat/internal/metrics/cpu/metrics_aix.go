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

	"github.com/elastic/beats/v8/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v8/libbeat/opt"
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
	ticks := val * 1000 / system.ticks
	return ticks
}

// Get returns a metrics object for CPU data
func Get(_ resolve.Resolver) (CPUMetrics, error) {

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

// getCPUTotals gets the global CPU stats
func getCPUTotals() (CPU, error) {
	cpudata := C.perfstat_cpu_total_t{}

	if _, err := C.perfstat_cpu_total(nil, &cpudata, C.sizeof_perfstat_cpu_total_t, 1); err != nil {
		return CPU{}, fmt.Errorf("perfstat_cpu_total: %s", err)
	}

	totals := CPU{}
	totals.User = opt.UintWith((uint64(cpudata.user)))
	totals.Sys = opt.UintWith(tick2msec(uint64(cpudata.sys)))
	totals.Idle = opt.UintWith(tick2msec(uint64(cpudata.idle)))
	totals.Wait = opt.UintWith(tick2msec(uint64(cpudata.wait)))

	return totals, nil
}

// getPerCPUMetrics gets per-CPU metrics
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
		cpu.User = opt.UintWith(tick2msec(uint64(cpudata.user)))
		cpu.Sys = opt.UintWith(tick2msec(uint64(cpudata.sys)))
		cpu.Idle = opt.UintWith(tick2msec(uint64(cpudata.idle)))
		cpu.Wait = opt.UintWith(tick2msec(uint64(cpudata.wait)))

		list = append(list, cpu)

		if id.name[0] == 0 {
			break
		}
	}

	return list, nil

}
