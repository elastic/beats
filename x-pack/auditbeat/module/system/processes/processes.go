// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processes

import (
	"strconv"

	"github.com/OneOfOne/xxhash"
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
func (pInfo ProcessInfo) Hash() uint64 {
	h := xxhash.New64()
	h.WriteString(strconv.Itoa(pInfo.PID))
	h.WriteString(pInfo.StartTime.String())
	return h.Sum64()
}

func (pInfo ProcessInfo) toMapStr() common.MapStr {
	return common.MapStr{
		// https://github.com/elastic/ecs#-process-fields
		"name":      pInfo.Name,
		"args":      pInfo.Args,
		"pid":       pInfo.PID,
		"ppid":      pInfo.PPID,
		"cwd":       pInfo.CWD,
		"exe":       pInfo.Exe,
		"starttime": pInfo.StartTime,
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
		started, stopped := ms.cache.DiffAndUpdateCache(convertToCacheable(processInfos))

		for _, pInfo := range started {
			pInfoMapStr := pInfo.(*ProcessInfo).toMapStr()
			pInfoMapStr.Put("status", "started")

			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"process": pInfoMapStr,
				},
			})
		}

		for _, pInfo := range stopped {
			pInfoMapStr := pInfo.(*ProcessInfo).toMapStr()
			pInfoMapStr.Put("status", "stopped")

			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"process": pInfoMapStr,
				},
			})
		}
	} else {
		// Report all running processes
		var processEvents []common.MapStr

		for _, pInfo := range processInfos {
			pInfoMapStr := pInfo.toMapStr()
			pInfoMapStr.Put("status", "running")

			processEvents = append(processEvents, pInfoMapStr)
		}

		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"process": processEvents,
			},
		})

		if ms.cache != nil {
			// This will initialize the cache with the current processes
			ms.cache.DiffAndUpdateCache(convertToCacheable(processInfos))
		}
	}
}

func convertToCacheable(processInfos []*ProcessInfo) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(processInfos))

	for _, p := range processInfos {
		c = append(c, p)
	}

	return c
}

func (ms *MetricSet) getProcessInfos() ([]*ProcessInfo, []error) {
	// TODO: Implement Processes() in go-sysinfo
	// e.g. https://github.com/elastic/go-sysinfo/blob/master/providers/darwin/process_darwin_amd64.go#L41
	pids, err := process.Pids()
	if err != nil {
		return nil, []error{errors.Wrap(err, "Failed to fetch the list of PIDs")}
	}

	var processInfos []*ProcessInfo
	var errorList []error

	for _, pid := range pids {
		if p, err := sysinfo.Process(pid); err == nil {
			if pInfo, err := p.Info(); err == nil {
				processInfos = append(processInfos, &ProcessInfo{pInfo})
			} else {
				errorList = append(errorList, errors.Wrap(err, "Failed to load process information"))
			}
		} else {
			errorList = append(errorList, errors.Wrap(err, "Failed to load process"))
		}
	}

	return processInfos, errorList
}
