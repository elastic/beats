// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"fmt"
	"os"
	"runtime"
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
	namespace     = "system.audit.process"

	bucketName              = "auditbeat.process.v1"
	bucketKeyStateTimestamp = "state_timestamp"

	eventTypeState = "state"
	eventTypeEvent = "event"
	eventTypeError = "error"
)

type eventAction uint8

const (
	eventActionExistingProcess eventAction = iota
	eventActionProcessStarted
	eventActionProcessStopped
	eventActionProcessError
)

func (action eventAction) String() string {
	switch action {
	case eventActionExistingProcess:
		return "existing_process"
	case eventActionProcessStarted:
		return "process_started"
	case eventActionProcessStopped:
		return "process_stopped"
	case eventActionProcessError:
		return "process_error"
	default:
		return ""
	}
}

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithNamespace(namespace),
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

	suppressPermissionWarnings bool
}

// Process represents information about a process.
type Process struct {
	Info  types.ProcessInfo
	Error error
}

// Hash creates a hash for Process.
func (p Process) Hash() uint64 {
	h := xxhash.New64()
	h.WriteString(strconv.Itoa(p.Info.PID))
	h.WriteString(p.Info.StartTime.String())
	return h.Sum64()
}

func (p Process) toMapStr() common.MapStr {
	return common.MapStr{
		// https://github.com/elastic/ecs#-process-fields
		"name":              p.Info.Name,
		"args":              p.Info.Args,
		"pid":               p.Info.PID,
		"ppid":              p.Info.PPID,
		"working_directory": p.Info.CWD,
		"executable":        p.Info.Exe,
		"start":             p.Info.StartTime,
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

	if os.Geteuid() != 0 {
		ms.log.Warn("Running as non-root user, will likely not report all processes.")
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

	processes, err := ms.getProcesses()
	if err != nil {
		return errors.Wrap(err, "failed to get process infos")
	}
	ms.log.Debugf("Found %v processes", len(processes))

	stateID, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "error generating state ID")
	}
	for _, p := range processes {
		if p.Error == nil {
			event := processEvent(p, eventTypeState, eventActionExistingProcess)
			event.RootFields.Put("event.id", stateID.String())
			report.Event(event)
		} else {
			ms.log.Warn(p.Error)
			report.Event(processEvent(p, eventTypeError, eventActionProcessError))
		}
	}

	if ms.cache != nil {
		// This will initialize the cache with the current processes
		ms.cache.DiffAndUpdateCache(convertToCacheable(processes))
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
	processes, err := ms.getProcesses()
	if err != nil {
		return errors.Wrap(err, "failed to get processes")
	}
	ms.log.Debugf("Found %v processes", len(processes))

	started, stopped := ms.cache.DiffAndUpdateCache(convertToCacheable(processes))

	for _, cacheValue := range started {
		p := cacheValue.(*Process)

		if p.Error == nil {
			report.Event(processEvent(p, eventTypeEvent, eventActionProcessStarted))
		} else {
			ms.log.Warn(p.Error)
			report.Event(processEvent(p, eventTypeError, eventActionProcessError))
		}
	}

	for _, cacheValue := range stopped {
		p := cacheValue.(*Process)

		if p.Error == nil {
			report.Event(processEvent(p, eventTypeEvent, eventActionProcessStopped))
		}
	}

	return nil
}

func processEvent(process *Process, eventType string, action eventAction) mb.Event {
	event := mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"kind":   eventType,
				"action": action.String(),
			},
			"process": process.toMapStr(),
			"message": processMessage(process, action),
		},
	}

	if process.Error != nil {
		event.RootFields.Put("error.message", process.Error.Error())
	}

	return event
}

func processMessage(process *Process, action eventAction) string {
	if process.Error != nil {
		return fmt.Sprintf("ERROR for PID %d: %v", process.Info.PID, process.Error)
	}

	var actionString string
	switch action {
	case eventActionProcessStarted:
		actionString = "STARTED"
	case eventActionProcessStopped:
		actionString = "STOPPED"
	case eventActionExistingProcess:
		actionString = "is RUNNING"
	}

	return fmt.Sprintf("Process %v (PID: %d) %v",
		process.Info.Name, process.Info.PID, actionString)
}

func convertToCacheable(processes []*Process) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(processes))

	for _, p := range processes {
		c = append(c, p)
	}

	return c
}

func (ms *MetricSet) getProcesses() ([]*Process, error) {
	// TODO: Implement Processes() in go-sysinfo
	// e.g. https://github.com/elastic/go-sysinfo/blob/master/providers/darwin/process_darwin_amd64.go#L41
	pids, err := process.Pids()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch the list of PIDs")
	}

	var processes []*Process
	for _, pid := range pids {
		var process *Process

		sysinfoProc, err := sysinfo.Process(pid)
		if err != nil {
			if os.IsNotExist(err) {
				// Skip - process probably just terminated since our call
				// to Pids()
				continue
			}

			// Record what we can and continue
			process = &Process{
				Info: types.ProcessInfo{
					PID: pid,
				},
				Error: errors.Wrapf(err, "failed to load process with PID %d", pid),
			}
		} else {
			pInfo, err := sysinfoProc.Info()
			if err == nil {
				process = &Process{
					Info: pInfo,
				}
			} else {
				if os.IsNotExist(err) {
					// Skip - process probably just terminated since our call
					// to Pids()
					continue
				}

				if os.Geteuid() != 0 {
					if os.IsPermission(err) || runtime.GOOS == "darwin" {
						/*
							Running as non-root, permission issues when trying to access other user's private
							process information are expected.

							Unfortunately, for darwin os.IsPermission() does not
							work because it is a custom error created using errors.New() in
							getProcTaskAllInfo() in go-sysinfo/providers/darwin/process_darwin_amd64.go

							TODO: Fix go-sysinfo to have better error for darwin.
						*/
						if !ms.suppressPermissionWarnings {
							ms.log.Warnf("Failed to load process information for PID %d as non-root user. "+
								"Will suppress further errors of this kind. Error: %v", pid, err)

							// Only warn once at the start of Auditbeat.
							ms.suppressPermissionWarnings = true
						}

						//continue
					}
				}

				// Record what we can and continue
				process = &Process{
					Info:  pInfo,
					Error: errors.Wrapf(err, "failed to load process information for PID %d", pid),
				}
				process.Info.PID = pid // in case pInfo did not contain it
			}
		}

		processes = append(processes, process)
	}

	return processes, nil
}
