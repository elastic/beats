// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	"github.com/pkg/errors"
)

var debugf = logp.MakeDebug("system.filesystem")

func init() {
	mb.Registry.MustAddMetricSet("system", "filesystem", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// MetricSet for fetching filesystem metrics.
type MetricSet struct {
	mb.BaseMetricSet
	config Config
}

// New creates and returns a new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.IgnoreTypes == nil {
		config.IgnoreTypes = DefaultIgnoredTypes()
	}
	if len(config.IgnoreTypes) > 0 {
		logp.Info("Ignoring filesystem types: %s", strings.Join(config.IgnoreTypes, ", "))
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
	}, nil
}

// Fetch fetches filesystem metrics for all mounted filesystems and returns
// an event for each mount point.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	fss, err := GetFileSystemList()
	if err != nil {
		return nil, errors.Wrap(err, "filesystem list")
	}

	if len(m.config.IgnoreTypes) > 0 {
		fss = Filter(fss, BuildTypeFilter(m.config.IgnoreTypes...))
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
