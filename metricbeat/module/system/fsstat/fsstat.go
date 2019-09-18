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

// +build darwin freebsd linux openbsd windows

package fsstat

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/system/filesystem"

	"github.com/pkg/errors"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "fsstat", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
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

	if config.IgnoreTypes == nil {
		config.IgnoreTypes = filesystem.DefaultIgnoredTypes()
	}
	if len(config.IgnoreTypes) > 0 {
		base.Logger().Info("Ignoring filesystem types: %s", strings.Join(config.IgnoreTypes, ", "))
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
	}, nil
}

// Fetch fetches filesystem metrics for all mounted filesystems and returns
// a single event containing aggregated data.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	fss, err := filesystem.GetFileSystemList()
	if err != nil {
		return errors.Wrap(err, "filesystem list")
	}

	if len(m.config.IgnoreTypes) > 0 {
		fss = filesystem.Filter(fss, filesystem.BuildTypeFilter(m.config.IgnoreTypes...))
	}

	// These values are optional and could also be calculated by Kibana
	var totalFiles, totalSize, totalSizeFree, totalSizeUsed uint64

	for _, fs := range fss {
		stat, err := filesystem.GetFileSystemStat(fs)
		if err != nil {
			m.Logger().Debugf("error fetching filesystem stats for '%s': %v", fs.DirName, err)
			continue
		}
		m.Logger().Debugf("filesystem: %s total=%d, used=%d, free=%d", stat.Mount, stat.Total, stat.Used, stat.Free)

		totalFiles += stat.Files
		totalSize += stat.Total
		totalSizeFree += stat.Free
		totalSizeUsed += stat.Used
	}

	r.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"total_size": common.MapStr{
				"free":  totalSizeFree,
				"used":  totalSizeUsed,
				"total": totalSize,
			},
			"count":       len(fss),
			"total_files": totalFiles,
		},
	})

	return nil
}
