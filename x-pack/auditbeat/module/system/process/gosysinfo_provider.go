// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/auditbeat/datastore"
	"github.com/elastic/beats/v7/auditbeat/helper/hasher"
	"github.com/elastic/beats/v7/libbeat/common/capabilities"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/cache"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

const (
	bucketName              = "auditbeat.process.v1"
	bucketKeyStateTimestamp = "state_timestamp"
)

// SysinfoMetricSet collects data about the host.
type SysInfoMetricSet struct {
	MetricSet
	hasher    *hasher.FileHasher
	cache     *cache.Cache
	bucket    datastore.Bucket
	lastState time.Time

	suppressPermissionWarnings bool
}

// Process represents information about a process.
type Process struct {
	Info         types.ProcessInfo
	UserInfo     *types.UserInfo
	User         *user.User
	Group        *user.Group
	CapEffective []string
	CapPermitted []string
	Hashes       map[hasher.HashType]hasher.Digest
	Error        error
}

// Hash creates a hash for Process.
func (p Process) Hash() uint64 {
	h := xxhash.New()
	//nolint:errcheck // always return nil err
	h.WriteString(strconv.Itoa(p.Info.PID))
	//nolint:errcheck // always return nil err
	h.WriteString(p.Info.StartTime.String())
	return h.Sum64()
}

func (p Process) toMapStr() mapstr.M {
	return mapstr.M{
		// https://github.com/elastic/ecs#-process-fields
		"name": p.Info.Name,
		"args": p.Info.Args,
		"pid":  p.Info.PID,
		"parent": mapstr.M{
			"pid": p.Info.PPID,
		},
		"working_directory": p.Info.CWD,
		"executable":        p.Info.Exe,
		"start":             p.Info.StartTime,
	}
}

// NewFromSysInfo constructs a new MetricSet backed by go-sysinfo.
func NewFromSysInfo(ms MetricSet) (mb.MetricSet, error) {
	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to open persistent datastore: %w", err)
	}

	// Load from disk: Time when state was last sent
	var lastState time.Time
	err = bucket.Load(bucketKeyStateTimestamp, func(blob []byte) error {
		if len(blob) > 0 {
			return lastState.UnmarshalBinary(blob)
		}
		return nil
	})
	if err != nil {
		bucket.Close()
		return nil, err
	}
	if !lastState.IsZero() {
		ms.log.Debugf("Last state was sent at %v. Next state update by %v.",
			lastState, lastState.Add(ms.config.effectiveStatePeriod()))
	} else {
		ms.log.Debug("No state timestamp found")
	}

	hasher, err := hasher.NewFileHasher(ms.config.HasherConfig, nil)
	if err != nil {
		bucket.Close()
		return nil, err
	}

	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		ms.log.Warn("Running as non-root user, will likely not report all processes.")
	}

	sm := &SysInfoMetricSet{
		MetricSet: ms,
		cache:     cache.New(),
		bucket:    bucket,
		lastState: lastState,
		hasher:    hasher,
	}

	return sm, nil
}

// Close cleans up the MetricSet when it finishes.
func (ms *SysInfoMetricSet) Close() error {
	if ms.bucket != nil {
		return ms.bucket.Close()
	}
	return nil
}

// Fetch collects process information. It is invoked periodically.
func (ms *SysInfoMetricSet) Fetch(report mb.ReporterV2) {
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
func (ms *SysInfoMetricSet) reportState(report mb.ReporterV2) error {
	// Only update lastState if this state update was regularly scheduled,
	// i.e. not caused by an Auditbeat restart (when the cache would be empty).
	if !ms.cache.IsEmpty() {
		ms.lastState = time.Now()
	}

	processes, err := ms.getProcesses()
	if err != nil {
		return fmt.Errorf("failed to get process infos: %w", err)
	}
	ms.log.Debugf("Found %v processes", len(processes))

	stateID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("error generating state ID: %w", err)
	}
	for _, p := range processes {
		ms.enrichProcess(p)

		if p.Error == nil {
			event := ms.processEvent(p, eventTypeState, eventActionExistingProcess)
			event.RootFields.Put("event.id", stateID.String())
			report.Event(event)
		} else {
			ms.log.Warn(p.Error)
			report.Event(ms.processEvent(p, eventTypeEvent, eventActionProcessError))
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
		return fmt.Errorf("error writing state timestamp to disk: %w", err)
	}

	return nil
}

// reportChanges detects and reports any changes to processes on this system since the last call.
func (ms *SysInfoMetricSet) reportChanges(report mb.ReporterV2) error {
	processes, err := ms.getProcesses()
	if err != nil {
		return fmt.Errorf("failed to get processes: %w", err)
	}
	ms.log.Debugf("Found %v processes", len(processes))

	started, stopped := ms.cache.DiffAndUpdateCache(convertToCacheable(processes))

	for _, cacheValue := range started {
		p, ok := cacheValue.(*Process)
		if !ok {
			return fmt.Errorf("cache type error")
		}
		ms.enrichProcess(p)

		if p.Error == nil {
			report.Event(ms.processEvent(p, eventTypeEvent, eventActionProcessStarted))
		} else {
			ms.log.Warn(p.Error)
			report.Event(ms.processEvent(p, eventTypeEvent, eventActionProcessError))
		}
	}

	for _, cacheValue := range stopped {
		p, ok := cacheValue.(*Process)
		if !ok {
			return fmt.Errorf("cache type error")
		}

		if p.Error == nil {
			report.Event(ms.processEvent(p, eventTypeEvent, eventActionProcessStopped))
		}
	}

	return nil
}

// enrichProcess enriches a process with user lookup information
// and executable file hash.
func (ms *SysInfoMetricSet) enrichProcess(process *Process) {
	if process.UserInfo != nil {
		goUser, err := user.LookupId(process.UserInfo.UID)
		if err == nil {
			process.User = goUser
		}

		group, err := user.LookupGroupId(process.UserInfo.GID)
		if err == nil {
			process.Group = group
		}
	}

	if process.Info.Exe != "" {
		sharedMntNS, err := isNsSharedWith(process.Info.PID, "mnt")
		if err != nil {
			if process.Error == nil {
				process.Error = fmt.Errorf("failed to get namespaces for %v PID %v: %w", process.Info.Exe, process.Info.PID, err)
			}
			return
		}
		if !sharedMntNS {
			return
		}
		hashes, err := ms.hasher.HashFile(process.Info.Exe)
		if err != nil {
			if process.Error == nil {
				process.Error = fmt.Errorf("failed to hash executable %v for PID %v: %w", process.Info.Exe, process.Info.PID, err)
			}
			return
		}
		process.Hashes = hashes
	}
}

func (ms *SysInfoMetricSet) processEvent(process *Process, eventType string, action eventAction) mb.Event {
	event := mb.Event{
		RootFields: mapstr.M{
			"event": mapstr.M{
				"kind":     eventType,
				"category": []string{"process"},
				"type":     []string{action.Type()},
				"action":   action.String(),
			},
			"process": process.toMapStr(),
			"message": processMessage(process, action),
		},
	}

	if process.UserInfo != nil {
		putIfNotEmpty(&event.RootFields, "user.id", process.UserInfo.UID)
		putIfNotEmpty(&event.RootFields, "user.group.id", process.UserInfo.GID)

		putIfNotEmpty(&event.RootFields, "user.effective.id", process.UserInfo.EUID)
		putIfNotEmpty(&event.RootFields, "user.effective.group.id", process.UserInfo.EGID)

		putIfNotEmpty(&event.RootFields, "user.saved.id", process.UserInfo.SUID)
		putIfNotEmpty(&event.RootFields, "user.saved.group.id", process.UserInfo.SGID)
	}

	if process.User != nil {
		if process.User.Username != "" {
			event.RootFields.Put("user.name", process.User.Username)
		} else if process.User.Name != "" {
			event.RootFields.Put("user.name", process.User.Name)
		}
	}

	if process.Group != nil {
		event.RootFields.Put("user.group.name", process.Group.Name)
	}

	if len(process.CapEffective) > 0 {
		event.RootFields.Put("process.thread.capabilities.effective", process.CapEffective)
	}
	if len(process.CapPermitted) > 0 {
		event.RootFields.Put("process.thread.capabilities.permitted", process.CapPermitted)
	}

	if process.Hashes != nil {
		for hashType, digest := range process.Hashes {
			fieldName := "process.hash." + string(hashType)
			event.RootFields.Put(fieldName, digest)
		}
	}

	if process.Error != nil {
		event.RootFields.Put("error.message", process.Error.Error())
	}

	if ms.HostID() != "" {
		event.RootFields.Put("process.entity_id",
			entityID(ms.HostID(), process.Info.PID, process.Info.StartTime))
	}

	return event
}

func putIfNotEmpty(mapstr *mapstr.M, key string, value string) {
	if value != "" {
		mapstr.Put(key, value)
	}
}

func processMessage(process *Process, action eventAction) string {
	var username string
	if process.User != nil {
		username = process.User.Username
	}

	return makeMessage(process.Info.PID, action, process.Info.Name, username, process.Error)
}

func convertToCacheable(processes []*Process) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(processes))

	for _, p := range processes {
		c = append(c, p)
	}

	return c
}

func (ms *SysInfoMetricSet) getProcesses() ([]*Process, error) {
	sysinfoProcs, err := sysinfo.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch processes: %w", err)
	}

	processes := make([]*Process, 0, len(sysinfoProcs))
	for _, sysinfoProc := range sysinfoProcs {
		var process *Process

		pInfo, err := sysinfoProc.Info()
		if err != nil {
			if os.IsNotExist(err) {
				// Skip - process probably just terminated since our call to Processes().
				continue
			}

			if os.Geteuid() != 0 && os.IsPermission(err) {
				// Running as non-root, permission issues when trying to access
				// other user's private process information are expected.

				if !ms.suppressPermissionWarnings {
					ms.log.Warnf("Failed to load process information for PID %d as non-root user. "+
						"Will suppress further errors of this kind. Error: %v", sysinfoProc.PID(), err)

					// Only warn once at the start of Auditbeat.
					ms.suppressPermissionWarnings = true
				}

				continue
			}

			// Record what we can and continue
			process = &Process{
				Info:  pInfo,
				Error: fmt.Errorf("failed to load process information for PID %d: %w", sysinfoProc.PID(), err),
			}
			process.Info.PID = sysinfoProc.PID() // in case pInfo did not contain it
		} else {
			process = &Process{
				Info: pInfo,
			}
		}

		userInfo, err := sysinfoProc.User()
		if err != nil {
			if process.Error == nil {
				process.Error = fmt.Errorf("failed to load user for PID %d: %w", sysinfoProc.PID(), err)
			}
		} else {
			process.UserInfo = &userInfo
		}

		// Exclude Linux kernel processes, they are not very interesting.
		if runtime.GOOS == "linux" {
			if userInfo.UID == "0" && process.Info.Exe == "" {
				continue
			}

			// Fetch Effective and Permitted capabilities
			process.CapEffective, err = capabilities.FromPid(capabilities.Effective, pInfo.PID)
			if err != nil && process.Error == nil {
				process.Error = err
			}
			process.CapPermitted, err = capabilities.FromPid(capabilities.Permitted, pInfo.PID)
			if err != nil && process.Error == nil {
				process.Error = err
			}
		}

		processes = append(processes, process)
	}

	return processes, nil
}
