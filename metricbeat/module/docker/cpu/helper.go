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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
)

type CPUStats struct {
	Time                                  common.Time
	Container                             *docker.Container
	PerCPUUsage                           mapstr.M
	TotalUsage                            opt.Float
	TotalUsageNormalized                  opt.Float
	UsageInKernelmode                     uint64
	UsageInKernelmodePercentage           opt.Float
	UsageInKernelmodePercentageNormalized opt.Float
	UsageInUsermode                       uint64
	UsageInUsermodePercentage             opt.Float
	UsageInUsermodePercentageNormalized   opt.Float
	SystemUsage                           uint64
	SystemUsagePercentage                 opt.Float
	SystemUsagePercentageNormalized       opt.Float
}

// CPUService is a helper to collect docker CPU metrics
type CPUService struct {
	Cores bool
}

func (c *CPUService) getCPUStatsList(rawStats []docker.Stat, dedot bool) []CPUStats {
	formattedStats := []CPUStats{}

	for _, stats := range rawStats {
		if stats.Stats.Read.IsZero() {
			continue
		}
		formattedStats = append(formattedStats, c.getCPUStats(&stats, dedot))
	}

	return formattedStats
}

func (c *CPUService) getCPUStats(myRawStat *docker.Stat, dedot bool) CPUStats {
	usage := CPUUsage{Stat: myRawStat}
	total, totalNormalized := usage.total()
	kernel, kernelNormalized := usage.inKernelMode()
	user, userNormalized := usage.inUserMode()
	system, systemNormalized := usage.system()

	stats := CPUStats{
		Time:                                  common.Time(myRawStat.Stats.Read),
		Container:                             docker.NewContainer(myRawStat.Container, dedot),
		TotalUsage:                            total,
		TotalUsageNormalized:                  totalNormalized,
		UsageInKernelmode:                     myRawStat.Stats.CPUStats.CPUUsage.UsageInKernelmode,
		UsageInKernelmodePercentage:           kernel,
		UsageInKernelmodePercentageNormalized: kernelNormalized,
		UsageInUsermode:                       myRawStat.Stats.CPUStats.CPUUsage.UsageInUsermode,
		UsageInUsermodePercentage:             user,
		UsageInUsermodePercentageNormalized:   userNormalized,
		SystemUsage:                           myRawStat.Stats.CPUStats.SystemUsage,
		SystemUsagePercentage:                 system,
		SystemUsagePercentageNormalized:       systemNormalized,
	}

	if c.Cores {
		stats.PerCPUUsage = usage.PerCPU()
	}

	return stats
}

// TODO: These helper should be merged with the cpu helper in system/cpu

type CPUUsage struct {
	*docker.Stat

	cpus                  uint32
	systemDelta           uint64
	systemDeltaCalculated bool
	systemDeltaValid      bool
}

// CPUS returns the number of cpus. If number of cpus is equal to zero, the field will
// be updated/initialized with the corresponding value retrieved from Docker API.
func (u *CPUUsage) CPUs() uint32 {
	if u.cpus == 0 {
		if u.Stats.CPUStats.OnlineCPUs != 0 {
			u.cpus = u.Stats.CPUStats.OnlineCPUs
		} else {
			//Certain versions of docker don't have `online_cpus`
			//In addition to this, certain kernel versions will report spurious zeros from the cgroups usage_percpu
			var realCPUCount uint32
			for _, rCPUUsage := range u.Stats.CPUStats.CPUUsage.PercpuUsage {
				if rCPUUsage != 0 {
					realCPUCount++
				}
			}
			u.cpus = realCPUCount
		}

	}
	return u.cpus
}

// SystemDelta calculates system delta.
func (u *CPUUsage) SystemDelta() uint64 {
	systemDelta, _ := u.validSystemDelta()
	return systemDelta
}

func (u *CPUUsage) validSystemDelta() (uint64, bool) {
	if !u.systemDeltaCalculated {
		u.systemDeltaCalculated = true
		if u.Stats.CPUStats.SystemUsage >= u.Stats.PreCPUStats.SystemUsage {
			u.systemDelta = u.Stats.CPUStats.SystemUsage - u.Stats.PreCPUStats.SystemUsage
			u.systemDeltaValid = true
		}
	}
	return u.systemDelta, u.systemDeltaValid
}

// PerCPU calculates per CPU usage.
func (u *CPUUsage) PerCPU() mapstr.M {
	var output mapstr.M
	if len(u.Stats.CPUStats.CPUUsage.PercpuUsage) == len(u.Stats.PreCPUStats.CPUUsage.PercpuUsage) {
		output = mapstr.M{}
		for index := range u.Stats.CPUStats.CPUUsage.PercpuUsage {
			percentage, normalized := u.calculatePercentages(
				u.Stats.CPUStats.CPUUsage.PercpuUsage[index],
				u.Stats.PreCPUStats.CPUUsage.PercpuUsage[index],
			)
			cpu := mapstr.M{"ticks": u.Stats.CPUStats.CPUUsage.PercpuUsage[index]}
			addPercentageFields(cpu, percentage, normalized)
			output[strconv.Itoa(index)] = cpu
		}
	}
	return output
}

func (u *CPUUsage) total() (opt.Float, opt.Float) {
	return u.calculatePercentages(u.Stats.CPUStats.CPUUsage.TotalUsage, u.Stats.PreCPUStats.CPUUsage.TotalUsage)
}

// Total calculates total CPU usage.
func (u *CPUUsage) Total() opt.Float {
	percentage, _ := u.total()
	return percentage
}

// TotalNormalized calculates total CPU usage normalized by the number of CPU cores.
func (u *CPUUsage) TotalNormalized() opt.Float {
	_, normalized := u.total()
	return normalized
}

func (u *CPUUsage) inKernelMode() (opt.Float, opt.Float) {
	return u.calculatePercentages(u.Stats.CPUStats.CPUUsage.UsageInKernelmode, u.Stats.PreCPUStats.CPUUsage.UsageInKernelmode)
}

// InKernelMode calculates percentage of time in kernel space.
func (u *CPUUsage) InKernelMode() opt.Float {
	percentage, _ := u.inKernelMode()
	return percentage
}

// InKernelModeNormalized calculates percentage of time in kernel space normalized by the number of CPU cores.
func (u *CPUUsage) InKernelModeNormalized() opt.Float {
	_, normalized := u.inKernelMode()
	return normalized
}

func (u *CPUUsage) inUserMode() (opt.Float, opt.Float) {
	return u.calculatePercentages(u.Stats.CPUStats.CPUUsage.UsageInUsermode, u.Stats.PreCPUStats.CPUUsage.UsageInUsermode)
}

// InUserMode calculates percentage of time in user space.
func (u *CPUUsage) InUserMode() opt.Float {
	percentage, _ := u.inUserMode()
	return percentage
}

// InUserModeNormalized calculates percentage of time in user space normalized by the number of CPU cores.
func (u *CPUUsage) InUserModeNormalized() opt.Float {
	_, normalized := u.inUserMode()
	return normalized
}

func (u *CPUUsage) system() (opt.Float, opt.Float) {
	return u.calculatePercentages(u.Stats.CPUStats.SystemUsage, u.Stats.PreCPUStats.SystemUsage)
}

// System calculates percentage of total CPU time in the system.
func (u *CPUUsage) System() opt.Float {
	percentage, _ := u.system()
	return percentage
}

// SystemNormalized calculates percentage of total CPU time in the system, normalized by the number of CPU cores.
func (u *CPUUsage) SystemNormalized() opt.Float {
	_, normalized := u.system()
	return normalized
}

// calculatePercentages calculates CPU usage between two successive readings.
// It returns the percentage scaled by the number of CPUs and the normalized
// per-CPU percentage, respectively.
func (u *CPUUsage) calculatePercentages(newValue uint64, oldValue uint64) (opt.Float, opt.Float) {
	if newValue < oldValue {
		return opt.NewFloatNone(), opt.NewFloatNone()
	}

	value := newValue - oldValue
	if value == 0 {
		zero := opt.FloatWith(0)
		return zero, zero
	}

	systemDelta, valid := u.validSystemDelta()
	if !valid {
		return opt.NewFloatNone(), opt.NewFloatNone()
	}

	if systemDelta == 0 {
		zero := opt.FloatWith(0)
		return zero, zero
	}

	normalized := opt.FloatWith(float64(value) / float64(systemDelta))
	percentage := opt.FloatWith(float64(uint64(u.CPUs())*value) / float64(systemDelta))
	return percentage, normalized
}
