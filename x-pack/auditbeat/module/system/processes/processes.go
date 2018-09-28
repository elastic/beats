// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processes

import (
	"strconv"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
	"github.com/elastic/go-sysinfo/types"

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
	config Config
	cache  *cache.Cache
	log    *logp.Logger
}

// ProcessInfo wraps the process information and implements cache.Cacheable.
type ProcessInfo struct {
	types.ProcessInfo
}

// Hash creates a hash for ProcessInfo.
func (pInfo ProcessInfo) Hash() string {
	// Could use real hash e.g. FNV if there is an advantage
	return strconv.Itoa(pInfo.PID) + pInfo.StartTime.String()
}

func (pInfo ProcessInfo) toMapStr() common.MapStr {
	return common.MapStr{
		// https://github.com/elastic/ecs#-process-fields
		"process.name": pInfo.Name,
		"process.args": pInfo.Args,
		"process.pid":  pInfo.PID,
		"process.ppid": pInfo.PPID,

		"process.cwd":       pInfo.CWD,
		"process.exe":       pInfo.Exe,
		"process.starttime": pInfo.StartTime,
	}
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(moduleName),
	}

	if config.ReportChanges {
		ms.cache = cache.New()
	}

	return ms, nil
}

// Fetch checks which processes are running on the host and reports them.
// It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	processInfos, errorList := ms.getProcessInfos()
	if len(errorList) != 0 {
		for _, err := range errorList {
			ms.log.Error(err)
			report.Error(err)
		}
	}
	if processInfos == nil {
		return
	}

	if ms.cache != nil && !ms.cache.IsEmpty() {
		started, stopped := ms.cache.DiffAndUpdateCache(processInfos)

		for _, pInfo := range started {
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"status":    "started",
					"processes": pInfo.(ProcessInfo).toMapStr(),
				},
			})
		}

		for _, pInfo := range stopped {
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"status":    "stopped",
					"processes": pInfo.(ProcessInfo).toMapStr(),
				},
			})
		}
	} else {
		// Report all running processes
		var processEvents []common.MapStr

		for _, pInfo := range processInfos {
			processEvents = append(processEvents, pInfo.(ProcessInfo).toMapStr())
		}

		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"processes": processEvents,
			},
		})

		if ms.cache != nil {
			// This will initialize the cache with the current processes
			ms.cache.DiffAndUpdateCache(processInfos)
		}
	}
}

func (ms *MetricSet) getProcessInfos() ([]cache.Cacheable, []error) {
	// TODO: Implement Processes() in go-sysinfo
	// e.g. https://github.com/elastic/go-sysinfo/blob/master/providers/darwin/process_darwin_amd64.go#L41
	pids, err := process.Pids()
	if err != nil {
		return nil, []error{errors.Wrap(err, "Failed to fetch the list of PIDs")}
	}

	var processInfos []cache.Cacheable
	var errorList []error

	for _, pid := range pids {
		if p, err := sysinfo.Process(pid); err == nil {
			if pInfo, err := p.Info(); err == nil {
				processInfos = append(processInfos, ProcessInfo{pInfo})
			} else {
				errorList = append(errorList, errors.Wrap(err, "Failed to load process information"))
			}
		} else {
			errorList = append(errorList, errors.Wrap(err, "Failed to load process"))
		}
	}

	return processInfos, errorList
}
