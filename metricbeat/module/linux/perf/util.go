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

package perf

import (
	"github.com/hodgesds/perf-utils"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/process"
)

// matchProcesses takes a config list and returns a list of associated processes.
// This is basically a search, so a single process term could return multiple processes.
// the ioctls that underpin perf require a pid.
func matchProcesses(procList []sampleConfig) ([]procInfo, error) {
	var monitorProcs []procInfo
	for _, proc := range procList {

		config := &process.Stats{Procs: []string{proc.ProcessGlob}}

		err := config.Init()
		if err != nil {
			return nil, errors.Wrap(err, "error initializing process list")
		}

		matches, err := config.Get()
		if err != nil {
			return nil, errors.Wrap(err, "Erorr fetching matching processes")
		}

		for _, match := range matches {
			pi := procInfo{}
			pid := match["pid"].(int)

			// Events are summed across all CPUs.
			if proc.Events.HardwareEvents {
				hw := perf.NewHardwareProfiler(pid, -1)
				pi.HardwareProc = hw
			}
			if proc.Events.SoftwareEvents {
				sw := perf.NewSoftwareProfiler(pid, -1)
				pi.SoftwareProc = sw
			}

			pi.PID = pid
			pi.Metadata = match
			monitorProcs = append(monitorProcs, pi)
		}

	} // end of proc iteration

	return monitorProcs, nil
}
