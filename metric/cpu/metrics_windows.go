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

/*
For testing via the win2012 vagrant box:
vagrant winrm -s cmd -e -c "cd C:\\Gopath\src\\github.com\\elastic\\beats\\metricbeat\\module\\system\\cpu; go test -v -tags=integration -run TestFetch"  win2012
*/

package cpu

import (
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-libs/helpers/windows/pdh"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/gosigar/sys/windows"
)

var (
	processorInformationCounter = "\\Processor Information(%s)\\%s"
	totalKernelTimeCounter      = fmt.Sprintf(processorInformationCounter, "*", "% Privileged Time")
	totalIdleTimeCounter        = fmt.Sprintf(processorInformationCounter, "*", "% Idle Time")
	totalUserTimeCounter        = fmt.Sprintf(processorInformationCounter, "*", "% User Time")
)

// Get fetches Windows CPU system times
func Get(m *Monitor) (CPUMetrics, error) {
	if m.query == nil {
		return getUsingSystemTimes()
	}
	return getUsingPerfCounters(m.query)
}

func getUsingSystemTimes() (CPUMetrics, error) {
	idle, kernel, user, err := windows.GetSystemTimes()
	if err != nil {
		return CPUMetrics{}, fmt.Errorf("call to GetSystemTimes failed: %w", err)
	}

	globalMetrics := CPUMetrics{}
	//convert from duration to ticks
	idleMetric := uint64(idle / time.Millisecond)
	sysMetric := uint64(kernel / time.Millisecond)
	userMetrics := uint64(user / time.Millisecond)
	globalMetrics.totals.Idle = opt.UintWith(idleMetric)
	globalMetrics.totals.Sys = opt.UintWith(sysMetric)
	globalMetrics.totals.User = opt.UintWith(userMetrics)

	// get per-cpu data
	cpus, err := windows.NtQuerySystemProcessorPerformanceInformation()
	if err != nil {
		return CPUMetrics{}, fmt.Errorf("catll to NtQuerySystemProcessorPerformanceInformation failed: %w", err)
	}
	globalMetrics.list = make([]CPU, 0, len(cpus))
	for _, cpu := range cpus {
		idleMetric := uint64(cpu.IdleTime / time.Millisecond)
		sysMetric := uint64(cpu.KernelTime / time.Millisecond)
		userMetrics := uint64(cpu.UserTime / time.Millisecond)
		globalMetrics.list = append(globalMetrics.list, CPU{
			Idle: opt.UintWith(idleMetric),
			Sys:  opt.UintWith(sysMetric),
			User: opt.UintWith(userMetrics),
		})
	}

	return globalMetrics, nil
}

func getUsingPerfCounters(query *pdh.Query) (CPUMetrics, error) {
	globalMetrics := CPUMetrics{}

	if err := query.CollectData(); err != nil {
		return globalMetrics, err
	}

	kernelRawData, err := query.GetRawCounterArray(totalKernelTimeCounter, true)
	if err != nil {
		return globalMetrics, fmt.Errorf("error calling GetRawCounterArray for kernel counter: %w", err)
	}
	idleRawData, err := query.GetRawCounterArray(totalIdleTimeCounter, true)
	if err != nil {
		return globalMetrics, fmt.Errorf("error calling GetRawCounterArray for idle counter: %w", err)
	}
	userRawData, err := query.GetRawCounterArray(totalUserTimeCounter, true)
	if err != nil {
		return globalMetrics, fmt.Errorf("error calling GetRawCounterArray for user counter: %w", err)
	}
	var idle, kernel, user time.Duration
	globalMetrics.list = make([]CPU, len(userRawData))
	for i := 0; i < len(globalMetrics.list); i++ {
		// The values returned by GetRawCounterArray are of equal length and are sorted by instance names.
		// For CPU core {i}, idleRawData[i], kernelRawData[i], and userRawData[i] correspond to the idle time, kernel time, and user time, respectively.

		// values returned by counter are in 100-ns intervals. Hence, convert it to millisecond.
		idleTime := time.Duration(idleRawData[i].RawValue.FirstValue*100) / time.Millisecond
		kernelTime := time.Duration(kernelRawData[i].RawValue.FirstValue*100) / time.Millisecond
		userTime := time.Duration(userRawData[i].RawValue.FirstValue*100) / time.Millisecond

		globalMetrics.list[i].Idle = opt.UintWith(uint64(idleTime))
		globalMetrics.list[i].Sys = opt.UintWith(uint64(kernelTime))
		globalMetrics.list[i].User = opt.UintWith(uint64(userTime))

		// add the per-cpu time to track the total time spent by system
		idle += idleTime
		kernel += kernelTime
		user += userTime
	}

	globalMetrics.totals.Idle = opt.UintWith(uint64(idle))
	globalMetrics.totals.Sys = opt.UintWith(uint64(kernel))
	globalMetrics.totals.User = opt.UintWith(uint64(user))

	return globalMetrics, nil
}
