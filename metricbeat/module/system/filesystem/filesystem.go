// +build darwin linux openbsd windows

/*

An example event looks as following:

    {
      "@timestamp": "2016-04-26T19:30:19.475Z",
      "beat": {
        "hostname": "ruflin",
        "name": "ruflin"
      },
      "metricset": "filesystem",
      "module": "system",
      "rtt": 434,
      "system-filesystem": {
        "avail": 41159540736,
        "device_name": "/dev/disk1",
        "files": 60981246,
        "free": 41421684736,
        "free_files": 10048716,
        "mount_point": "/",
        "total": 249779191808,
        "used": 208357507072,
        "used_p": 0.83
      },
      "type": "metricsets"
    }


*/

package filesystem

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/topbeat/system"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "filesystem", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
}

// New creates new instance of MetricSeter
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

func (m *MetricSet) Fetch(host string) (events []common.MapStr, err error) {

	fss, err := system.GetFileSystemList()
	if err != nil {
		logp.Warn("Getting filesystem list: %v", err)
		return nil, err
	}

	filesSystems := []common.MapStr{}

	for _, fs := range fss {
		fsStat, err := system.GetFileSystemStat(fs)
		if err != nil {
			logp.Debug("filesystem", "Skip filesystem %d: %v", fsStat, err)
			continue
		}
		system.AddFileSystemUsedPercentage(fsStat)
		stat := system.GetFilesystemEvent(fsStat)

		filesSystems = append(filesSystems, stat)
	}

	return filesSystems, nil
}
