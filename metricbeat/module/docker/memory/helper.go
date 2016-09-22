package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/module/docker"
)

type MemoryData struct {
	Time        common.Time
	MyContainer *docker.Container
	Failcnt     uint64
	Limit       uint64
	MaxUsage    uint64
	TotalRss    uint64
	TotalRss_p  float64
	Usage       uint64
	Usage_p     float64
}
type MemoryService struct{}

func (c *MemoryService) GetMemoryStatsList(rawStats []docker.DockerStat) []MemoryData {
	formatedStats := []MemoryData{}
	if len(rawStats) != 0 {
		for _, myRawStats := range rawStats {
			formatedStats = append(formatedStats, c.GetMemoryStats(myRawStats))
		}
	} else {
		logp.Info("No container is running")
	}
	return formatedStats
}
func (ms *MemoryService) GetMemoryStats(myRawStat docker.DockerStat) MemoryData {

	return MemoryData{
		Time:        common.Time(myRawStat.Stats.Read),
		MyContainer: docker.InitCurrentContainer(&myRawStat.Container),
		Failcnt:     myRawStat.Stats.MemoryStats.Failcnt,
		Limit:       myRawStat.Stats.MemoryStats.Limit,
		MaxUsage:    myRawStat.Stats.MemoryStats.MaxUsage,
		TotalRss:    myRawStat.Stats.MemoryStats.Stats.TotalRss,
		TotalRss_p:  float64(myRawStat.Stats.MemoryStats.Stats.TotalRss) / float64(myRawStat.Stats.MemoryStats.Limit),
		Usage:       myRawStat.Stats.MemoryStats.Usage,
		Usage_p:     float64(myRawStat.Stats.MemoryStats.Usage) / float64(myRawStat.Stats.MemoryStats.Limit),
	}
}
