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
	"errors"
	"math"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-system-metrics/metric"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/numcpu"
)

// GetCLKTCK emulates the _SC_CLK_TCK syscall
func GetCLKTCK() uint32 {
	// return uint32(C.sysconf(C._SC_CLK_TCK))
	// NOTE: _SC_CLK_TCK should be fetched from sysconf using cgo
	return uint32(100)
}

// IOCounters should map functionality to disk package for linux os.
func IOCounters(_ *logp.Logger, names ...string) (map[string]disk.IOCountersStat, error) {
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
	times, err := cpu.Times(false)
	if err != nil {
		return err
	}
	if len(times) > 0 {
		stat.curCPU = times[0]
	}
	return nil
}

// a few of the diskio counters are actually 32-bit on the kernel side, which means they can roll over fairly easily.
// Here we try to reconstruct the values by calculating the pre-rollover delta from unt32 max, then adding.
// If you want to get technical, this could be a tad unsafe, as we don't actually have any way of knowing if the word size changes in a future kernel, and we've rolled over at UINT64_MAX

// See https://docs.kernel.org/admin-guide/iostats.html and https://github.com/torvalds/linux/blob/master/block/genhd.c diskstats_show()
func returnOrFix32BitRollover(current, prev uint64) uint64 {
	var maxUint32 uint64 = math.MaxUint32 //4_294_967_295 Max value in uint32/unsigned int

	if current >= prev {
		return current - prev
	}
	// we're at a uint64 if we hit this
	if prev > maxUint32 {
		return 0
	}

	delta := maxUint32 - prev

	return delta + current

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
	deltams := 1000.0 * (float64(uint64(total(stat.curCPU)) - uint64(total(stat.lastCPU)))) / float64(numcpu.NumCPUWithLogger(logp.NewLogger("")))
	if deltams <= 0 {
		return IOMetric{}, errors.New("the delta cpu time between close sampling and open sampling is less or equal to 0")
	}

	rdIOs := counter.ReadCount - last.ReadCount
	rdMerges := counter.MergedReadCount - last.MergedReadCount
	rdBytes := counter.ReadBytes - last.ReadBytes
	rdTicks := returnOrFix32BitRollover(counter.ReadTime, last.ReadTime)
	wrIOs := counter.WriteCount - last.WriteCount
	wrMerges := counter.MergedWriteCount - last.MergedWriteCount
	wrBytes := counter.WriteBytes - last.WriteBytes
	wrTicks := returnOrFix32BitRollover(counter.WriteTime, last.WriteTime)
	ticks := returnOrFix32BitRollover(counter.IoTime, last.IoTime)
	aveq := returnOrFix32BitRollover(counter.WeightedIO, last.WeightedIO)

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
		return metric.Round(1000.0 * float64(x) / deltams)
	}

	result := IOMetric{}
	result.ReadRequestMergeCountPerSec = perSec(rdMerges)
	result.WriteRequestMergeCountPerSec = perSec(wrMerges)
	result.ReadRequestCountPerSec = perSec(rdIOs)
	result.WriteRequestCountPerSec = perSec(wrIOs)
	result.ReadBytesPerSec = perSec(rdBytes)
	result.WriteBytesPerSec = perSec(wrBytes)
	result.AvgRequestSize = metric.Round(size)
	result.AvgQueueSize = metric.Round(queue)
	result.AvgAwaitTime = metric.Round(wait)
	if rdIOs > 0 {
		result.AvgReadAwaitTime = metric.Round(float64(rdTicks) / float64(rdIOs))
	}
	if wrIOs > 0 {
		result.AvgWriteAwaitTime = metric.Round(float64(wrTicks) / float64(wrIOs))
	}
	result.AvgServiceTime = metric.Round(svct)
	result.BusyPct = metric.Round(100.0 * float64(ticks) / deltams)
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

func total(cpu cpu.TimesStat) float64 {
	return cpu.User + cpu.Nice + cpu.System + cpu.Idle +
		cpu.Iowait + cpu.Irq + cpu.Softirq + cpu.Steal
}
