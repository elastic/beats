package cpu

import (
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"

	dc "github.com/fsouza/go-dockerclient"
)

type CPUCalculator interface {
	perCpuUsage(stats *dc.Stats) common.MapStr
	totalUsage(stats *dc.Stats) float64
	usageInKernelmode(stats *dc.Stats) float64
	usageInUsermode(stats *dc.Stats) float64
}

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

func (c *CPUService) getCPUStatsList(rawStats []docker.Stat) []CPUStats {
	formattedStats := []CPUStats{}

	for _, stats := range rawStats {
		formattedStats = append(formattedStats, c.getCpuStats(&stats))
	}

	return formattedStats
}

func (c *CPUService) getCpuStats(myRawStat *docker.Stat) CPUStats {

	return CPUStats{
		Time:                        common.Time(myRawStat.Stats.Read),
		Container:                   docker.NewContainer(&myRawStat.Container),
		PerCpuUsage:                 perCpuUsage(&myRawStat.Stats),
		TotalUsage:                  totalUsage(&myRawStat.Stats),
		UsageInKernelmode:           myRawStat.Stats.CPUStats.CPUUsage.UsageInKernelmode,
		UsageInKernelmodePercentage: usageInKernelmode(&myRawStat.Stats),
		UsageInUsermode:             myRawStat.Stats.CPUStats.CPUUsage.UsageInUsermode,
		UsageInUsermodePercentage:   usageInUsermode(&myRawStat.Stats),
		SystemUsage:                 myRawStat.Stats.CPUStats.SystemCPUUsage,
		SystemUsagePercentage:       systemUsage(&myRawStat.Stats),
	}
}

func perCpuUsage(stats *dc.Stats) common.MapStr {
	var output common.MapStr
	if len(stats.CPUStats.CPUUsage.PercpuUsage) == len(stats.PreCPUStats.CPUUsage.PercpuUsage) {
		output = common.MapStr{}
		for index := range stats.CPUStats.CPUUsage.PercpuUsage {
			cpu := common.MapStr{}
			cpu["pct"] = calculateLoad(stats.CPUStats.CPUUsage.PercpuUsage[index] - stats.PreCPUStats.CPUUsage.PercpuUsage[index])
			cpu["ticks"] = stats.CPUStats.CPUUsage.PercpuUsage[index]
			output[strconv.Itoa(index)] = cpu
		}
	}
	return output
}

// TODO: These helper should be merged with the cpu helper in system/cpu

func totalUsage(stats *dc.Stats) float64 {
	return calculateLoad(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
}

func usageInKernelmode(stats *dc.Stats) float64 {
	return calculateLoad(stats.CPUStats.CPUUsage.UsageInKernelmode - stats.PreCPUStats.CPUUsage.UsageInKernelmode)
}

func usageInUsermode(stats *dc.Stats) float64 {
	return calculateLoad(stats.CPUStats.CPUUsage.UsageInUsermode - stats.PreCPUStats.CPUUsage.UsageInUsermode)
}

func systemUsage(stats *dc.Stats) float64 {
	return calculateLoad(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)
}

func calculateLoad(value uint64) float64 {
	return float64(value) / float64(1000000000)
}
