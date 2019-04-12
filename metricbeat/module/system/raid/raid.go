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

package raid

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/procfs"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/system"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "raid", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// MetricSet contains proc fs data.
type MetricSet struct {
	mb.BaseMetricSet
	fs procfs.FS
}

// New creates a new instance of the raid metricset.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	systemModule, ok := base.Module().(*system.Module)
	if !ok {
		return nil, errors.New("unexpected module type")
	}

	// Additional configuration options
	config := struct {
		MountPoint string `config:"raid.mount_point"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.MountPoint == "" {
		config.MountPoint = systemModule.HostFS
	}

	mountPoint := filepath.Join(config.MountPoint, procfs.DefaultMountPoint)
	fs, err := procfs.NewFS(mountPoint)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		fs:            fs,
	}, nil
}

// Fetch fetches one event for each device
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	stats, err := m.fs.ParseMDStat()
	if err != nil {
		r.Error(errors.Wrap(err, "failed to parse mdstat"))
		return
	}

	for _, stat := range stats {
		event := common.MapStr{
			"name":           stat.Name,
			"activity_state": stat.ActivityState,
			"disks": common.MapStr{
				"active": stat.DisksActive,
				"total":  stat.DisksTotal,
			},
			"blocks": common.MapStr{
				"synced": stat.BlocksSynced,
				"total":  stat.BlocksTotal,
			},
		}

		r.Event(mb.Event{
			MetricSetFields: event,
		})
	}
}
