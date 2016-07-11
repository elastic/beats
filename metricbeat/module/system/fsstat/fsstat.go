// +build darwin freebsd linux openbsd windows

package fsstat

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/system/filesystem"

	"github.com/pkg/errors"
)

var debugf = logp.MakeDebug("system-fsstat")

func init() {
	if err := mb.Registry.AddMetricSet("system", "fsstat", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching a summary of filesystem stats.
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
// a single event containing aggregated data.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	fss, err := filesystem.GetFileSystemList()
	if err != nil {
		return nil, errors.Wrap(err, "filesystem list")
	}

	// These values are optional and could also be calculated by Kibana
	var totalFiles, totalSize, totalSizeFree, totalSizeUsed uint64

	for _, fs := range fss {
		fsStat, err := filesystem.GetFileSystemStat(fs)
		if err != nil {
			debugf("error fetching filesystem stats for '%s': %v", fs.DirName, err)
			continue
		}

		totalFiles += fsStat.Files
		totalSize += fsStat.Total
		totalSizeFree += fsStat.Free
		totalSizeUsed += fsStat.Used
	}

	return common.MapStr{
		"total_size": common.MapStr{
			"free":  totalSizeFree,
			"used":  totalSizeUsed,
			"total": totalSize,
		},
		"count":       len(fss),
		"total_files": totalFiles,
	}, nil
}
