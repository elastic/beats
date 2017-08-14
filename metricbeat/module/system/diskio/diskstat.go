// +build darwin freebsd linux windows

package diskio

import (
	"github.com/shirou/gopsutil/disk"

	sigar "github.com/elastic/gosigar"
)

// mapping fields which output by `iostat -x` on linux
//
// Device:         rrqm/s   wrqm/s     r/s     w/s   rsec/s   wsec/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
// sda               0.06     0.78    0.09    0.27     9.42     8.06    48.64     0.00    1.34    0.99    1.45   0.77   0.03
type DiskIOMetric struct {
	ReadRequestMergeCountPerSec  float64 `json:"rrqmCps"`
	WriteRequestMergeCountPerSec float64 `json:"wrqmCps"`
	ReadRequestCountPerSec       float64 `json:"rrqCps"`
	WriteRequestCountPerSec      float64 `json:"wrqCps"`
	// using bytes instead of sector
	ReadBytesPerSec  float64 `json:"rBps"`
	WriteBytesPerSec float64 `json:"wBps"`
	AvgRequestSize   float64 `json:"avgrqSz"`
	AvgQueueSize     float64 `json:"avgquSz"`
	AvgAwaitTime     float64 `json:"await"`
	AvgServiceTime   float64 `json:"svctm"`
	BusyPct          float64 `json:"busy"`
}

type DiskIOStat struct {
	lastDiskIOCounters map[string]disk.IOCountersStat
	lastCpu            sigar.Cpu
	curCpu             sigar.Cpu
}
