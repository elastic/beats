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

import (
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/module/docker"
)

type CPUStats struct {
	Time                                  common.Time
	Container                             *docker.Container
	PerCPUUsage                           common.MapStr
	TotalUsage                            float64
	TotalUsageNormalized                  float64
	UsageInKernelmode                     uint64
	UsageInKernelmodePercentage           float64
	UsageInKernelmodePercentageNormalized float64
	UsageInUsermode                       uint64
	UsageInUsermodePercentage             float64
	UsageInUsermodePercentageNormalized   float64
	SystemUsage                           uint64
	SystemUsagePercentage                 float64
	SystemUsagePercentageNormalized       float64
}

// CPUService is a helper to collect docker CPU metrics
type CPUService struct {
	Cores bool
}

func NewCpuService() *CPUService {
	return &CPUService{}
}

func (c *CPUService) getCPUStatsList(rawStats []docker.Stat, dedot bool) []CPUStats {
	formattedStats := []CPUStats{}

	for _, stats := range rawStats {
		formattedStats = append(formattedStats, c.getCPUStats(&stats, dedot))
	}

	return formattedStats
}

func (c *CPUService) getCPUStats(myRawStat *docker.Stat, dedot bool) CPUStats {
	usage := cpuUsage{Stat: myRawStat}

	stats := CPUStats{
		Time:                                  common.Time(myRawStat.Stats.Read),
		Container:                             docker.NewContainer(myRawStat.Container, dedot),
		TotalUsage:                            usage.Total(),
		TotalUsageNormalized:                  usage.TotalNormalized(),
		UsageInKernelmode:                     myRawStat.Stats.CPUStats.CPUUsage.UsageInKernelmode,
		UsageInKernelmodePercentage:           usage.InKernelMode(),
		UsageInKernelmodePercentageNormalized: usage.InKernelModeNormalized(),
		UsageInUsermode:                       myRawStat.Stats.CPUStats.CPUUsage.UsageInUsermode,
		UsageInUsermodePercentage:             usage.InUserMode(),
		UsageInUsermodePercentageNormalized:   usage.InUserModeNormalized(),
		SystemUsage:                           myRawStat.Stats.CPUStats.SystemUsage,
		SystemUsagePercentage:                 usage.System(),
		SystemUsagePercentageNormalized:       usage.SystemNormalized(),
	}

	if c.Cores {
		stats.PerCPUUsage = usage.PerCPU()
	}

	return stats
}

// TODO: These helper should be merged with the cpu helper in system/cpu

type cpuUsage struct {
	*docker.Stat

	cpus        uint32
	systemDelta uint64
}

// CPUS returns the number of cpus. If number of cpus is equal to zero, the field will
// be updated/initialized with the corresponding value retrieved from Docker API.
func (u *cpuUsage) CPUs() uint32 {
	if u.cpus == 0 {
		u.cpus = u.Stats.CPUStats.OnlineCPUs
	}
	return u.cpus
}

// SystemDelta calculates system delta.
func (u *cpuUsage) SystemDelta() uint64 {
	if u.systemDelta == 0 {
		u.systemDelta = u.Stats.CPUStats.SystemUsage - u.Stats.PreCPUStats.SystemUsage
	}
	return u.systemDelta
}

// PerCPU calculates per CPU usage.
func (u *cpuUsage) PerCPU() common.MapStr {
	var output common.MapStr
	if len(u.Stats.CPUStats.CPUUsage.PercpuUsage) == len(u.Stats.PreCPUStats.CPUUsage.PercpuUsage) {
		output = common.MapStr{}
		for index := range u.Stats.CPUStats.CPUUsage.PercpuUsage {
			cpu := common.MapStr{}
			cpu["pct"] = u.calculatePercentage(
				u.Stats.CPUStats.CPUUsage.PercpuUsage[index],
				u.Stats.PreCPUStats.CPUUsage.PercpuUsage[index],
				u.CPUs())
			cpu["norm"] = common.MapStr{
				"pct": u.calculatePercentage(
					u.Stats.CPUStats.CPUUsage.PercpuUsage[index],
					u.Stats.PreCPUStats.CPUUsage.PercpuUsage[index],
					1),
			}
			cpu["ticks"] = u.Stats.CPUStats.CPUUsage.PercpuUsage[index]
			output[strconv.Itoa(index)] = cpu
		}
	}
	return output
}

// TotalNormalized calculates total CPU usage normalized.
func (u *cpuUsage) Total() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.CPUUsage.TotalUsage, u.Stats.PreCPUStats.CPUUsage.TotalUsage, u.CPUs())
}

// TotalNormalized calculates total CPU usage normalized by the number of CPU cores.
func (u *cpuUsage) TotalNormalized() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.CPUUsage.TotalUsage, u.Stats.PreCPUStats.CPUUsage.TotalUsage, 1)
}

// InKernelMode calculates percentage of time in kernel space.
func (u *cpuUsage) InKernelMode() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.CPUUsage.UsageInKernelmode, u.Stats.PreCPUStats.CPUUsage.UsageInKernelmode, u.CPUs())
}

// InKernelModeNormalized calculates percentage of time in kernel space normalized by the number of CPU cores.
func (u *cpuUsage) InKernelModeNormalized() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.CPUUsage.UsageInKernelmode, u.Stats.PreCPUStats.CPUUsage.UsageInKernelmode, 1)
}

// InUserMode calculates percentage of time in user space.
func (u *cpuUsage) InUserMode() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.CPUUsage.UsageInUsermode, u.Stats.PreCPUStats.CPUUsage.UsageInUsermode, u.CPUs())
}

// InUserModeNormalized calculates percentage of time in user space normalized by the number of CPU cores.
func (u *cpuUsage) InUserModeNormalized() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.CPUUsage.UsageInUsermode, u.Stats.PreCPUStats.CPUUsage.UsageInUsermode, 1)
}

// System calculates percentage of total CPU time in the system.
func (u *cpuUsage) System() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.SystemUsage, u.Stats.PreCPUStats.SystemUsage, u.CPUs())
}

// SystemNormalized calculates percentage of total CPU time in the system, normalized by the number of CPU cores.
func (u *cpuUsage) SystemNormalized() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.SystemUsage, u.Stats.PreCPUStats.SystemUsage, 1)
}

// This function is meant to calculate the % CPU time change between two successive readings.
// The "oldValue" refers to the CPU statistics of the last read.
// Time here is expressed by second and not by nanoseconde.
// The main goal is to expose the %, in the same way, it's displayed by docker Client.
func (u *cpuUsage) calculatePercentage(newValue uint64, oldValue uint64, numCPUS uint32) float64 {
	if newValue < oldValue {
		logp.Err("Error calculating CPU time change for docker module: new stats value (%v) is lower than the old one(%v)", newValue, oldValue)
		return -1
	}
	value := newValue - oldValue
	if value == 0 || u.SystemDelta() == 0 {
		return 0
	}

	return float64(uint64(numCPUS)*value) / float64(u.SystemDelta())
}
