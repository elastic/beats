package cpu

import (
	"strconv"

	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
)

type CPURaw struct {
	PerCpuUsage       []uint64
	TotalUsage        uint64
	UsageInKernelmode uint64
	UsageInUsermode   uint64
}

type CPUCalculator interface {
	perCpuUsage(stats *dc.Stats) common.MapStr
	totalUsage(stats *dc.Stats) float64
	usageInKernelmode(stats *dc.Stats) float64
	usageInUsermode(stats *dc.Stats) float64
}

type CPUStats struct {
	Time              common.Time
	Container         *docker.Container
	PerCpuUsage       common.MapStr
	TotalUsage        float64
	UsageInKernelmode float64
	UsageInUsermode   float64
}

type CPUService struct{}

func NewCpuService() *CPUService {
	return &CPUService{}
}

func (c *CPUService) getCPUStatsList(rawStats []docker.DockerStat) []CPUStats {
	formatedStats := []CPUStats{}

	for _, stats := range rawStats {
		formatedStats = append(formatedStats, c.getCpuStats(&stats))
	}

	return formatedStats
}

func (c *CPUService) getCpuStats(myRawStat *docker.DockerStat) CPUStats {

	return CPUStats{
		Time:              common.Time(myRawStat.Stats.Read),
		Container:         docker.NewContainer(&myRawStat.Container),
		PerCpuUsage:       c.PerCpuUsage(&myRawStat.Stats),
		TotalUsage:        c.TotalUsage(&myRawStat.Stats),
		UsageInKernelmode: c.UsageInKernelmode(&myRawStat.Stats),
		UsageInUsermode:   c.UsageInUsermode(&myRawStat.Stats),
	}
}

func getOldCpu(stats *dc.Stats) CPURaw {
	return CPURaw{
		PerCpuUsage:       stats.PreCPUStats.CPUUsage.percpuUsage,
		TotalUsage:        stats.PreCPUStats.CPUUsage.totalUsage,
		UsageInKernelmode: stats.PreCPUStats.CPUUsage.usageInKernelmode,
		UsageInUsermode:   stats.PreCPUStats.CPUUsage.usageInUsermode,
	}
}

func getNewCpu(stats *dc.Stats) CPURaw {
	return CPURaw{
		PerCpuUsage:       stats.CPUStats.CPUUsage.percpuUsage,
		TotalUsage:        stats.CPUStats.CPUUsage.totalUsage,
		UsageInKernelmode: stats.CPUStats.CPUUsage.usageInKernelmode,
		UsageInUsermode:   stats.CPUStats.CPUUsage.usageInUsermode,
	}
}

func (c *CPUService) perCpuUsage(stats *dc.Stats) common.MapStr {
	var output common.MapStr
	if cap(getNewCpu(stats).PerCpuUsage) == cap(getOldCpu(stats).PerCpuUsage) {
		output = common.MapStr{}
		for index := range getNewCpu(stats).PerCpuUsage {
			output[strconv.Itoa(index)] = c.calculateLoad(int64(getNewCpu(stats).PerCpuUsage[index] - getOldCpu(stats).PerCpuUsage[index]))
		}
	}
	return output
}

func (c *CPUService) totalUsage(stats *dc.Stats) float64 {
	return c.calculateLoad(int64(getNewCpu(stats).TotalUsage - getOldCpu(stats).TotalUsage))
}

func (c *CPUService) usageInKernelmode(stats *dc.Stats) float64 {
	return c.calculateLoad(int64(getNewCpu(stats).UsageInKernelmode - getOldCpu(stats).UsageInKernelmode))
}

func (c *CPUService) usageInUsermode(stats *dc.Stats) float64 {
	return c.calculateLoad(int64(getNewCpu(stats).UsageInUsermode - getOldCpu(stats).UsageInUsermode))
}

func (c *CPUService) calculateLoad(value int64) float64 {
	if value < 0 {
		value = 0
	}
	return float64(value) / float64(1000000000)
}
