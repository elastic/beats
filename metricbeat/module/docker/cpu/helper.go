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
	PerCpuUsage(stats *dc.Stats) common.MapStr
	TotalUsage(stats *dc.Stats) float64
	UsageInKernelmode(stats *dc.Stats) float64
	UsageInUsermode(stats *dc.Stats) float64
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

func getOLdCpu(stats *dc.Stats) CPURaw {
	return CPURaw{
		PerCpuUsage:       stats.PreCPUStats.CPUUsage.PercpuUsage,
		TotalUsage:        stats.PreCPUStats.CPUUsage.TotalUsage,
		UsageInKernelmode: stats.PreCPUStats.CPUUsage.UsageInKernelmode,
		UsageInUsermode:   stats.PreCPUStats.CPUUsage.UsageInUsermode,
	}
}

func getNewCpu(stats *dc.Stats) CPURaw {
	return CPURaw{
		PerCpuUsage:       stats.CPUStats.CPUUsage.PercpuUsage,
		TotalUsage:        stats.CPUStats.CPUUsage.TotalUsage,
		UsageInKernelmode: stats.CPUStats.CPUUsage.UsageInKernelmode,
		UsageInUsermode:   stats.CPUStats.CPUUsage.UsageInUsermode,
	}
}

func (c *CPUService) PerCpuUsage(stats *dc.Stats) common.MapStr {
	var output common.MapStr
	if cap(getNewCpu(stats).PerCpuUsage) == cap(getOLdCpu(stats).PerCpuUsage) {
		output = common.MapStr{}
		for index := range getNewCpu(stats).PerCpuUsage {
			output[strconv.Itoa(index)] = c.calculateLoad(int64(getNewCpu(stats).PerCpuUsage[index] - getOLdCpu(stats).PerCpuUsage[index]))
		}
	}
	return output
}

func (c *CPUService) TotalUsage(stats *dc.Stats) float64 {
	return c.calculateLoad(int64(getNewCpu(stats).TotalUsage - getOLdCpu(stats).TotalUsage))
}

func (c *CPUService) UsageInKernelmode(stats *dc.Stats) float64 {
	return c.calculateLoad(int64(getNewCpu(stats).UsageInKernelmode - getOLdCpu(stats).UsageInKernelmode))
}

func (c *CPUService) UsageInUsermode(stats *dc.Stats) float64 {
	return c.calculateLoad(int64(getNewCpu(stats).UsageInUsermode - getOLdCpu(stats).UsageInUsermode))
}

func (c *CPUService) calculateLoad(value int64) float64 {
	return float64(value) / float64(1000000000)
}
