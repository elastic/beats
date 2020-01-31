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

// +build linux

package diskio

import (
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"

	"github.com/elastic/beats/libbeat/metric/system/cpu"
)

func Get_CLK_TCK() uint32 {
	// return uint32(C.sysconf(C._SC_CLK_TCK))
	// NOTE: _SC_CLK_TCK should be fetched from sysconf using cgo
	return uint32(100)
}

// IOCounters should map functionality to disk package for linux os.
func IOCounters(names ...string) (map[string]disk.IOCountersStat, error) {
	stats, err := disk.IOCounters(names...)
	if err != nil {
		return nil, err
	}

	// Use `stats`, as `names` might be empty
	stats["summary"] = summaryIOCounter(stats)
	return stats, nil
}

func summaryIOCounter(stats map[string]disk.IOCountersStat) disk.IOCountersStat {

	topCounters := topLevelCounters(stats)

	sum := disk.IOCountersStat{}
	sum.Name = "summary"

	for _, counter := range topCounters {
		sum.ReadCount += counter.ReadCount
		sum.ReadBytes += counter.ReadBytes
		sum.ReadTime += counter.ReadTime
		sum.WriteCount += counter.WriteCount
		sum.WriteBytes += counter.WriteBytes
		sum.WriteTime += counter.WriteTime
		sum.IoTime += counter.IoTime
		sum.MergedReadCount += counter.MergedReadCount
		sum.MergedWriteCount += counter.MergedWriteCount
		sum.WeightedIO += counter.WeightedIO
	}

	return sum
}

// topLevelCounters will return a map of top level counters,
// and it removes only aggregate counters from the given map `stats`.
// For example, when `stats` with keys {"hda","hda1","sda1","sdb"} is given,
// it will return a map with keys {"hda","sda1","sdb"},
// and `stats` will remain elements with keys {"hda1","sda1","sdb"}
func topLevelCounters(stats map[string]disk.IOCountersStat) map[string]disk.IOCountersStat {

	result := map[string]disk.IOCountersStat{}

	hasChild := make(map[string]struct{})

Outer:
	for name := range stats {
		i := len(name)

		// Skip tailing numbers
		for c := name[i-1]; c <= '9' && c >= '0'; c = name[i-1] {
			i--
		}

		// a) If nothing skipped, the name represents a disk (Not NVMe).
		if i == len(name) {
			result[name] = stats[name]
			continue
		}

		// b) Search the stem for `name`.
		// Loop a bit as there are two naming conventions:
		// - For SATA and IDE devices(as well as RAID), a partition name of the disk "sda" would be "sda#"(# stands for numbers),
		// - For NVMe devices, the names would be like `nvme0n1`(disk) and `nvme0n1p1`(partition)
		for j := i; j > i-2; j-- {

			if _, ok := hasChild[name[:j]]; ok {
				continue Outer
			}

			if _, ok := stats[name[:j]]; ok {
				hasChild[name[:j]] = struct{}{}
				continue Outer
			}
		}

		// c) If no stem was found, add the element to `separated`,
		// This happens when:
		//   i) the stem is not contained,
		//   ii) it is a NVMe disk name.
		result[name] = stats[name]
	}

	for name := range hasChild {
		if _, ok := result[name]; !ok {
			result[name] = stats[name]
		}
		delete(stats, name)
	}

	return result
}

// NewDiskIOStat :init DiskIOStat object.
func NewDiskIOStat() *DiskIOStat {
	return &DiskIOStat{
		lastDiskIOCounters: map[string]disk.IOCountersStat{},
	}
}

// OpenSampling creates current cpu sampling
// need call as soon as get IOCounters.
func (stat *DiskIOStat) OpenSampling() error {
	return stat.curCpu.Get()
}

// CalIOStatistics calculates IO statistics.
func (stat *DiskIOStat) CalIOStatistics(result *DiskIOMetric, counter disk.IOCountersStat) error {
	var last disk.IOCountersStat
	var ok bool

	// if last counter not found, create one and return all 0
	if last, ok = stat.lastDiskIOCounters[counter.Name]; !ok {
		stat.lastDiskIOCounters[counter.Name] = counter
		return nil
	}

	// calculate the delta ms between the CloseSampling and OpenSampling
	deltams := 1000.0 * float64(stat.curCpu.Total()-stat.lastCpu.Total()) / float64(cpu.NumCores) / float64(Get_CLK_TCK())
	if deltams <= 0 {
		return errors.New("The delta cpu time between close sampling and open sampling is less or equal to 0")
	}

	rd_ios := counter.ReadCount - last.ReadCount
	rd_merges := counter.MergedReadCount - last.MergedReadCount
	rd_bytes := counter.ReadBytes - last.ReadBytes
	rd_ticks := counter.ReadTime - last.ReadTime
	wr_ios := counter.WriteCount - last.WriteCount
	wr_merges := counter.MergedWriteCount - last.MergedWriteCount
	wr_bytes := counter.WriteBytes - last.WriteBytes
	wr_ticks := counter.WriteTime - last.WriteTime
	ticks := counter.IoTime - last.IoTime
	aveq := counter.WeightedIO - last.WeightedIO
	n_ios := rd_ios + wr_ios
	n_ticks := rd_ticks + wr_ticks
	n_bytes := rd_bytes + wr_bytes
	size := float64(0)
	wait := float64(0)
	svct := float64(0)

	if n_ios > 0 {
		size = float64(n_bytes) / float64(n_ios)
		wait = float64(n_ticks) / float64(n_ios)
		svct = float64(ticks) / float64(n_ios)
	}

	queue := float64(aveq) / deltams
	per_sec := func(x uint64) float64 {
		return 1000.0 * float64(x) / deltams
	}

	result.ReadRequestMergeCountPerSec = per_sec(rd_merges)
	result.WriteRequestMergeCountPerSec = per_sec(wr_merges)
	result.ReadRequestCountPerSec = per_sec(rd_ios)
	result.WriteRequestCountPerSec = per_sec(wr_ios)
	result.ReadBytesPerSec = per_sec(rd_bytes)
	result.WriteBytesPerSec = per_sec(wr_bytes)
	result.AvgRequestSize = size
	result.AvgQueueSize = queue
	result.AvgAwaitTime = wait
	if rd_ios > 0 {
		result.AvgReadAwaitTime = float64(rd_ticks) / float64(rd_ios)
	}
	if wr_ios > 0 {
		result.AvgWriteAwaitTime = float64(wr_ticks) / float64(wr_ios)
	}
	result.AvgServiceTime = svct
	result.BusyPct = 100.0 * float64(ticks) / deltams
	if result.BusyPct > 100.0 {
		result.BusyPct = 100.0
	}

	stat.lastDiskIOCounters[counter.Name] = counter
	return nil

}

func (stat *DiskIOStat) CloseSampling() {
	stat.lastCpu = stat.curCpu
}
