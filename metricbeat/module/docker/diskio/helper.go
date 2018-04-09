package diskio

import (
	"time"

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/metricbeat/module/docker"
)

type BlkioStats struct {
	Time      time.Time
	Container *docker.Container
	reads     float64
	writes    float64
	totals    float64

	serviced      BlkioRaw
	servicedBytes BlkioRaw
}

type BlkioRaw struct {
	Time   time.Time
	reads  uint64
	writes uint64
	totals uint64
}

// BlkioService is a helper to collect and calculate disk I/O metrics
type BlkioService struct {
	lastStatsPerContainer map[string]BlkioRaw
}

// NewBlkioService builds a new initialized BlkioService
func NewBlkioService() *BlkioService {
	return &BlkioService{
		lastStatsPerContainer: make(map[string]BlkioRaw),
	}
}

func (io *BlkioService) getBlkioStatsList(rawStats []docker.Stat, dedot bool) []BlkioStats {
	formattedStats := []BlkioStats{}

	statsPerContainer := make(map[string]BlkioRaw)
	for _, myRawStats := range rawStats {
		stats := io.getBlkioStats(&myRawStats, dedot)
		statsPerContainer[myRawStats.Container.ID] = stats.serviced
		formattedStats = append(formattedStats, stats)
	}

	io.lastStatsPerContainer = statsPerContainer
	return formattedStats
}

func (io *BlkioService) getBlkioStats(myRawStat *docker.Stat, dedot bool) BlkioStats {
	newBlkioStats := io.getNewStats(myRawStat.Stats.Read, myRawStat.Stats.BlkioStats.IoServicedRecursive)
	bytesBlkioStats := io.getNewStats(myRawStat.Stats.Read, myRawStat.Stats.BlkioStats.IoServiceBytesRecursive)

	myBlkioStats := BlkioStats{
		Time:      myRawStat.Stats.Read,
		Container: docker.NewContainer(myRawStat.Container, dedot),

		serviced:      newBlkioStats,
		servicedBytes: bytesBlkioStats,
	}

	oldBlkioStats, exist := io.lastStatsPerContainer[myRawStat.Container.ID]
	if exist {
		myBlkioStats.reads = io.getReadPs(&oldBlkioStats, &newBlkioStats)
		myBlkioStats.writes = io.getWritePs(&oldBlkioStats, &newBlkioStats)
		myBlkioStats.totals = io.getTotalPs(&oldBlkioStats, &newBlkioStats)
	}

	return myBlkioStats
}

func (io *BlkioService) getNewStats(time time.Time, blkioEntry []types.BlkioStatEntry) BlkioRaw {
	stats := BlkioRaw{
		Time:   time,
		reads:  0,
		writes: 0,
		totals: 0,
	}

	for _, myEntry := range blkioEntry {
		switch myEntry.Op {
		case "Write":
			stats.writes += myEntry.Value
		case "Read":
			stats.reads += myEntry.Value
		case "Total":
			stats.totals += myEntry.Value
		}
	}
	return stats
}

func (io *BlkioService) getReadPs(old *BlkioRaw, new *BlkioRaw) float64 {
	duration := new.Time.Sub(old.Time)
	return calculatePerSecond(duration, old.reads, new.reads)
}

func (io *BlkioService) getWritePs(old *BlkioRaw, new *BlkioRaw) float64 {
	duration := new.Time.Sub(old.Time)
	return calculatePerSecond(duration, old.writes, new.writes)
}

func (io *BlkioService) getTotalPs(old *BlkioRaw, new *BlkioRaw) float64 {
	duration := new.Time.Sub(old.Time)
	return calculatePerSecond(duration, old.totals, new.totals)
}

func calculatePerSecond(duration time.Duration, old uint64, new uint64) float64 {
	value := float64(new) - float64(old)
	if value < 0 {
		value = 0
	}
	return value / duration.Seconds()
}
