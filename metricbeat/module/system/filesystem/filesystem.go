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

package filesystem

import (
	"strings"

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
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	fss, err := GetFileSystemList()
	if err != nil {
		return errors.Wrap(err, "error getting filesystem list")

	}

	if len(m.config.IgnoreTypes) > 0 {
		fss = Filter(fss, BuildTypeFilter(m.config.IgnoreTypes...))
	}

	for _, fs := range fss {
		fsStat, err := GetFileSystemStat(fs)
		if err != nil {
			debugf("error getting filesystem stats for '%s': %v", fs.DirName, err)
			continue
		}
		AddFileSystemUsedPercentage(fsStat)

		event := mb.Event{
			MetricSetFields: GetFilesystemEvent(fsStat),
		}
		if !r.Event(event) {
			return nil
		}
	}
	return nil
}
