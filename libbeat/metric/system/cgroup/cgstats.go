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
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/metric/system/numcpu"
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/elastic-agent-libs/mapstr"
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

//Format converts the stats object to a MapStr that can be sent to Report()
func (stat StatsV1) Format() (mapstr.M, error) {
	to := mapstr.M{}
	err := typeconv.Convert(&to, stat)
	if err != nil {
		return to, errors.Wrap(err, "error formatting statsV1 object")
	}

	return to, nil
}

// FillPercentages uses a previous CGStats object to fill out the percentage values
// in the cgroup metrics. The `prev` object must be from the same process.
// curTime and Prev time should be time.Time objects that corrispond to the "scrape time" of when the metrics were gathered.
func (curStat *StatsV1) FillPercentages(prev CGStats, curTime, prevTime time.Time) {
	if prev != nil && prev.CGVersion() != CgroupsV1 {
		return
	}
	prevStat := prev.(*StatsV1)

	if prevStat == nil || curStat == nil || curStat.CPUAccounting == nil || prevStat.CPUAccounting == nil {
		return
	}

	timeDelta := curTime.Sub(prevTime)
	timeDeltaNanos := timeDelta / time.Nanosecond
	totalCPUDeltaNanos := int64(curStat.CPUAccounting.Total.NS - prevStat.CPUAccounting.Total.NS)

	pct := float64(totalCPUDeltaNanos) / float64(timeDeltaNanos)
	var cpuCount int
	if len(curStat.CPUAccounting.UsagePerCPU) > 0 {
		cpuCount = len(curStat.CPUAccounting.UsagePerCPU)
	} else {
		cpuCount = numcpu.NumCPU()
	}

	// if you look at the raw cgroup stats, the following normalized value is literally an average of per-cpu numbers.
	normalizedPct := pct / float64(cpuCount)
	userCPUDeltaMillis := int64(curStat.CPUAccounting.Stats.User.NS - prevStat.CPUAccounting.Stats.User.NS)
	systemCPUDeltaMillis := int64(curStat.CPUAccounting.Stats.System.NS - prevStat.CPUAccounting.Stats.System.NS)

	userPct := float64(userCPUDeltaMillis) / float64(timeDeltaNanos)
	systemPct := float64(systemCPUDeltaMillis) / float64(timeDeltaNanos)

	normalizedUser := userPct / float64(cpuCount)
	normalizedSystem := systemPct / float64(cpuCount)

	curStat.CPUAccounting.Total.Pct = opt.FloatWith(common.Round(pct, common.DefaultDecimalPlacesCount))
	curStat.CPUAccounting.Total.Norm.Pct = opt.FloatWith(common.Round(normalizedPct, common.DefaultDecimalPlacesCount))
	curStat.CPUAccounting.Stats.User.Pct = opt.FloatWith(common.Round(userPct, common.DefaultDecimalPlacesCount))
	curStat.CPUAccounting.Stats.User.Norm.Pct = opt.FloatWith(common.Round(normalizedUser, common.DefaultDecimalPlacesCount))
	curStat.CPUAccounting.Stats.System.Pct = opt.FloatWith(common.Round(systemPct, common.DefaultDecimalPlacesCount))
	curStat.CPUAccounting.Stats.System.Norm.Pct = opt.FloatWith(common.Round(normalizedSystem, common.DefaultDecimalPlacesCount))

}

//Format converts the stats object to a MapStr that can be sent to Report()
func (stat StatsV2) Format() (mapstr.M, error) {
	to := mapstr.M{}
	err := typeconv.Convert(&to, stat)
	if err != nil {
		return to, errors.Wrap(err, "error formatting statsV2 object")
	}

	return to, nil
}

// CGVersion returns the version of the underlying cgroups stats
func (stat StatsV2) CGVersion() CgroupsVersion {
	return CgroupsV2
}

// FillPercentages uses a previous CGStats object to fill out the percentage values
// in the cgroup metrics. The `prev` object must be from the same process.
// curTime and Prev time should be time.Time objects that corrispond to the "scrape time" of when the metrics were gathered.
func (curStat *StatsV2) FillPercentages(prev CGStats, curTime, prevTime time.Time) {
	if prev != nil && prev.CGVersion() != CgroupsV2 {
		return
	}
	prevStat := prev.(*StatsV2)

	if prevStat == nil || curStat == nil || curStat.CPU == nil || prevStat.CPU == nil {
		return
	}
	timeDelta := curTime.Sub(prevTime)
	timeDeltaNanos := timeDelta / time.Nanosecond
	totalCPUDeltaNanos := int64(curStat.CPU.Stats.Usage.NS - prevStat.CPU.Stats.Usage.NS)

	pct := float64(totalCPUDeltaNanos) / float64(timeDeltaNanos)

	cpuCount := numcpu.NumCPU()

	// if you look at the raw cgroup stats, the following normalized value is literally an average of per-cpu numbers.
	normalizedPct := pct / float64(cpuCount)
	userCPUDeltaMillis := int64(curStat.CPU.Stats.User.NS - prevStat.CPU.Stats.User.NS)
	systemCPUDeltaMillis := int64(curStat.CPU.Stats.System.NS - prevStat.CPU.Stats.System.NS)

	userPct := float64(userCPUDeltaMillis) / float64(timeDeltaNanos)
	systemPct := float64(systemCPUDeltaMillis) / float64(timeDeltaNanos)

	normalizedUser := userPct / float64(cpuCount)
	normalizedSystem := systemPct / float64(cpuCount)

	curStat.CPU.Stats.Usage.Pct = opt.FloatWith(common.Round(pct, common.DefaultDecimalPlacesCount))
	curStat.CPU.Stats.Usage.Norm.Pct = opt.FloatWith(common.Round(normalizedPct, common.DefaultDecimalPlacesCount))
	curStat.CPU.Stats.User.Pct = opt.FloatWith(common.Round(userPct, common.DefaultDecimalPlacesCount))
	curStat.CPU.Stats.User.Norm.Pct = opt.FloatWith(common.Round(normalizedUser, common.DefaultDecimalPlacesCount))
	curStat.CPU.Stats.System.Pct = opt.FloatWith(common.Round(systemPct, common.DefaultDecimalPlacesCount))
	curStat.CPU.Stats.System.Norm.Pct = opt.FloatWith(common.Round(normalizedSystem, common.DefaultDecimalPlacesCount))
}
