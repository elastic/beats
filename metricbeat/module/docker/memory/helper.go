package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
)

type MemoryData struct {
	Time      common.Time
	Container *docker.Container
	Failcnt   uint64
	Limit     uint64
	MaxUsage  uint64
	TotalRss  uint64
	TotalRssP float64
	Usage     uint64
	UsageP    float64
}

type MemoryService struct{}

func (s *MemoryService) getMemoryStatsList(rawStats []docker.Stat, dedot bool) []MemoryData {
	formattedStats := []MemoryData{}
	for _, myRawStats := range rawStats {
		formattedStats = append(formattedStats, s.getMemoryStats(myRawStats, dedot))
	}

	return formattedStats
}

func (s *MemoryService) getMemoryStats(myRawStat docker.Stat, dedot bool) MemoryData {
	totalRSS := myRawStat.Stats.MemoryStats.Stats["total_rss"]
	return MemoryData{
		Time:      common.Time(myRawStat.Stats.Read),
		Container: docker.NewContainer(myRawStat.Container, dedot),
		Failcnt:   myRawStat.Stats.MemoryStats.Failcnt,
		Limit:     myRawStat.Stats.MemoryStats.Limit,
		MaxUsage:  myRawStat.Stats.MemoryStats.MaxUsage,
		TotalRss:  totalRSS,
		TotalRssP: float64(totalRSS) / float64(myRawStat.Stats.MemoryStats.Limit),
		Usage:     myRawStat.Stats.MemoryStats.Usage,
		UsageP:    float64(myRawStat.Stats.MemoryStats.Usage) / float64(myRawStat.Stats.MemoryStats.Limit),
	}
}
