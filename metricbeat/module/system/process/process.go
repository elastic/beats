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

//go:build darwin || freebsd || linux || windows || aix

package process

import (
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/system/process/network"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/process"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

var debugf = logp.NewLogger("system.process").Debugf

func init() {
	mb.Registry.MustAddMetricSet("system", "process", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet that fetches process metrics.
type MetricSet struct {
	mb.BaseMetricSet
	stats  *process.Stats
	perCPU bool

	networkMonitoring *network.Tracker
}

// New creates and returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	sys := base.Module().(resolve.Resolver)
	enableCgroups := false
	if runtime.GOOS == "linux" {
		if config.Cgroups == nil || *config.Cgroups {
			enableCgroups = true
			debugf("process cgroup data collection is enabled, using hostfs='%v'", sys.ResolveHostFS(""))
		}
	}

	m := &MetricSet{
		BaseMetricSet: base,
		stats: &process.Stats{
			Procs:         config.Procs,
			Hostfs:        sys,
			EnvWhitelist:  config.EnvWhitelist,
			CPUTicks:      config.IncludeCPUTicks || (config.CPUTicks != nil && *config.CPUTicks),
			CacheCmdLine:  config.CacheCmdLine,
			IncludeTop:    config.IncludeTop,
			EnableCgroups: enableCgroups,
			CgroupOpts: cgroup.ReaderOptions{
				RootfsMountpoint:  sys,
				IgnoreRootCgroups: true,
			},
		},
		perCPU: config.IncludePerCPU,
	}

	// If hostfs is set, we may not want to force the hierarchy override, as the user could be expecting a custom path.
	if !sys.IsSet() {
		override, isset := os.LookupEnv("LIBBEAT_MONITORING_CGROUPS_HIERARCHY_OVERRIDE")
		if isset {
			m.stats.CgroupOpts.CgroupsHierarchyOverride = override
		}
	}

	// setup network tracking
	if config.TrackNetworkData {
		m.Logger().Infof("Starting system/process network monitoring")
		var err error
		m.networkMonitoring, err = network.NewNetworkTracker()
		if err != nil {
			return nil, fmt.Errorf("error creating network tracker: %w", err)
		}
		err = m.networkMonitoring.Track()
		if err != nil {
			return nil, fmt.Errorf("error starting network tracker: %w", err)
		}
	}

	err := m.stats.Init()
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Fetch fetches metrics for all processes. It iterates over each PID and
// collects process metadata, CPU metrics, and memory metrics.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	procs, roots, err := m.stats.Get()
	if err != nil {
		return fmt.Errorf("process stats: %w", err)
	}

	for evtI := range procs {
		if m.networkMonitoring != nil {
			pid, err := extractPID(roots[evtI])
			if err != nil {
				m.Logger().Debugf("error fetching pid for network data: %s", err)
			} else {
				netData := m.networkMonitoring.Get(pid)
				if netData.ContainsMetrics() {
					procs[evtI].Put("network.usage", mapstr.M{
						"inbound": mapstr.M{
							"tcp": netData.Incoming.TCP,
							"udp": netData.Incoming.UDP,
						},
						"outbound": mapstr.M{
							"tcp": netData.Outgoing.TCP,
							"udp": netData.Outgoing.UDP,
						},
					})
				}
			}
		}

		isOpen := r.Event(mb.Event{
			MetricSetFields: procs[evtI],
			RootFields:      roots[evtI],
		})
		if !isOpen {
			return nil
		}
	}

	return nil
}

func (m *MetricSet) Close() error {
	if m.networkMonitoring != nil {
		m.networkMonitoring.Stop()
	}
	return nil
}

func extractPID(rootMap mapstr.M) (int, error) {
	rawPid, err := rootMap.GetValue("process.pid")
	if err != nil {
		return 0, fmt.Errorf("error fetching root event PID for pid: %w", err)
	}

	if unwrappedPid, ok := rawPid.(int); ok {
		return unwrappedPid, nil
	}
	return 0, fmt.Errorf("could not unpack PID from root event, got %T", rawPid)

}
