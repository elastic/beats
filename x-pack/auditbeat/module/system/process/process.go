// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"strconv"
	"time"

	"github.com/OneOfOne/xxhash"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/process"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

const (
	moduleName    = "system"
	metricsetName = "process"

	bucketName              = "auditbeat.process.v1"
	bucketKeyStateTimestamp = "state_timestamp"

	eventTypeState = "state"
	eventTypeEvent = "event"

	eventActionExistingProcess = "existing_process"
	eventActionProcessStarted  = "process_started"
	eventActionProcessStopped  = "process_stopped"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about the host.
type MetricSet struct {
	mb.BaseMetricSet
	config    Config
	cache     *cache.Cache
	log       *logp.Logger
	bucket    datastore.Bucket
	lastState time.Time
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
		"name":              pInfo.Name,
		"args":              pInfo.Args,
		"pid":               pInfo.PID,
		"ppid":              pInfo.PPID,
		"working_directory": pInfo.CWD,
		"executable":        pInfo.Exe,
		"start":             pInfo.StartTime,
	}
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open persistent datastore")
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(metricsetName),
		cache:         cache.New(),
		bucket:        bucket,
	}

	// Load from disk: Time when state was last sent
	err = bucket.Load(bucketKeyStateTimestamp, func(blob []byte) error {
		if len(blob) > 0 {
			return ms.lastState.UnmarshalBinary(blob)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !ms.lastState.IsZero() {
		ms.log.Debugf("Last state was sent at %v. Next state update by %v.", ms.lastState, ms.lastState.Add(ms.config.effectiveStatePeriod()))
	} else {
		ms.log.Debug("No state timestamp found")
	}

	return ms, nil
}

// Close cleans up the MetricSet when it finishes.
func (ms *MetricSet) Close() error {
	if ms.bucket != nil {
		return ms.bucket.Close()
	}
	return nil
}

// Fetch collects process information. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	needsStateUpdate := time.Since(ms.lastState) > ms.config.effectiveStatePeriod()
	if needsStateUpdate || ms.cache.IsEmpty() {
		ms.log.Debugf("State update needed (needsStateUpdate=%v, cache.IsEmpty()=%v)", needsStateUpdate, ms.cache.IsEmpty())
		err := ms.reportState(report)
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
		}
		ms.log.Debugf("Next state update by %v", ms.lastState.Add(ms.config.effectiveStatePeriod()))
	}

	err := ms.reportChanges(report)
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
	}
}

// reportState reports all running processes on the system.
func (ms *MetricSet) reportState(report mb.ReporterV2) error {
	// Only update lastState if this state update was regularly scheduled,
	// i.e. not caused by an Auditbeat restart (when the cache would be empty).
	if !ms.cache.IsEmpty() {
		ms.lastState = time.Now()
	}

	processInfos, err := ms.getProcessInfos()
	if err != nil {
		return errors.Wrap(err, "failed to get process infos")
	}
	ms.log.Debugf("Found %v processes", len(processInfos))

	stateID, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "error generating state ID")
	}
	for _, pInfo := range processInfos {
		event := processEvent(pInfo, eventTypeState, eventActionExistingProcess)
		event.RootFields.Put("event.id", stateID.String())
		report.Event(event)
	}

	if ms.cache != nil {
		// This will initialize the cache with the current processes
		ms.cache.DiffAndUpdateCache(convertToCacheable(processInfos))
	}

	// Save time so we know when to send the state again (config.StatePeriod)
	timeBytes, err := ms.lastState.MarshalBinary()
	if err != nil {
		return err
	}
	err = ms.bucket.Store(bucketKeyStateTimestamp, timeBytes)
	if err != nil {
		return errors.Wrap(err, "error writing state timestamp to disk")
	}

	return nil
}

// reportChanges detects and reports any changes to processes on this system since the last call.
func (ms *MetricSet) reportChanges(report mb.ReporterV2) error {
	processInfos, err := ms.getProcessInfos()
	if err != nil {
		return errors.Wrap(err, "failed to get process infos")
	}
	ms.log.Debugf("Found %v processes", len(processInfos))

	started, stopped := ms.cache.DiffAndUpdateCache(convertToCacheable(processInfos))

	for _, pInfo := range started {
		report.Event(processEvent(pInfo.(*ProcessInfo), eventTypeEvent, eventActionProcessStarted))
	}

	for _, pInfo := range stopped {
		report.Event(processEvent(pInfo.(*ProcessInfo), eventTypeEvent, eventActionProcessStopped))
	}

	return nil
}

func processEvent(pInfo *ProcessInfo, eventType string, eventAction string) mb.Event {
	return mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"kind":   eventType,
				"action": eventAction,
			},
			"process": pInfo.toMapStr(),
		},
	}
}

func convertToCacheable(processInfos []*ProcessInfo) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(processInfos))

	for _, p := range processInfos {
		c = append(c, p)
	}

	return c
}

func (ms *MetricSet) getProcessInfos() ([]*ProcessInfo, error) {
	// TODO: Implement Processes() in go-sysinfo
	// e.g. https://github.com/elastic/go-sysinfo/blob/master/providers/darwin/process_darwin_amd64.go#L41
	pids, err := process.Pids()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch the list of PIDs")
	}

	var processInfos []*ProcessInfo

	for _, pid := range pids {
		process, err := sysinfo.Process(pid)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load process")
		}

		pInfo, err := process.Info()
		if err != nil {
			return nil, errors.Wrap(err, "failed to load process information")
		}

		processInfos = append(processInfos, &ProcessInfo{pInfo})
	}

	return processInfos, nil
}
