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
	"bufio"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
)

// fillTicks is the linux implementation of FillTicks
func (self CPUMetrics) fillTicks(event *common.MapStr) {
	event.Put("user.ticks", self.totals.user)
	event.Put("system.ticks", self.totals.sys)
	event.Put("idle.ticks", self.totals.idle)
	event.Put("nice.ticks", self.totals.nice)
	event.Put("irq.ticks", self.totals.irq)
	event.Put("iowait.ticks", self.totals.wait)
	event.Put("softirq.ticks", self.totals.softIrq)
	event.Put("steal.ticks", self.totals.stolen)

}

func fillCPUMetrics(event *common.MapStr, current, prev CPUMetrics, numCPU int, timeDelta uint64, pathPostfix string) {
	// IOWait time is excluded from the total as per #7627.
	idleTime := cpuMetricTimeDelta(prev.totals.idle, current.totals.idle, timeDelta, numCPU) + cpuMetricTimeDelta(prev.totals.wait, current.totals.wait, timeDelta, numCPU)
	totalPct := common.Round(float64(numCPU)-idleTime, common.DefaultDecimalPlacesCount)

	event.Put("total"+pathPostfix, totalPct)
	event.Put("user"+pathPostfix, cpuMetricTimeDelta(prev.totals.user, current.totals.user, timeDelta, numCPU))
	event.Put("system"+pathPostfix, cpuMetricTimeDelta(prev.totals.sys, current.totals.sys, timeDelta, numCPU))
	event.Put("idle"+pathPostfix, cpuMetricTimeDelta(prev.totals.idle, current.totals.idle, timeDelta, numCPU))
	event.Put("nice"+pathPostfix, cpuMetricTimeDelta(prev.totals.nice, current.totals.nice, timeDelta, numCPU))
	event.Put("irq"+pathPostfix, cpuMetricTimeDelta(prev.totals.irq, current.totals.irq, timeDelta, numCPU))
	event.Put("softirq"+pathPostfix, cpuMetricTimeDelta(prev.totals.softIrq, current.totals.softIrq, timeDelta, numCPU))
	event.Put("iowait"+pathPostfix, cpuMetricTimeDelta(prev.totals.wait, current.totals.wait, timeDelta, numCPU))
	event.Put("steal"+pathPostfix, cpuMetricTimeDelta(prev.totals.stolen, current.totals.stolen, timeDelta, numCPU))
}

func scanStatFile(scanner *bufio.Scanner) (CPUMetrics, error) {
	cpuData, err := statScanner(scanner, parseCPULine)
	if err != nil {
		return CPUMetrics{}, errors.Wrap(err, "error scanning stat file")
	}
	return cpuData, nil
}

func parseCPULine(line string) (CPU, error) {
	cpuData := CPU{}
	fields := strings.Fields(line)
	var errs multierror.Errors
	var err error

	cpuData.user, err = touint(fields[1])
	if err != nil {
		errs = append(errs, err)
	}
	cpuData.nice, err = touint(fields[2])
	if err != nil {
		errs = append(errs, err)
	}
	cpuData.sys, err = touint(fields[3])
	if err != nil {
		errs = append(errs, err)
	}
	cpuData.idle, err = touint(fields[4])
	if err != nil {
		errs = append(errs, err)
	}
	cpuData.wait, err = touint(fields[5])
	if err != nil {
		errs = append(errs, err)
	}
	cpuData.irq, err = touint(fields[6])
	if err != nil {
		errs = append(errs, err)
	}
	cpuData.softIrq, err = touint(fields[7])
	if err != nil {
		errs = append(errs, err)
	}
	cpuData.stolen, err = touint(fields[8])
	if err != nil {
		errs = append(errs, err)
	}
	return cpuData, errs.Err()
}
