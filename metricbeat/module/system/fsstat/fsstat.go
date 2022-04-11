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

//go:build darwin || freebsd || linux || openbsd || windows
// +build darwin freebsd linux openbsd windows

package fsstat

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/system/filesystem"
	fs "github.com/elastic/elastic-agent-system-metrics/metric/system/filesystem"
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
	sys    resolve.Resolver
}

// New creates and returns a new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var config filesystem.Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	sys, _ := base.Module().(resolve.Resolver)
	if config.IgnoreTypes == nil {
		config.IgnoreTypes = fs.DefaultIgnoredTypes(sys)
	}
	if len(config.IgnoreTypes) > 0 {
		base.Logger().Info("Ignoring filesystem types: %s", strings.Join(config.IgnoreTypes, ", "))
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		sys:           sys,
	}, nil
}

// Fetch fetches filesystem metrics for all mounted filesystems and returns
// a single event containing aggregated data.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	fsList, err := fs.GetFilesystems(m.sys, fs.BuildFilterWithList(m.config.IgnoreTypes))
	if err != nil {
		return fmt.Errorf("error fetching filesystem list: %w", err)
	}

	// These values are optional and could also be calculated by Kibana
	var totalFiles, totalSize, totalSizeFree, totalSizeUsed uint64

	for _, fs := range fsList {
		err := fs.GetUsage()
		if err != nil {
			m.Logger().Debugf("error fetching filesystem stats for '%s': %v", fs.Directory, err)
			continue
		}
		m.Logger().Debugf("filesystem: %s total=%d, used=%d, free=%d", fs.Directory, fs.Total, fs.Used.Bytes.ValueOr(0), fs.Free)

		totalFiles += fs.Files.ValueOr(0)
		totalSize += fs.Total.ValueOr(0)
		totalSizeFree += fs.Free.ValueOr(0)
		totalSizeUsed += fs.Used.Bytes.ValueOr(0)
	}

	event := common.MapStr{
		"total_size": common.MapStr{
			"free":  totalSizeFree,
			"used":  totalSizeUsed,
			"total": totalSize,
		},
		"count":       len(fsList),
		"total_files": totalFiles,
	}

	//We don't get the `Files` field on Windows
	if runtime.GOOS == "windows" {
		event["total_files"] = totalFiles
	}

	r.Event(mb.Event{
		MetricSetFields: event,
	})

	return nil
}
