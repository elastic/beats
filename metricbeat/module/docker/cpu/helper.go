package cpu

import (
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/module/docker"
)

type CPUStats struct {
	Time                        common.Time
	Container                   *docker.Container
	PerCpuUsage                 common.MapStr
	TotalUsage                  float64
	UsageInKernelmode           uint64
	UsageInKernelmodePercentage float64
	UsageInUsermode             uint64
	UsageInUsermodePercentage   float64
	SystemUsage                 uint64
	SystemUsagePercentage       float64
}

type CPUService struct{}

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

	return CPUStats{
		Time:                        common.Time(myRawStat.Stats.Read),
		Container:                   docker.NewContainer(myRawStat.Container, dedot),
		PerCpuUsage:                 usage.PerCPU(),
		TotalUsage:                  usage.Total(),
		UsageInKernelmode:           myRawStat.Stats.CPUStats.CPUUsage.UsageInKernelmode,
		UsageInKernelmodePercentage: usage.InKernelMode(),
		UsageInUsermode:             myRawStat.Stats.CPUStats.CPUUsage.UsageInUsermode,
		UsageInUsermodePercentage:   usage.InUserMode(),
		SystemUsage:                 myRawStat.Stats.CPUStats.SystemUsage,
		SystemUsagePercentage:       usage.System(),
	}
}

// TODO: These helper should be merged with the cpu helper in system/cpu

type cpuUsage struct {
	*docker.Stat

	cpus        int
	systemDelta uint64
}

func (u *cpuUsage) CPUs() int {
	if u.cpus == 0 {
		u.cpus = len(u.Stats.CPUStats.CPUUsage.PercpuUsage)
	}
	return u.cpus
}

func (u *cpuUsage) SystemDelta() uint64 {
	if u.systemDelta == 0 {
		u.systemDelta = u.Stats.CPUStats.SystemUsage - u.Stats.PreCPUStats.SystemUsage
	}
	return u.systemDelta
}

func (u *cpuUsage) PerCPU() common.MapStr {
	var output common.MapStr
	if len(u.Stats.CPUStats.CPUUsage.PercpuUsage) == len(u.Stats.PreCPUStats.CPUUsage.PercpuUsage) {
		output = common.MapStr{}
		for index := range u.Stats.CPUStats.CPUUsage.PercpuUsage {
			cpu := common.MapStr{}
			cpu["pct"] = u.calculatePercentage(
				u.Stats.CPUStats.CPUUsage.PercpuUsage[index],
				u.Stats.PreCPUStats.CPUUsage.PercpuUsage[index])
			cpu["ticks"] = u.Stats.CPUStats.CPUUsage.PercpuUsage[index]
			output[strconv.Itoa(index)] = cpu
		}
	}
	return output
}

func (u *cpuUsage) Total() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.CPUUsage.TotalUsage, u.Stats.PreCPUStats.CPUUsage.TotalUsage)
}

func (u *cpuUsage) InKernelMode() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.CPUUsage.UsageInKernelmode, u.Stats.PreCPUStats.CPUUsage.UsageInKernelmode)
}

func (u *cpuUsage) InUserMode() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.CPUUsage.UsageInUsermode, u.Stats.PreCPUStats.CPUUsage.UsageInUsermode)
}

func (u *cpuUsage) System() float64 {
	return u.calculatePercentage(u.Stats.CPUStats.SystemUsage, u.Stats.PreCPUStats.SystemUsage)
}

// This function is meant to calculate the % CPU time change between two successive readings.
// The "oldValue" refers to the CPU statistics of the last read.
// Time here is expressed by second and not by nanoseconde.
// The main goal is to expose the %, in the same way, it's displayed by docker Client.
func (u *cpuUsage) calculatePercentage(newValue uint64, oldValue uint64) float64 {
	if newValue < oldValue {
		logp.Err("Error calculating CPU time change for docker module: new stats value (%v) is lower than the old one(%v)", newValue, oldValue)
		return -1
	}
	value := newValue - oldValue
	if value == 0 || u.SystemDelta() == 0 {
		return 0
	}

	return float64(uint64(u.CPUs())*value) / float64(u.SystemDelta())
}
