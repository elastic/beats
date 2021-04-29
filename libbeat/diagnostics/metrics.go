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

package diagnostics

import (
	"encoding/json"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

func getMetrics(diag Diagnostics) {
	diag.Metrics.Timestamp = time.Now()
	//if diag.Docker.IsContainer {
	//	diag.Docker.Timestamp = time.Now()
	//	getDockerCPUMetrics(&diag)
	//}
	diag.Logger.Debug("Metrics collection interval")
	getMemoryMetrics(&diag)
	getCPUMetrics(&diag)
	getRoutineCount(&diag)
	getAvgLoad(&diag)
	getNetworkMetrics(&diag)
	getDiskMetrics(&diag)
	mjson, err := json.Marshal(diag.Metrics)
	if err != nil {
		diag.Logger.Error("Failed to marshal beats metrics")
	}
	writeToFile(diag.DiagFolder, "metrics.json", mjson)
}

func getMemoryMetrics(diag *Diagnostics) *Diagnostics {
	s, err := mem.SwapMemoryWithContext(diag.Context)
	if err != nil {
		diag.Logger.Error("Unable to find swap stats")
	}
	diag.Metrics.Swap = s
	vm, err := mem.VirtualMemoryWithContext(diag.Context)
	if err != nil {
		diag.Logger.Error("Unable to find memory stats")
	}
	diag.Metrics.Memory = vm
	return diag
}

func getCPUMetrics(diag *Diagnostics) *Diagnostics {
	cs, err := cpu.TimesWithContext(diag.Context, false)
	if err != nil {
		diag.Logger.Error("Unable to find CPU stats")
	}
	diag.Metrics.CPUStats = cs
	return diag
}

func getAvgLoad(diag *Diagnostics) *Diagnostics {
	al, err := load.AvgWithContext(diag.Context)
	if err != nil {
		diag.Logger.Error("Unable to find average CPU stats")
	}
	diag.Metrics.AvgLoad = al
	return diag
}

func getNetworkMetrics(diag *Diagnostics) *Diagnostics {
	cio, err := net.IOCountersWithContext(diag.Context, true)
	if err != nil {
		diag.Logger.Error("Unable to find network IO stats")
	}
	diag.Metrics.Network.IO = cio
	return diag
}

func getDiskMetrics(diag *Diagnostics) *Diagnostics {
	partitions, err := disk.Partitions(true)
	if err != nil {
		diag.Logger.Error("Unable to find list of partitions")
	}
	names := []string{}
	for _, p := range partitions {
		names = append(names, p.Device)
	}
	dm, err := disk.IOCounters(names...)
	if err != nil {
		diag.Logger.Error("Unable to find disk IO stats")
	}
	diag.Metrics.Disk.Stats = dm
	return diag
}

func getRoutineCount(diag *Diagnostics) *Diagnostics {
	diag.Metrics.NumGoroutine = runtime.NumGoroutine()
	return diag
}
