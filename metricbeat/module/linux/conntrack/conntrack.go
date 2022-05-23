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
	"github.com/pkg/errors"
	"github.com/prometheus/procfs"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
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
	mod resolve.Resolver
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The linux conntrack metricset is beta.")

	sys := base.Module().(resolve.Resolver)

	return &MetricSet{
		BaseMetricSet: base,
		mod:           sys,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	newFS, err := procfs.NewFS(m.mod.ResolveHostFS("/proc"))
	if err != nil {
		return errors.Wrapf(err, "error creating new Host FS at %s", m.mod.ResolveHostFS("/proc"))
	}
	conntrackStats, err := newFS.ConntrackStat()
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
		MetricSetFields: mapstr.M{
			"summary": mapstr.M{
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
