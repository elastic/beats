// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processes

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/process"
	"github.com/elastic/go-sysinfo"
)

const (
	moduleName    = "system"
	metricsetName = "processes"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about the host.
type MetricSet struct {
	mb.BaseMetricSet
	log *logp.Logger
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	return &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger(moduleName),
	}, nil
}

// Fetch collects data about the host. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	// TODO: Implement Processes() in go-sysinfo
	// e.g. https://github.com/elastic/go-sysinfo/blob/master/providers/darwin/process_darwin_amd64.go#L41
	pids, err := process.Pids()
	if err != nil {
		ms.log.Errorw("Failed to fetch the list of PIDs", "error", err)
	}

	var processInfos []common.MapStr

	for _, pid := range pids {
		if p, err := sysinfo.Process(pid); err == nil {
			if pInfo, err := p.Info(); err == nil {
				processInfos = append(processInfos, common.MapStr{
					// https://github.com/elastic/ecs#-process-fields
					"process.args": pInfo.Args,
					"process.name": pInfo.Name,
					"process.pid":  pInfo.PID,
					"process.ppid": pInfo.PPID,

					"process.cwd":       pInfo.CWD,
					"process.exe":       pInfo.Exe,
					"process.starttime": pInfo.StartTime,
				})
			} else {
				ms.log.Errorw("Failed to load process information", "error", err)
			}
		} else {
			ms.log.Errorw("Failed to load process", "error", err)
		}
	}

	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"processes": processInfos,
		},
	})
}
