package diskio

import (
	"time"

	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/module/docker"
)

type BlkioStats struct {
	Time        time.Time
	MyContainer *docker.Container
	reads       float64
	writes      float64
	totals      float64
}
type BlkioCalculator interface {
	getReadPs(old *BlkioRaw, new *BlkioRaw) float64
	getWritePs(old *BlkioRaw, new *BlkioRaw) float64
	getTotalPs(old *BlkioRaw, new *BlkioRaw) float64
}

type BlkioRaw struct {
	Time   time.Time
	reads  uint64
	writes uint64
	totals uint64
}
type BLkioService struct {
	BlkioSTatsPerContainer map[string]BlkioRaw
}

func (io *BLkioService) GetBlkioStatsList(rawStats []docker.DockerStat) []BlkioStats {
	formatedStats := []BlkioStats{}
	if len(rawStats) != 0 {
		for _, myRawStats := range rawStats {
			formatedStats = append(formatedStats, io.getBlkioStats(&myRawStats))
		}
	} else {
		logp.Info("No container is running")
	}
	return formatedStats
}
func (io *BLkioService) getBlkioStats(myRawStat *docker.DockerStat) BlkioStats {

	myBlkioStats := BlkioStats{}
	newBlkioStats := io.getNewStats(myRawStat.Stats.Read, myRawStat.Stats.BlkioStats.IOServicedRecursive)
	oldBlkioStats, exist := io.BlkioSTatsPerContainer[myRawStat.Container.ID]

	if exist {
		myBlkioStats = BlkioStats{
			Time:        myRawStat.Stats.Read,
			MyContainer: docker.InitCurrentContainer(&myRawStat.Container),
			reads:       io.getReadPs(&oldBlkioStats, &newBlkioStats),
			writes:      io.getWritePs(&oldBlkioStats, &newBlkioStats),
			totals:      io.getReadPs(&oldBlkioStats, &newBlkioStats),
		}
	} else {
		myBlkioStats = BlkioStats{
			Time:        myRawStat.Stats.Read,
			MyContainer: docker.InitCurrentContainer(&myRawStat.Container),
			reads:       0,
			writes:      0,
			totals:      0,
		}
	}
	if _, exist := io.BlkioSTatsPerContainer[myRawStat.Container.ID]; !exist {
		io.BlkioSTatsPerContainer = make(map[string]BlkioRaw)
	}
	io.BlkioSTatsPerContainer[myRawStat.Container.ID] = newBlkioStats

	return myBlkioStats
}

func (io *BLkioService) getNewStats(time time.Time, blkioEntry []dc.BlkioStatsEntry) BlkioRaw {
	stats := BlkioRaw{
		Time:   time,
		reads:  0,
		writes: 0,
		totals: 0,
	}
	for _, myEntry := range blkioEntry {
		if myEntry.Op == "Write" {
			stats.writes = myEntry.Value
		} else if myEntry.Op == "Read" {
			stats.reads = myEntry.Value
		} else if myEntry.Op == "Total" {
			stats.totals = myEntry.Value
		}
	}
	return stats
}

func (io *BLkioService) getReadPs(old *BlkioRaw, new *BlkioRaw) float64 {
	duration := new.Time.Sub(old.Time)
	return io.calculatePerSecond(duration, old.reads, new.reads)
}
func (io *BLkioService) getWritePs(old *BlkioRaw, new *BlkioRaw) float64 {
	duration := new.Time.Sub(old.Time)
	return io.calculatePerSecond(duration, old.writes, new.writes)
}
func (io *BLkioService) getTotalPs(old *BlkioRaw, new *BlkioRaw) float64 {
	duration := new.Time.Sub(old.Time)
	return io.calculatePerSecond(duration, old.totals, new.totals)
}

func (io *BLkioService) calculatePerSecond(duration time.Duration, old uint64, new uint64) float64 {
	return float64(new-old) / duration.Seconds()
}
