package memory

import (
	"github.com/elastic/beats/metricbeat/module/docker"
	"github.com/elastic/beats/libbeat/common"
	"fmt"
)

type MEMORYData struct {
	Time common.Time
	MyContainer *docker.Container
	Failcnt	uint64
	Limit	uint64
	MaxUsage uint64
	TotalRss uint64
	TotalRss_p float64
	Usage 	uint64
	Usage_p	float64

}
type MEMORYService struct {

}
func NewMemoryService() *MEMORYService{
	return &MEMORYService{}
}

func (c *MEMORYService) GetMemorystatsList(rawStats []docker.DockerStat)[]MEMORYData{
	formatedStats :=[]MEMORYData{}
	if len(rawStats) !=0 {
		for _, myRawStats := range rawStats {
			formatedStats = append(formatedStats, c.getMEMData(myRawStats))
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

func (ms *MEMORYService) getMEMData(myRawStat docker.DockerStat) MEMORYData {

	return MEMORYData{
		Time: common.Time(myRawStat.Stats.Read),
		MyContainer: docker.InitCurrentContainer(&myRawStat.Container),
		Failcnt: myRawStat.Stats.MemoryStats.Failcnt,
		Limit:      myRawStat.Stats.MemoryStats.Limit,
		MaxUsage:   myRawStat.Stats.MemoryStats.MaxUsage,
		TotalRss:   myRawStat.Stats.MemoryStats.Stats.TotalRss,
		TotalRss_p: float64(myRawStat.Stats.MemoryStats.Stats.TotalRss) / float64(myRawStat.Stats.MemoryStats.Limit),
		Usage:      myRawStat.Stats.MemoryStats.Usage,
		Usage_p:    float64(myRawStat.Stats.MemoryStats.Usage) / float64(myRawStat.Stats.MemoryStats.Limit),
	}
}
