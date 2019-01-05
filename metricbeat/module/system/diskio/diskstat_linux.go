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
	//return uint32(C.sysconf(C._SC_CLK_TCK))
	//NOTE: _SC_CLK_TCK should be fetched from sysconf using cgo
	return uint32(100)
}

func NewDiskIOStat() *DiskIOStat {
	d := &DiskIOStat{}
	d.lastDiskIOCounters = make(map[string]disk.IOCountersStat)
	return d
}

// create current cpu sampling
// need call as soon as get IOCounters
func (stat *DiskIOStat) OpenSampling() error {
	return stat.curCpu.Get()
}

func (stat *DiskIOStat) CalIOStatistics(counter disk.IOCountersStat) (DiskIOMetric, error) {
	var last disk.IOCountersStat
	var ok bool
	var result DiskIOMetric

	// if last counter not found, create one and return all 0
	if last, ok = stat.lastDiskIOCounters[counter.Name]; !ok {
		stat.lastDiskIOCounters[counter.Name] = counter
		return result, nil
	}

	// calculate the delta ms between the CloseSampling and OpenSampling
	deltams := 1000.0 * float64(stat.curCpu.Total()-stat.lastCpu.Total()) / float64(cpu.NumCores) / float64(Get_CLK_TCK())
	if deltams <= 0 {
		return result, errors.New("The delta cpu time between close sampling and open sampling is less or equal to 0")
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
	return result, nil

}

func (stat *DiskIOStat) CloseSampling() {
	stat.lastCpu = stat.curCpu
}
