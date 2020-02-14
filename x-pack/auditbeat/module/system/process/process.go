// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/auditbeat/datastore"
	"github.com/elastic/beats/v7/auditbeat/helper/hasher"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/cache"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
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
	system.SystemMetricSet
	config    Config
	cache     *cache.Cache
	log       *logp.Logger
	bucket    datastore.Bucket
	lastState time.Time
	hasher    *hasher.FileHasher

	suppressPermissionWarnings bool
}

// Process represents information about a process.
type Process struct {
	Info     types.ProcessInfo
	UserInfo *types.UserInfo
	User     *user.User
	Group    *user.Group
	Hashes   map[hasher.HashType]hasher.Digest
	Error    error
}

// Hash creates a hash for Process.
func (p Process) Hash() uint64 {
	h := xxhash.New()
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

// entityID creates an ID that uniquely identifies this process across machines.
func (p Process) entityID(hostID string) string {
	h := system.NewEntityHash()
	h.Write([]byte(hostID))
	binary.Write(h, binary.LittleEndian, int64(p.Info.PID))
	binary.Write(h, binary.LittleEndian, int64(p.Info.StartTime.Nanosecond()))
	return h.Sum()
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The %v/%v dataset is beta", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open persistent datastore")
	}

	hasher, err := hasher.NewFileHasher(config.HasherConfig, nil)
	if err != nil {
		return nil, err
	}

	ms := &MetricSet{
		SystemMetricSet: system.NewSystemMetricSet(base),
		config:          config,
		log:             logp.NewLogger(metricsetName),
		cache:           cache.New(),
		bucket:          bucket,
		hasher:          hasher,
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

	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
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
		ms.enrichProcess(p)

		if p.Error == nil {
			event := ms.processEvent(p, eventTypeState, eventActionExistingProcess)
			event.RootFields.Put("event.id", stateID.String())
			report.Event(event)
		} else {
			ms.log.Warn(p.Error)
			report.Event(ms.processEvent(p, eventTypeError, eventActionProcessError))
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
		ms.enrichProcess(p)

		if p.Error == nil {
			report.Event(ms.processEvent(p, eventTypeEvent, eventActionProcessStarted))
		} else {
			ms.log.Warn(p.Error)
			report.Event(ms.processEvent(p, eventTypeError, eventActionProcessError))
		}
	}

	for _, cacheValue := range stopped {
		p := cacheValue.(*Process)

		if p.Error == nil {
			report.Event(ms.processEvent(p, eventTypeEvent, eventActionProcessStopped))
		}
	}

	return nil
}

// enrichProcess enriches a process with user lookup information
// and executable file hash.
func (ms *MetricSet) enrichProcess(process *Process) {
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
		hashes, err := ms.hasher.HashFile(process.Info.Exe)
		if err != nil {
			if process.Error == nil {
				process.Error = errors.Wrapf(err, "failed to hash executable %v for PID %v", process.Info.Exe,
					process.Info.PID)
			}
		} else {
			process.Hashes = hashes
		}
	}
}

func (ms *MetricSet) processEvent(process *Process, eventType string, action eventAction) mb.Event {
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
		event.RootFields.Put("process.entity_id", process.entityID(ms.HostID()))
	}

	return event
}

func putIfNotEmpty(mapstr *common.MapStr, key string, value string) {
	if value != "" {
		mapstr.Put(key, value)
	}
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

	var userString string
	if process.User != nil {
		userString = fmt.Sprintf(" by user %v", process.User.Username)
	}

	return fmt.Sprintf("Process %v (PID: %d)%v %v",
		process.Info.Name, process.Info.PID, userString, actionString)
}

func convertToCacheable(processes []*Process) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(processes))

	for _, p := range processes {
		c = append(c, p)
	}

	return c
}

func (ms *MetricSet) getProcesses() ([]*Process, error) {
	var processes []*Process

	sysinfoProcs, err := sysinfo.Processes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch processes")
	}

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
				Error: errors.Wrapf(err, "failed to load process information for PID %d", sysinfoProc.PID()),
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
				process.Error = errors.Wrapf(err, "failed to load user for PID %d", sysinfoProc.PID())
			}
		} else {
			process.UserInfo = &userInfo
		}

		// Exclude Linux kernel processes, they are not very interesting.
		if runtime.GOOS == "linux" && userInfo.UID == "0" && process.Info.Exe == "" {
			continue
		}

		processes = append(processes, process)
	}

	return processes, nil
}
