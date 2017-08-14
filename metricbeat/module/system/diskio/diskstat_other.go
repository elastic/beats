// +build darwin,cgo freebsd windows

package diskio

import (
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
)

func NewDiskIOStat() *DiskIOStat {
	d := &DiskIOStat{}
	d.lastDiskIOCounters = make(map[string]disk.IOCountersStat)
	return d
}

func (stat *DiskIOStat) OpenSampling() error {
	return nil
}

func (stat *DiskIOStat) CalIOStatistics(counter disk.IOCountersStat) (DiskIOMetric, error) {
	var result DiskIOMetric
	return result, errors.New("Not implemented out of linux")
}

func (stat *DiskIOStat) CloseSampling() {
	return
}
