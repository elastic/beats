// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

var debugf = logp.MakeDebug("system-filesystem")

func init() {
	if err := mb.Registry.AddMetricSet("system", "filesystem", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching filesystem metrics.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates and returns a new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch fetches filesystem metrics for all mounted filesystems and returns
// an event for each mount point.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	fss, err := GetFileSystemList()
	if err != nil {
		return nil, errors.Wrap(err, "filesystem list")
	}

	filesSystems := make([]common.MapStr, 0, len(fss))
	for _, fs := range fss {
		fsStat, err := GetFileSystemStat(fs)
		if err != nil {
			debugf("error getting filesystem stats for '%s': %v", fs.DirName, err)
			continue
		}
		AddFileSystemUsedPercentage(fsStat)
		filesSystems = append(filesSystems, GetFilesystemEvent(fsStat))
	}

	return filesSystems, nil
}
