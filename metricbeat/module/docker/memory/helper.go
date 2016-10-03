package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
)

type MemoryData struct {
	Time       common.Time
	Container  *docker.Container
	Failcnt    uint64
	Limit      uint64
	MaxUsage   uint64
	TotalRss   uint64
	TotalRss_p float64
	Usage      uint64
	Usage_p    float64
}

type MemoryService struct{}

func (c *MemoryService) getMemoryStatsList(rawStats []docker.DockerStat) []MemoryData {
	formatedStats := []MemoryData{}
	for _, myRawStats := range rawStats {
		formatedStats = append(formatedStats, c.GetMemoryStats(myRawStats))
	}

	return formatedStats
}

func (ms *MemoryService) GetMemoryStats(myRawStat docker.DockerStat) MemoryData {

	return MemoryData{
		Time:       common.Time(myRawStat.Stats.Read),
		Container:  docker.NewContainer(&myRawStat.Container),
		Failcnt:    myRawStat.Stats.MemoryStats.Failcnt,
		Limit:      myRawStat.Stats.MemoryStats.Limit,
		MaxUsage:   myRawStat.Stats.MemoryStats.MaxUsage,
		TotalRss:   myRawStat.Stats.MemoryStats.Stats.TotalRss,
		TotalRss_p: float64(myRawStat.Stats.MemoryStats.Stats.TotalRss) / float64(myRawStat.Stats.MemoryStats.Limit),
		Usage:      myRawStat.Stats.MemoryStats.Usage,
		Usage_p:    float64(myRawStat.Stats.MemoryStats.Usage) / float64(myRawStat.Stats.MemoryStats.Limit),
	}
}
