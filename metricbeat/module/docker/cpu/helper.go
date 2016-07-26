package cpu

import (
	"strconv"
	dc"github.com/fsouza/go-dockerclient"
	"github.com/elastic/beats/metricbeat/module/docker"
	"github.com/elastic/beats/libbeat/common"
	"fmt"
)


type CPURaw struct {
	PerCpuUsage       []uint64
	TotalUsage        uint64
	UsageInKernelmode uint64
	UsageInUsermode   uint64
}
type CPUCalculator interface {
	PerCpuUsage() common.MapStr
	TotalUsage() float64
	UsageInKernelmode() float64
	UsageInUsermode() float64
}
type CPUStats struct {
	Time common.Time
	MyContainer 	*docker.Container
	PerCpuUsage       common.MapStr
	TotalUsage        float64
	UsageInKernelmode float64
	UsageInUsermode   float64

}
type CPUService struct{}


func (c CPUService) GetCPUstatsList(rawStats []docker.DockerStat)[]CPUStats{
	 formatedStats :=[]CPUStats{}
	if len(rawStats) !=0 {
		for _, myRawStats := range rawStats {
			formatedStats = append(formatedStats, c.getCpuStats(myRawStats))
		}
	}else{
		fmt.Printf("No container is running \n")
	}
	/*fmt.Printf("From helper/getCPUStatsList \n")
	for _, event := range myEvents{
		fmt.Printf(" container's name ", event.MyContainer.Name,"\n")
	}*/
	return formatedStats
}
func (c CPUService) getCpuStats(myRawStat docker.DockerStat)  CPUStats {

	return CPUStats{
		Time: common.Time(myRawStat.Stats.Read),
		MyContainer: docker.InitCurrentContainer(&myRawStat.Container),
		PerCpuUsage:c.perCpuUsage(&myRawStat.Stats),
		TotalUsage: c.totalUsage(&myRawStat.Stats),
		UsageInKernelmode: c.usageInKernelmode(&myRawStat.Stats),
		UsageInUsermode: c.usageInUsermode(&myRawStat.Stats),
	}
}

func NewCpuService() *CPUService{
	return &CPUService{}
}
func getOLdCpu(stats *dc.Stats ) CPURaw{
	return CPURaw{
		PerCpuUsage: stats.PreCPUStats.CPUUsage.PercpuUsage,
		TotalUsage: stats.PreCPUStats.CPUUsage.TotalUsage,
		UsageInKernelmode: stats.PreCPUStats.CPUUsage.UsageInKernelmode,
		UsageInUsermode: stats.PreCPUStats.CPUUsage.UsageInUsermode,
	}
}
func getNewCpu( stats *dc.Stats) CPURaw{
	return CPURaw{
		PerCpuUsage: stats.CPUStats.CPUUsage.PercpuUsage,
		TotalUsage: stats.CPUStats.CPUUsage.TotalUsage,
		UsageInKernelmode: stats.CPUStats.CPUUsage.UsageInKernelmode,
		UsageInUsermode: stats.CPUStats.CPUUsage.UsageInUsermode,
	}
}

func (c CPUService) perCpuUsage(stats *dc.Stats) common.MapStr {
	var output common.MapStr
	if cap(getNewCpu(stats).PerCpuUsage) == cap(getOLdCpu(stats).PerCpuUsage) {
		output = common.MapStr{}
		for index := range getNewCpu(stats).PerCpuUsage {
			output["cpu"+strconv.Itoa(index)] = c.calculateLoad(getNewCpu(stats).PerCpuUsage[index] - getOLdCpu(stats).PerCpuUsage[index])
		}
	}
	return output
}
func (c CPUService) totalUsage(stats *dc.Stats) float64 {
	return c.calculateLoad(getNewCpu(stats).TotalUsage - getOLdCpu(stats).TotalUsage)
}
func (c CPUService) usageInKernelmode(stats *dc.Stats) float64 {
	return c.calculateLoad(getNewCpu(stats).UsageInKernelmode - getOLdCpu(stats).UsageInKernelmode)
}
func (c CPUService) usageInUsermode(stats *dc.Stats) float64 {
	return c.calculateLoad(getNewCpu(stats).UsageInUsermode - getOLdCpu(stats).UsageInUsermode)
}
func (c CPUService) calculateLoad(value uint64) float64 {
	// value is the count of CPU nanosecond in 1sec
	// TODO save the old stat timestamp and reuse here in case of docker read time changes...
	// 1s = 1000000000 ns
	// value / 1000000000
	return float64(value) / float64(1000000000)
}
