// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package diskio

import (
	"strings"
	"time"

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/v8/metricbeat/module/docker"
)

// BlkioStats contains all formatted blkio stats
type BlkioStats struct {
	Time      time.Time
	Container *docker.Container
	reads     float64
	writes    float64
	totals    float64

	serviced      BlkioRaw
	servicedBytes BlkioRaw
	servicedTime  BlkioRaw
	waitTime      BlkioRaw
	queued        BlkioRaw
}

// Add adds blkio stats
func (s *BlkioStats) Add(o *BlkioStats) {
	s.reads += o.reads
	s.writes += o.writes
	s.totals += o.totals

	s.serviced.Add(&o.serviced)
	s.servicedBytes.Add(&o.servicedBytes)
}

// BlkioRaw sums raw Blkio stats
type BlkioRaw struct {
	Time   time.Time
	reads  uint64
	writes uint64
	totals uint64
}

// Add adds blkio raw stats
func (s *BlkioRaw) Add(o *BlkioRaw) {
	s.reads += o.reads
	s.writes += o.writes
	s.totals += o.totals
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

func (io *BlkioService) getBlkioStatsList(rawStats []docker.Stat, dedot bool, skipDev []uint64) []BlkioStats {
	formattedStats := []BlkioStats{}

	statsPerContainer := make(map[string]BlkioRaw)
	for i := range rawStats {
		stats := io.getBlkioStats(&rawStats[i], dedot, skipDev)
		storageStats := io.getStorageStats(&rawStats[i], dedot)
		stats.Add(&storageStats)

		oldStats, exist := io.lastStatsPerContainer[stats.Container.ID]
		if exist {
			stats.reads = io.getReadPs(&oldStats, &stats.serviced)
			stats.writes = io.getWritePs(&oldStats, &stats.serviced)
			stats.totals = io.getTotalPs(&oldStats, &stats.serviced)
		}

		statsPerContainer[stats.Container.ID] = stats.serviced
		formattedStats = append(formattedStats, stats)
	}

	io.lastStatsPerContainer = statsPerContainer
	return formattedStats
}

// getStorageStats collects diskio metrics from StorageStats structure, that
// is populated in Windows systems only
func (io *BlkioService) getStorageStats(myRawStats *docker.Stat, dedot bool) BlkioStats {
	return BlkioStats{
		Time:      myRawStats.Stats.Read,
		Container: docker.NewContainer(myRawStats.Container, dedot),

		serviced: BlkioRaw{
			reads:  myRawStats.Stats.StorageStats.ReadCountNormalized,
			writes: myRawStats.Stats.StorageStats.WriteCountNormalized,
			totals: myRawStats.Stats.StorageStats.ReadCountNormalized + myRawStats.Stats.StorageStats.WriteCountNormalized,
		},

		servicedBytes: BlkioRaw{
			reads:  myRawStats.Stats.StorageStats.ReadSizeBytes,
			writes: myRawStats.Stats.StorageStats.WriteSizeBytes,
			totals: myRawStats.Stats.StorageStats.ReadSizeBytes + myRawStats.Stats.StorageStats.WriteSizeBytes,
		},
	}
}

// getBlkioStats collects diskio metrics from BlkioStats structures, that
// are not populated in Windows
func (io *BlkioService) getBlkioStats(myRawStat *docker.Stat, dedot bool, skipDev []uint64) BlkioStats {
	return BlkioStats{
		Time:      myRawStat.Stats.Read,
		Container: docker.NewContainer(myRawStat.Container, dedot),

		serviced: getNewStats(
			skipDev,
			myRawStat.Stats.Read,
			myRawStat.Stats.BlkioStats.IoServicedRecursive),
		servicedBytes: getNewStats(
			skipDev,
			myRawStat.Stats.Read,
			myRawStat.Stats.BlkioStats.IoServiceBytesRecursive),
		servicedTime: getNewStats(
			skipDev,
			myRawStat.Stats.Read,
			myRawStat.Stats.BlkioStats.IoServiceTimeRecursive),
		waitTime: getNewStats(
			skipDev,
			myRawStat.Stats.Read,
			myRawStat.Stats.BlkioStats.IoWaitTimeRecursive),
		queued: getNewStats(
			skipDev,
			myRawStat.Stats.Read,
			myRawStat.Stats.BlkioStats.IoQueuedRecursive),
	}
}

func getNewStats(skip []uint64, time time.Time, blkioEntry []types.BlkioStatEntry) BlkioRaw {
	stats := BlkioRaw{
		Time:   time,
		reads:  0,
		writes: 0,
		totals: 0,
	}

	for _, myEntry := range blkioEntry {

		// certain devices, like software raid and device-mapper devices, will just control and re-report the disks
		// under them in the hierarchy. We want to skip them, lest we merely duplicate the metrics.
		if skipDev(myEntry.Major, skip) {
			continue
		}
		// These op value strings can either be Capitalized or lowercase, depending on the platform.
		switch strings.ToLower(myEntry.Op) {
		case "write":
			stats.writes += myEntry.Value
		case "read":
			stats.reads += myEntry.Value
		case "total":
			stats.totals += myEntry.Value
		}
	}

	return stats
}

func skipDev(major uint64, skipList []uint64) bool {
	for _, dev := range skipList {
		if major == dev {
			return true
		}
	}
	return false
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

	timeSec := duration.Seconds()
	if timeSec == 0 {
		return 0
	}

	return value / timeSec
}
