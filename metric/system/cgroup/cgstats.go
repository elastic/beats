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

package cgroup

import (
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-libs/transform/typeconv"
	"github.com/elastic/elastic-agent-system-metrics/metric"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/numcpu"
)

// CGStats in an interface wrapper around the V2 and V1 cgroup stat objects
type CGStats interface {
	Format() (mapstr.M, error)
	CGVersion() CgroupsVersion
	FillPercentages(prev CGStats, curTime, prevTime time.Time)
}

// CGVersion returns the version of the underlying cgroups stats
func (stat StatsV1) CGVersion() CgroupsVersion {
	return CgroupsV1
}

// Format converts the stats object to a MapStr that can be sent to Report()
func (stat StatsV1) Format() (mapstr.M, error) {
	to := mapstr.M{}
	err := typeconv.Convert(&to, stat)
	if err != nil {
		return to, fmt.Errorf("error formatting statsV1 object: %w", err)
	}

	return to, nil
}

// FillPercentages uses a previous CGStats object to fill out the percentage values
// in the cgroup metrics. The `prev` object must be from the same process.
// curTime and Prev time should be time.Time objects that correspond to the "scrape time" of when the metrics were gathered.
func (stat *StatsV1) FillPercentages(prev CGStats, curTime, prevTime time.Time) {
	if prev != nil && prev.CGVersion() != CgroupsV1 {
		return
	}
	prevStat, ok := prev.(*StatsV1)

	if !ok || prevStat == nil || stat == nil || stat.CPUAccounting == nil || prevStat.CPUAccounting == nil {
		return
	}

	timeDelta := curTime.Sub(prevTime)
	timeDeltaNanos := timeDelta / time.Nanosecond
	totalCPUDeltaNanos := int64(stat.CPUAccounting.Total.NS - prevStat.CPUAccounting.Total.NS)

	pct := float64(totalCPUDeltaNanos) / float64(timeDeltaNanos)
	var cpuCount int
	if len(stat.CPUAccounting.UsagePerCPU) > 0 {
		cpuCount = len(stat.CPUAccounting.UsagePerCPU)
	} else {
		cpuCount = numcpu.NumCPU()
	}

	// if you look at the raw cgroup stats, the following normalized value is literally an average of per-cpu numbers.
	normalizedPct := pct / float64(cpuCount)
	userCPUDeltaMillis := int64(stat.CPUAccounting.Stats.User.NS - prevStat.CPUAccounting.Stats.User.NS)
	systemCPUDeltaMillis := int64(stat.CPUAccounting.Stats.System.NS - prevStat.CPUAccounting.Stats.System.NS)

	userPct := float64(userCPUDeltaMillis) / float64(timeDeltaNanos)
	systemPct := float64(systemCPUDeltaMillis) / float64(timeDeltaNanos)

	normalizedUser := userPct / float64(cpuCount)
	normalizedSystem := systemPct / float64(cpuCount)

	stat.CPUAccounting.Total.Pct = opt.FloatWith(metric.Round(pct))
	stat.CPUAccounting.Total.Norm.Pct = opt.FloatWith(metric.Round(normalizedPct))
	stat.CPUAccounting.Stats.User.Pct = opt.FloatWith(metric.Round(userPct))
	stat.CPUAccounting.Stats.User.Norm.Pct = opt.FloatWith(metric.Round(normalizedUser))
	stat.CPUAccounting.Stats.System.Pct = opt.FloatWith(metric.Round(systemPct))
	stat.CPUAccounting.Stats.System.Norm.Pct = opt.FloatWith(metric.Round(normalizedSystem))
}

// Format converts the stats object to a MapStr that can be sent to Report()
func (stat StatsV2) Format() (mapstr.M, error) {
	to := mapstr.M{}
	err := typeconv.Convert(&to, stat)
	if err != nil {
		return to, fmt.Errorf("error formatting statsV2 object: %w", err)
	}

	return to, nil
}

// CGVersion returns the version of the underlying cgroups stats
func (stat StatsV2) CGVersion() CgroupsVersion {
	return CgroupsV2
}

// FillPercentages uses a previous CGStats object to fill out the percentage values
// in the cgroup metrics. The `prev` object must be from the same process.
// curTime and Prev time should be time.Time objects that correspond to the "scrape time" of when the metrics were gathered.
func (stat *StatsV2) FillPercentages(prev CGStats, curTime, prevTime time.Time) {
	if prev != nil && prev.CGVersion() != CgroupsV2 {
		return
	}
	prevStat, ok := prev.(*StatsV2)

	if !ok || prevStat == nil || stat == nil || stat.CPU == nil || prevStat.CPU == nil {
		return
	}
	timeDelta := curTime.Sub(prevTime)
	timeDeltaNanos := timeDelta / time.Nanosecond
	totalCPUDeltaNanos := int64(stat.CPU.Stats.Usage.NS - prevStat.CPU.Stats.Usage.NS)

	pct := float64(totalCPUDeltaNanos) / float64(timeDeltaNanos)

	cpuCount := numcpu.NumCPU()

	// if you look at the raw cgroup stats, the following normalized value is literally an average of per-cpu numbers.
	normalizedPct := pct / float64(cpuCount)
	userCPUDeltaMillis := int64(stat.CPU.Stats.User.NS - prevStat.CPU.Stats.User.NS)
	systemCPUDeltaMillis := int64(stat.CPU.Stats.System.NS - prevStat.CPU.Stats.System.NS)

	userPct := float64(userCPUDeltaMillis) / float64(timeDeltaNanos)
	systemPct := float64(systemCPUDeltaMillis) / float64(timeDeltaNanos)

	normalizedUser := userPct / float64(cpuCount)
	normalizedSystem := systemPct / float64(cpuCount)

	stat.CPU.Stats.Usage.Pct = opt.FloatWith(metric.Round(pct))
	stat.CPU.Stats.Usage.Norm.Pct = opt.FloatWith(metric.Round(normalizedPct))
	stat.CPU.Stats.User.Pct = opt.FloatWith(metric.Round(userPct))
	stat.CPU.Stats.User.Norm.Pct = opt.FloatWith(metric.Round(normalizedUser))
	stat.CPU.Stats.System.Pct = opt.FloatWith(metric.Round(systemPct))
	stat.CPU.Stats.System.Norm.Pct = opt.FloatWith(metric.Round(normalizedSystem))
}
