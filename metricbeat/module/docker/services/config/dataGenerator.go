package config

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/fsouza/go-dockerclient"
	"github.com/elastic/beats/metricbeat/module/docker/calculator"
	"strings"
	"time"
)
type DataGenerator struct {
	Socket            *string
	CalculatorFactory calculator.CalculatorFactory
	Period            time.Duration
}
type currentContainer struct{
	Time common.Time
	Id string
	Name string
	Labels []common.MapStr
	Socket *string
}
type CPUData struct {
	MyContainer 	*currentContainer
	PerCpuUsage       common.MapStr
	TotalUsage        float64
	UsageInKernelmode float64
	UsageInUsermode   float64

}
type MEMORYData struct {
	MyContainer *currentContainer
	Failcnt	uint64
	Limit	uint64
	MaxUsage uint64
	TotalRss uint64
	TotalRss_p float64
	Usage 	uint64
	Usage_p	float64

}

func (d *DataGenerator) GetCpuData(container *docker.APIContainers, stats *docker.Stats)  CPUData {

	calculator := d.CalculatorFactory.NewCPUCalculator(
		calculator.CPUData{
			PerCpuUsage:       stats.PreCPUStats.CPUUsage.PercpuUsage,
			TotalUsage:        stats.PreCPUStats.CPUUsage.TotalUsage,
			UsageInKernelmode: stats.PreCPUStats.CPUUsage.UsageInKernelmode,
			UsageInUsermode:   stats.PreCPUStats.CPUUsage.UsageInUsermode,
		},
		calculator.CPUData{
			PerCpuUsage:       stats.CPUStats.CPUUsage.PercpuUsage,
			TotalUsage:        stats.CPUStats.CPUUsage.TotalUsage,
			UsageInKernelmode: stats.CPUStats.CPUUsage.UsageInKernelmode,
			UsageInUsermode:   stats.CPUStats.CPUUsage.UsageInUsermode,
		},
	)
	myData := CPUData{
		MyContainer: d.initCurrentContainer(container, common.Time(stats.Read)),
		PerCpuUsage: calculator.PerCpuUsage(),
		TotalUsage: calculator.TotalUsage(),
		UsageInKernelmode: calculator.UsageInKernelmode(),
		UsageInUsermode: calculator.UsageInUsermode(),
	}

	return myData
}
func (d *DataGenerator) GetMemoryData(container *docker.APIContainers, stats *docker.Stats)  MEMORYData {

	myData := MEMORYData{
		MyContainer: d.initCurrentContainer(container, common.Time(stats.Read)),
		Failcnt: stats.MemoryStats.Failcnt,
		Limit:      stats.MemoryStats.Limit,
		MaxUsage:   stats.MemoryStats.MaxUsage,
		TotalRss:   stats.MemoryStats.Stats.TotalRss,
		TotalRss_p: float64(stats.MemoryStats.Stats.TotalRss) / float64(stats.MemoryStats.Limit),
		Usage:      stats.MemoryStats.Usage,
		Usage_p:    float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit),
	}

	return myData
}
func (d *DataGenerator) initCurrentContainer(container *docker.APIContainers, time common.Time) *currentContainer{
	return &currentContainer{
		Time: time,
		Id: container.ID,
		Name: d.extractContainerName(container.Names),
		Labels: d.buildLabelArray(container.Labels),
		Socket: d.Socket,
	}
}
func (d *DataGenerator) extractContainerName(names []string) string {
	output := names[0]

	if cap(names) > 1 {
		for _, name := range names {
			if strings.Count(output, "/") > strings.Count(name, "/") {
				output = name
			}
		}
	}
	return strings.Trim(output, "/")
}
func (d *DataGenerator) buildLabelArray(labels map[string]string) []common.MapStr {

	output_labels := make([]common.MapStr, len(labels))

	i := 0
	for k, v := range labels {
		label := strings.Replace(k, ".", "_", -1)
		output_labels[i] = common.MapStr{
			"key":   label,
			"value": v,
		}
		i++
	}
	return output_labels
}
