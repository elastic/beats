// +build darwin freebsd linux openbsd windows

package fsstat

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/system/filesystem"

	"github.com/pkg/errors"
)

var debugf = logp.MakeDebug("system-fsstat")

func init() {
	if err := mb.Registry.AddMetricSet("system", "fsstat", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

// MetricSet for fetching a summary of filesystem stats.
type MetricSet struct {
	mb.BaseMetricSet
	config filesystem.Config
}

// New creates and returns a new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var config filesystem.Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
	}, nil
}

// Fetch fetches filesystem metrics for all mounted filesystems and returns
// a single event containing aggregated data.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	fss, err := filesystem.GetFileSystemList()
	if err != nil {
		return nil, errors.Wrap(err, "filesystem list")
	}

	if len(m.config.IgnoreTypes) > 0 {
		fss = filesystem.Filter(fss, filesystem.BuildTypeFilter(m.config.IgnoreTypes...))
	}

	// These values are optional and could also be calculated by Kibana
	var totalFiles, totalSize, totalSizeFree, totalSizeUsed uint64
	dict := map[string]bool{}

	for _, fs := range fss {
		stat, err := filesystem.GetFileSystemStat(fs)
		if err != nil {
			debugf("error fetching filesystem stats for '%s': %v", fs.DirName, err)
			continue
		}
		logp.Debug("fsstat", "filesystem: %s total=%d, used=%d, free=%d", stat.Mount, stat.Total, stat.Used, stat.Free)

		if _, ok := dict[stat.Mount]; ok {
			// ignore filesystem with the same mounting point
			continue
		}

		totalFiles += stat.Files
		totalSize += stat.Total
		totalSizeFree += stat.Free
		totalSizeUsed += stat.Used

		dict[stat.Mount] = true

	}

	return common.MapStr{
		"total_size": common.MapStr{
			"free":  totalSizeFree,
			"used":  totalSizeUsed,
			"total": totalSize,
		},
		"count":       len(dict),
		"total_files": totalFiles,
	}, nil
}
