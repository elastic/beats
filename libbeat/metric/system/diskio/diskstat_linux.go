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

//go:build linux
// +build linux

package diskio

import (
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"

	"github.com/elastic/beats/v7/libbeat/metric/system/numcpu"
)

// GetCLKTCK emulates the _SC_CLK_TCK syscall
func GetCLKTCK() uint32 {
	// return uint32(C.sysconf(C._SC_CLK_TCK))
	// NOTE: _SC_CLK_TCK should be fetched from sysconf using cgo
	return uint32(100)
}

// IOCounters should map functionality to disk package for linux os.
func IOCounters(names ...string) (map[string]disk.IOCountersStat, error) {
	return disk.IOCounters(names...)
}

// NewDiskIOStat :init DiskIOStat object.
func NewDiskIOStat() *IOStat {
	return &IOStat{
		lastDiskIOCounters: map[string]disk.IOCountersStat{},
	}
}

// OpenSampling creates current cpu sampling
// need call as soon as get IOCounters.
func (stat *IOStat) OpenSampling() error {
	return stat.curCPU.Get()
}

// CalcIOStatistics calculates IO statistics.
func (stat *IOStat) CalcIOStatistics(counter disk.IOCountersStat) (IOMetric, error) {
	var last disk.IOCountersStat
	var ok bool

	// if last counter not found, create one and return all 0
	if last, ok = stat.lastDiskIOCounters[counter.Name]; !ok {
		stat.lastDiskIOCounters[counter.Name] = counter
		return IOMetric{}, nil
	}

	// calculate the delta ms between the CloseSampling and OpenSampling
	deltams := 1000.0 * float64(stat.curCPU.Total()-stat.lastCPU.Total()) / float64(numcpu.NumCPU()) / float64(GetCLKTCK())
	if deltams <= 0 {
		return IOMetric{}, errors.New("The delta cpu time between close sampling and open sampling is less or equal to 0")
	}

	rdIOs := counter.ReadCount - last.ReadCount
	rdMerges := counter.MergedReadCount - last.MergedReadCount
	rdBytes := counter.ReadBytes - last.ReadBytes
	rdTicks := counter.ReadTime - last.ReadTime
	wrIOs := counter.WriteCount - last.WriteCount
	wrMerges := counter.MergedWriteCount - last.MergedWriteCount
	wrBytes := counter.WriteBytes - last.WriteBytes
	wrTicks := counter.WriteTime - last.WriteTime
	ticks := counter.IoTime - last.IoTime
	aveq := counter.WeightedIO - last.WeightedIO
	nIOs := rdIOs + wrIOs
	nTicks := rdTicks + wrTicks
	nBytes := rdBytes + wrBytes
	size := float64(0)
	wait := float64(0)
	svct := float64(0)

	if nIOs > 0 {
		size = float64(nBytes) / float64(nIOs)
		wait = float64(nTicks) / float64(nIOs)
		svct = float64(ticks) / float64(nIOs)
	}

	queue := float64(aveq) / deltams
	perSec := func(x uint64) float64 {
		return 1000.0 * float64(x) / deltams
	}

	result := IOMetric{}
	result.ReadRequestMergeCountPerSec = perSec(rdMerges)
	result.WriteRequestMergeCountPerSec = perSec(wrMerges)
	result.ReadRequestCountPerSec = perSec(rdIOs)
	result.WriteRequestCountPerSec = perSec(wrIOs)
	result.ReadBytesPerSec = perSec(rdBytes)
	result.WriteBytesPerSec = perSec(wrBytes)
	result.AvgRequestSize = size
	result.AvgQueueSize = queue
	result.AvgAwaitTime = wait
	if rdIOs > 0 {
		result.AvgReadAwaitTime = float64(rdTicks) / float64(rdIOs)
	}
	if wrIOs > 0 {
		result.AvgWriteAwaitTime = float64(wrTicks) / float64(wrIOs)
	}
	result.AvgServiceTime = svct
	result.BusyPct = 100.0 * float64(ticks) / deltams
	if result.BusyPct > 100.0 {
		result.BusyPct = 100.0
	}

	stat.lastDiskIOCounters[counter.Name] = counter
	return result, nil

}

// CloseSampling closes the disk sampler
func (stat *IOStat) CloseSampling() {
	stat.lastCPU = stat.curCPU
}
