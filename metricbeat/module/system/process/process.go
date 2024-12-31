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
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/logp"
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
	stats            *process.Stats
	perCPU           bool
	setpid           int
	degradeOnPartial bool
}

// New creates and returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	sys, ok := base.Module().(resolve.Resolver)
	if !ok {
		return nil, fmt.Errorf("resolver cannot be cast from the module")
	}
	enableCgroups := false
	if runtime.GOOS == "linux" {
		if config.Cgroups == nil || *config.Cgroups {
			enableCgroups = true
			debugf("process cgroup data collection is enabled, using hostfs='%v'", sys.ResolveHostFS(""))
		}
	}

	if config.Pid != 0 && config.Procs[0] != ".*" {
		logp.L().Warnf("`process.pid` set to %d, but `processes` is set to a non-default value. Metricset will only report metrics for pid %d", config.Pid, config.Pid)
	}
	degradedConf := struct {
		DegradeOnPartial bool `config:"degrade_on_partial"`
	}{}
	if err := base.Module().UnpackConfig(&degradedConf); err != nil {
		logp.L().Warnf("Failed to unpack config; degraded mode will be disabled for partial metrics: %v", err)
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
		perCPU:           config.IncludePerCPU,
		degradeOnPartial: degradedConf.DegradeOnPartial,
	}

	m.setpid = config.Pid

	// If hostfs is set, we may not want to force the hierarchy override, as the user could be expecting a custom path.
	if !sys.IsSet() {
		override, isset := os.LookupEnv("LIBBEAT_MONITORING_CGROUPS_HIERARCHY_OVERRIDE")
		if isset {
			m.stats.CgroupOpts.CgroupsHierarchyOverride = override
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

	// monitor either a single PID, or the configured set of processes.
	if m.setpid == 0 {
		procs, roots, err := m.stats.Get()
		if err != nil && !errors.Is(err, process.NonFatalErr{}) {
			// return only if the error is fatal in nature
			return fmt.Errorf("process stats: %w", err)
		} else if (err != nil && errors.Is(err, process.NonFatalErr{})) {
			if m.degradeOnPartial {
				return fmt.Errorf("error fetching process list: %w", err)
			}
			err = mb.PartialMetricsError{Err: err}
		}

		for evtI := range procs {
			isOpen := r.Event(mb.Event{
				MetricSetFields: procs[evtI],
				RootFields:      roots[evtI],
			})
			if !isOpen {
				return err
			}
		}
		return err
	} else {
		proc, root, err := m.stats.GetOneRootEvent(m.setpid)
		if err != nil && !errors.Is(err, process.NonFatalErr{}) {
			// return only if the error is fatal in nature
			return fmt.Errorf("error fetching pid %d: %w", m.setpid, err)
		} else if (err != nil && errors.Is(err, process.NonFatalErr{})) {
			if m.degradeOnPartial {
				return fmt.Errorf("error fetching process list: %w", err)
			}
			err = mb.PartialMetricsError{Err: err}
		}
		// if error is non-fatal, emit partial metrics.
		r.Event(mb.Event{
			MetricSetFields: proc,
			RootFields:      root,
		})
		return err
	}
}
