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

//go:build darwin || freebsd || linux || windows || aix
// +build darwin freebsd linux windows aix

package diskio

import (
	"github.com/shirou/gopsutil/v3/disk"

	sigar "github.com/elastic/gosigar"
)

// IOMetric contains mapping fields which are outputed by `iostat -x` on linux
//
// Device:         rrqm/s   wrqm/s     r/s     w/s   rsec/s   wsec/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
// sda               0.06     0.78    0.09    0.27     9.42     8.06    48.64     0.00    1.34    0.99    1.45   0.77   0.03
type IOMetric struct {
	ReadRequestMergeCountPerSec  float64 `json:"rrqmCps"`
	WriteRequestMergeCountPerSec float64 `json:"wrqmCps"`
	ReadRequestCountPerSec       float64 `json:"rrqCps"`
	WriteRequestCountPerSec      float64 `json:"wrqCps"`
	// using bytes instead of sector
	ReadBytesPerSec   float64 `json:"rBps"`
	WriteBytesPerSec  float64 `json:"wBps"`
	AvgRequestSize    float64 `json:"avgrqSz"`
	AvgQueueSize      float64 `json:"avgquSz"`
	AvgAwaitTime      float64 `json:"await"`
	AvgReadAwaitTime  float64 `json:"r_await"`
	AvgWriteAwaitTime float64 `json:"w_await"`
	AvgServiceTime    float64 `json:"svctm"`
	BusyPct           float64 `json:"busy"`
}

// IOStat carries disk statistics for all devices
type IOStat struct {
	lastDiskIOCounters map[string]disk.IOCountersStat
	lastCPU            sigar.Cpu
	curCPU             sigar.Cpu
}
