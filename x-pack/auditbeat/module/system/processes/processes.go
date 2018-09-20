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
	cache map[string](*types.ProcessInfo)
	log   *logp.Logger
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

// fastHash returns a hash calculated using FNV-1 of the PID and StartTime.
// Based on https://github.com/elastic/gosigar/blob/master/sys/linux/inetdiag.go#L362
// TODO: Move to go-sysinfo
// Actually, might not need this after all. Delete?
/*func fastHashProcessInfo(pInfo *types.ProcessInfo) uint64 {
	h := fnv.New64()
	h.Write([]byte(strconv.Itoa(pInfo.PID)))
	h.Write([]byte(pInfo.StartTime.String()))
	return h.Sum64()
}*/

func processInfoNaiveHash(pInfo *types.ProcessInfo) string {
	// Could use real hash e.g. FNV if there is an advantage
	return strconv.Itoa(pInfo.PID) + pInfo.StartTime.String()
}

func (ms *MetricSet) diffCache(current []*types.ProcessInfo) (new, missing []*types.ProcessInfo) {
	// Check for new - what is in current but not in cache
	for _, pInfo := range current {
		if _, inCache := ms.cache[processInfoNaiveHash(pInfo)]; !inCache {
			new = append(new, pInfo)
		}
	}

	// Check for missing - what is no longer in current that was in the cache
	for cachedPInfoKey, cachedPInfo := range ms.cache {
		found := false
		for _, currentPInfo := range current {
			if processInfoNaiveHash(currentPInfo) == cachedPInfoKey {
				found = true
				break
			}
		}

		if !found {
			missing = append(missing, cachedPInfo)
		}
	}

	return
}

func processInfoToMapStr(pInfo *types.ProcessInfo) common.MapStr {
	return common.MapStr{
		// https://github.com/elastic/ecs#-process-fields
		"process.args": pInfo.Args,
		"process.name": pInfo.Name,
		"process.pid":  pInfo.PID,
		"process.ppid": pInfo.PPID,

		"process.cwd":       pInfo.CWD,
		"process.exe":       pInfo.Exe,
		"process.starttime": pInfo.StartTime,
	}
}

// Fetch collects data about the host. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	processInfos := ms.getProcessInfos()

	diff := true
	if ms.cache != nil && diff {
		// find out which processes were stopped or started, if any
		started, stopped := ms.diffCache(processInfos)

		for _, pInfo := range started {
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"status":  "started",
					"process": processInfoToMapStr(pInfo),
				},
			})
		}

		for _, pInfo := range stopped {
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"status":  "stopped",
					"process": processInfoToMapStr(pInfo),
				},
			})
		}
	} else {
		var processEvents []common.MapStr

		for _, pInfo := range processInfos {
			processEvents = append(processEvents, processInfoToMapStr(pInfo))
		}

		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"processes": processEvents,
			},
		})
	}

	if diff {
		// Refill cache
		ms.cache = make(map[string](*types.ProcessInfo))
		for _, pInfo := range processInfos {
			ms.cache[processInfoNaiveHash(pInfo)] = pInfo
		}
	}
}

func (ms *MetricSet) getProcessInfos() []*types.ProcessInfo {
	// TODO: Implement Processes() in go-sysinfo
	// e.g. https://github.com/elastic/go-sysinfo/blob/master/providers/darwin/process_darwin_amd64.go#L41
	pids, err := process.Pids()
	if err != nil {
		ms.log.Errorw("Failed to fetch the list of PIDs", "error", err)
	}

	var processInfos []*types.ProcessInfo

	for _, pid := range pids {
		if p, err := sysinfo.Process(pid); err == nil {
			if pInfo, err := p.Info(); err == nil {
				processInfos = append(processInfos, &pInfo)
			} else {
				ms.log.Errorw("Failed to load process information", "error", err)
			}
		} else {
			ms.log.Errorw("Failed to load process", "error", err)
		}
	}

	return processInfos
}
