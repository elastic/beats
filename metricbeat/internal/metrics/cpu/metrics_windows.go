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
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/gosigar/sys/windows"
)

// Get fetches Windows CPU system times
func Get(_ resolve.Resolver) (CPUMetrics, error) {
	idle, kernel, user, err := windows.GetSystemTimes()
	if err != nil {
		return CPUMetrics{}, errors.Wrap(err, "GetSystemTimes failed")
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
		return CPUMetrics{}, errors.Wrap(err, "NtQuerySystemProcessorPerformanceInformation failed")
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
