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

// +build darwin freebsd linux windows

package process

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/process"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/system"
	"github.com/elastic/gosigar/cgroup"
)

var debugf = logp.MakeDebug("system.process")

func init() {
	mb.Registry.MustAddMetricSet("system", "process", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet that fetches process metrics.
type MetricSet struct {
	mb.BaseMetricSet
	stats   *process.Stats
	cgroup  *cgroup.Reader
	perCPU  bool
	IsAgent bool
}

// New creates and returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	systemModule, ok := base.Module().(*system.Module)
	if !ok {
		return nil, fmt.Errorf("unexpected module type")
	}

	enableCgroups := false
	if runtime.GOOS == "linux" {
		if config.Cgroups == nil || *config.Cgroups {
			enableCgroups = true
			debugf("process cgroup data collection is enabled, using hostfs='%v'", paths.Paths.Hostfs)
		}
	}

	m := &MetricSet{
		BaseMetricSet: base,
		stats: &process.Stats{
			Procs:         config.Procs,
			EnvWhitelist:  config.EnvWhitelist,
			CPUTicks:      config.IncludeCPUTicks || (config.CPUTicks != nil && *config.CPUTicks),
			CacheCmdLine:  config.CacheCmdLine,
			IncludeTop:    config.IncludeTop,
			EnableCgroups: enableCgroups,
			CgroupOpts: cgroup.ReaderOptions{
				RootfsMountpoint:  paths.Paths.Hostfs,
				IgnoreRootCgroups: true,
			},
		},
		perCPU:  config.IncludePerCPU,
		IsAgent: systemModule.IsAgent,
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
	procs, err := m.stats.Get()
	if err != nil {
		return errors.Wrap(err, "process stats")
	}

	for _, proc := range procs {
		rootFields := common.MapStr{
			"process": common.MapStr{
				"name": getAndRemove(proc, "name"),
				"pid":  getAndRemove(proc, "pid"),
				"ppid": getAndRemove(proc, "ppid"),
				"pgid": getAndRemove(proc, "pgid"),
			},
			"user": common.MapStr{
				"name": getAndRemove(proc, "username"),
			},
		}

		if m.stats.EnableCgroups && !m.perCPU {
			proc.Delete("cgroup.cpuacct.percpu")
		}

		// Duplicate system.process.cmdline with ECS name process.command_line
		rootFields = getAndCopy(proc, "cmdline", rootFields, "process.command_line")

		// Duplicate system.process.state with process.state
		rootFields = getAndCopy(proc, "state", rootFields, "process.state")

		// Duplicate system.process.cpu.start_time with process.cpu.start_time
		rootFields = getAndCopy(proc, "cpu.start_time", rootFields, "process.cpu.start_time")

		// Duplicate system.process.cpu.total.norm.pct with process.cpu.pct
		rootFields = getAndCopy(proc, "cpu.total.norm.pct", rootFields, "process.cpu.pct")

		// Duplicate system.process.memory.rss.pct with process.memory.pct
		rootFields = getAndCopy(proc, "memory.rss.pct", rootFields, "process.memory.pct")

		if cwd := getAndRemove(proc, "cwd"); cwd != nil {
			rootFields.Put("process.working_directory", cwd)
		}

		if exe := getAndRemove(proc, "exe"); exe != nil {
			rootFields.Put("process.executable", exe)
		}

		if args := getAndRemove(proc, "args"); args != nil {
			rootFields.Put("process.args", args)
		}

		// "share" is unavailable on Windows.
		if runtime.GOOS == "windows" {
			proc.Delete("memory.share")
		}

		e := mb.Event{
			RootFields:      rootFields,
			MetricSetFields: proc,
		}
		isOpen := r.Event(e)
		if !isOpen {
			return nil
		}
	}

	return nil
}

func getAndRemove(from common.MapStr, field string) interface{} {
	if v, ok := from[field]; ok {
		delete(from, field)
		return v
	}
	return nil
}

func getAndCopy(from common.MapStr, field string, to common.MapStr, toField string) common.MapStr {
	v, err := from.GetValue(field)
	if err != nil {
		return to
	}

	_, err = to.Put(toField, v)
	return to
}
