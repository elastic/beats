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
	"fmt"
	"os"

	"github.com/prometheus/procfs"
	"github.com/ti-mo/conntrack"

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

type fetchFunc func() ([]procfs.ConntrackStatEntry, error)

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	mod       resolve.Resolver
	fetchFunc fetchFunc
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	base.Logger().Warn(cfgwarn.Beta("The linux conntrack metricset is beta."))

	sys, ok := base.Module().(resolve.Resolver)
	if !ok {
		return nil, fmt.Errorf("unexpected module type: %T", base.Module())
	}

	mset := &MetricSet{
		BaseMetricSet: base,
		mod:           sys,
	}

	err := mset.selectMetricsSource()
	if err != nil {
		return nil, fmt.Errorf("error selecting metrics source: %w", err)
	}

	return mset, nil
}

func (m *MetricSet) selectMetricsSource() error {
	var f fetchFunc
	procExists, err := fileExists(m.mod.ResolveHostFS("/proc/net/stat/nf_conntrack"))
	if err != nil {
		return fmt.Errorf("error checking for procfs: %w", err)
	}

	if procExists {
		m.Logger().Info("Using procfs to fetch conntrack metrics")
		f = m.fetchProcFSMetrics
	} else { // fallback to netlink
		m.Logger().Info("nf_conntrack kernel module not loaded, using netlink to fetch conntrack metrics")
		f = m.fetchNetlinkMetrics
	}

	m.fetchFunc = f
	return nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	conntrackStats, err := m.fetchFunc()
	if err != nil {
		return fmt.Errorf("error fetching conntrack stats: %w", err)
	}

	summedEvents := procfs.ConntrackStatEntry{}
	for i, conn := range conntrackStats {
		// Entries represents the total number of connections in the conntrack table,
		// but the value is reported once per CPU. Only add it from the first entry.
		if i == 0 {
			summedEvents.Entries = conn.Entries
		}
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

func fileExists(path string) (ok bool, err error) {
	if _, err = os.Stat(path); err == nil {
		ok = true
	} else if os.IsNotExist(err) {
		err = nil
	}
	return ok, err
}

func (m *MetricSet) fetchProcFSMetrics() ([]procfs.ConntrackStatEntry, error) {
	newFS, err := procfs.NewFS(m.mod.ResolveHostFS("/proc"))
	if err != nil {
		return nil, fmt.Errorf("error creating new Host FS at /proc: %w", err)
	}
	conntrackStats, err := newFS.ConntrackStat()
	if err != nil {
		if os.IsNotExist(err) {
			err = mb.PartialMetricsError{Err: fmt.Errorf("nf_conntrack kernel module not loaded: %w", err)}
		}
		return nil, err
	}
	return conntrackStats, nil
}

func (m *MetricSet) fetchNetlinkMetrics() ([]procfs.ConntrackStatEntry, error) {
	conn, err := conntrack.Dial(nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cpuStats, err := conn.Stats()
	if err != nil {
		return nil, err
	}

	stats := make([]procfs.ConntrackStatEntry, 0, len(cpuStats))

	for _, stat := range cpuStats {
		stats = append(stats, procfs.ConntrackStatEntry{
			Found:         uint64(stat.Found),
			Invalid:       uint64(stat.Invalid),
			Ignore:        uint64(stat.Ignore),
			InsertFailed:  uint64(stat.InsertFailed),
			Drop:          uint64(stat.Drop),
			EarlyDrop:     uint64(stat.EarlyDrop),
			SearchRestart: uint64(stat.SearchRestart),
		})
	}

	globalStats, err := conn.StatsGlobal()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch global stats: %w", err)
	}

	stats[0].Entries = uint64(globalStats.Entries)
	return stats, nil
}
