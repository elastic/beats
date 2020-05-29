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

package conntrack

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/procfs"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/linux"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("linux", "conntrack", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	fs procfs.FS
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The linux conntrack metricset is beta.")
	linuxModule, ok := base.Module().(*linux.Module)
	if !ok {
		return nil, errors.New("unexpected module type")
	}

	path := filepath.Join(linuxModule.HostFS, "proc")
	newFS, err := procfs.NewFS(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating new Host FS at %s", path)
	}

	return &MetricSet{
		BaseMetricSet: base,
		fs:            newFS,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	conntrackStats, err := m.fs.ConntrackStat()
	if err != nil {
		return errors.Wrap(err, "error fetching conntrack stats")
	}

	summedEvents := procfs.ConntrackStatEntry{}
	for _, conn := range conntrackStats {
		summedEvents.Entries += conn.Entries
		summedEvents.Found += conn.Found
		summedEvents.Invalid += conn.Invalid
		summedEvents.Ignore += conn.Ignore
		summedEvents.InsertFailed += conn.InsertFailed
		summedEvents.Drop += conn.Drop
		summedEvents.EarlyDrop += conn.EarlyDrop
		summedEvents.SearchRestart += conn.SearchRestart
	}

	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"summary": common.MapStr{
				"entries":        summedEvents.Entries,
				"found":          summedEvents.Found,
				"invalid":        summedEvents.Invalid,
				"ignore":         summedEvents.Ignore,
				"insert_failed":  summedEvents.InsertFailed,
				"drop":           summedEvents.Drop,
				"early_drop":     summedEvents.EarlyDrop,
				"search_restart": summedEvents.SearchRestart,
			},
		},
	})

	return nil
}
