package diskio

import (
	"time"

	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/metricbeat/module/docker"
)

type BlkioStats struct {
	Time      time.Time
	Container *docker.Container
	reads     float64
	writes    float64
	totals    float64
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

func (io *BLkioService) getBlkioStatsList(rawStats []docker.DockerStat) []BlkioStats {
	formatedStats := []BlkioStats{}

	for _, myRawStats := range rawStats {
		formatedStats = append(formatedStats, io.getBlkioStats(&myRawStats))
	}

	return formatedStats
}

func (io *BLkioService) getBlkioStats(myRawStat *docker.DockerStat) BlkioStats {

	newBlkioStats := io.getNewStats(myRawStat.Stats.Read, myRawStat.Stats.BlkioStats.IOServicedRecursive)
	oldBlkioStats, exist := io.BlkioSTatsPerContainer[myRawStat.Container.ID]

	myBlkioStats := BlkioStats{
		Time:      myRawStat.Stats.Read,
		Container: docker.NewContainer(&myRawStat.Container),
	}

	if exist {
		myBlkioStats.reads = io.getReadPs(&oldBlkioStats, &newBlkioStats)
		myBlkioStats.writes = io.getWritePs(&oldBlkioStats, &newBlkioStats)
		myBlkioStats.totals = io.getReadPs(&oldBlkioStats, &newBlkioStats)
	} else {
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
	return calculatePerSecond(duration, old.reads, new.reads)
}

func (io *BLkioService) getWritePs(old *BlkioRaw, new *BlkioRaw) float64 {
	duration := new.Time.Sub(old.Time)
	return calculatePerSecond(duration, old.writes, new.writes)
}

func (io *BLkioService) getTotalPs(old *BlkioRaw, new *BlkioRaw) float64 {
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
