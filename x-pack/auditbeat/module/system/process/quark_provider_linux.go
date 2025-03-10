// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && (amd64 || arm64) && cgo

package process

import (
	"fmt"
	"os/user"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/auditbeat/helper/hasher"
	"github.com/elastic/beats/v7/auditbeat/helper/tty"
	"github.com/elastic/beats/v7/libbeat/common/capabilities"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"

	quark "github.com/elastic/go-quark"
)

var quarkMetrics = struct {
	insertions      *monitoring.Uint
	removals        *monitoring.Uint
	aggregations    *monitoring.Uint
	nonAggregations *monitoring.Uint
	lost            *monitoring.Uint
	backend         *monitoring.String
}{}

func init() {
	reg := monitoring.Default.NewRegistry("process@quark")
	quarkMetrics.insertions = monitoring.NewUint(reg, "insertions")
	quarkMetrics.removals = monitoring.NewUint(reg, "removals")
	quarkMetrics.aggregations = monitoring.NewUint(reg, "aggregations")
	quarkMetrics.nonAggregations = monitoring.NewUint(reg, "non_aggregations")
	quarkMetrics.lost = monitoring.NewUint(reg, "lost")
	quarkMetrics.backend = monitoring.NewString(reg, "backend", monitoring.Report)
}

// QuarkMetricSet is a MetricSet with added members used only in and by
// quark. QuarkMetricSet uses mb.PushReporterV2 instead of
// mb.ReporterV2. More notably we don't do periodic state reports and
// we don't need a cache as it is provided by quark.
type QuarkMetricSet struct {
	MetricSet
	queue        *quark.Queue // Quark runtime state
	selfMntNsIno uint32       // Mnt inode from current process
	cachedHasher *hasher.CachedHasher
}

// Used for testing only and not exposed via config
var quarkForceKprobe bool

// NewFromQuark instantiates the module with quark's backend.
func NewFromQuark(ms MetricSet) (mb.MetricSet, error) {
	var qm QuarkMetricSet

	qm.MetricSet = ms

	ino64, err := selfNsIno("mnt")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch self mount inode: %w", err)
	}
	qm.selfMntNsIno = uint32(ino64)
	qm.cachedHasher, err = hasher.NewFileHasherWithCache(qm.config.HasherConfig, 4096)
	if err != nil {
		return nil, fmt.Errorf("can't create hash cache: %w", err)
	}

	attr := quark.DefaultQueueAttr()
	if quarkForceKprobe {
		attr.Flags &= ^quark.QQ_ALL_BACKENDS
		attr.Flags |= quark.QQ_KPROBE
	}
	qm.queue, err = quark.OpenQueue(attr, 1)
	if err != nil {
		qm.cachedHasher.Close()
		return nil, fmt.Errorf("can't open quark queue: %w", err)
	}
	stats := qm.queue.Stats()
	if stats.Backend == quark.QQ_EBPF {
		qm.log.Info("quark using EBPF")
	} else if stats.Backend == quark.QQ_KPROBE {
		qm.log.Info("quark using KPROBES")
	} else {
		qm.queue.Close()
		qm.cachedHasher.Close()
		return nil, fmt.Errorf("quark has an invalid backend")
	}

	return &qm, nil
}

// Run reads events from quark's queue and pushes them into output.
// The queue is owned by this goroutine and should not be touched
// from outside as there is no synchronization.
func (ms *QuarkMetricSet) Run(r mb.PushReporterV2) {
	ms.log.Info("Quark running")

	metricsStamp := time.Now()

MainLoop:
	for {
		// Poll for done
		select {
		case <-r.Done():
			break MainLoop
		default:
		}

		ms.maybeUpdateMetrics(&metricsStamp)

		x := time.Now()
		quarkEvents, err := ms.queue.GetEvents()
		if len(quarkEvents) == 1 {
			ms.log.Debugf("getevents took %v", time.Since(x))
		}
		if err != nil {
			ms.log.Error("quark GetEvents, unrecoverable error", err)
			break MainLoop
		}
		if len(quarkEvents) == 0 {
			err = ms.queue.Block()
			if err != nil {
				ms.log.Error("quark Block, unrecoverable error", err)
				break MainLoop
			}
			continue
		}
		for _, quarkEvent := range quarkEvents {
			if !wantedEvent(quarkEvent) {
				continue
			}
			if event, ok := ms.toEvent(quarkEvent); ok {
				r.Event(event)
			}
		}
	}

	// Queue is owned by this goroutine, if we ever access it from
	// outside, we need to consider synchronization.
	ms.cachedHasher.Close()
	ms.queue.Close()
	ms.queue = nil
}

// toEvent converts a quark.Event to a mb.Event, returns true if we
// were able to make an event.
func (ms *QuarkMetricSet) toEvent(quarkEvent quark.Event) (mb.Event, bool) {
	action, evtype := actionAndTypeOfEvent(quarkEvent)
	process := quarkEvent.Process
	event := mb.Event{RootFields: mapstr.M{}}

	var username string
	var processErr error
	defer func() {
		// Fill out root message and error.message
		event.RootFields.Put("message",
			makeMessage(int(process.Pid), action, process.Comm, username, processErr))
		if processErr != nil {
			event.RootFields.Put("error.message", processErr.Error())
		}
	}()

	// Values that are independent of Proc.Valid
	// Fill out event.*
	event.RootFields.Put("event.type", evtype)
	event.RootFields.Put("event.action", action.String())
	event.RootFields.Put("event.category", []string{"process"})
	event.RootFields.Put("event.kind", "event")
	// Fill out process.*
	event.RootFields.Put("process.name", process.Comm)
	event.RootFields.Put("process.args", process.Cmdline)
	event.RootFields.Put("process.args_count", len(process.Cmdline))
	event.RootFields.Put("process.pid", process.Pid)
	event.RootFields.Put("process.working_directory", process.Cwd)
	event.RootFields.Put("process.executable", process.Filename)
	if process.Exit.Valid {
		event.RootFields.Put("process.exit_code", process.Exit.ExitCode)
	}
	if !process.Proc.Valid {
		return event, true
	}

	//
	// Code below can rely on Proc
	//

	// Ids
	event.RootFields.Put("process.parent.pid", process.Proc.Ppid)
	startTime := time.Unix(0, int64(process.Proc.TimeBoot))
	if ms.HostID() != "" {
		// TODO unify with sessionview and guarantee loss of precision
		event.RootFields.Put("process.entity_id",
			entityID(ms.HostID(), int(process.Pid), startTime))
	}
	event.RootFields.Put("process.start", startTime)
	event.RootFields.Put("user.id", process.Proc.Uid)
	event.RootFields.Put("user.group.id", process.Proc.Gid)
	event.RootFields.Put("user.effective.id", process.Proc.Euid)
	event.RootFields.Put("user.effective.group.id", process.Proc.Egid)
	event.RootFields.Put("user.saved.id", process.Proc.Suid)
	event.RootFields.Put("user.saved.group.id", process.Proc.Sgid)
	if us, err := user.LookupId(strconv.FormatUint(uint64(process.Proc.Uid), 10)); err == nil {
		event.RootFields.Put("user.name", us.Username)
		username = us.Username
	}
	if group, err := user.LookupGroupId(strconv.FormatUint(uint64(process.Proc.Gid), 10)); err == nil {
		event.RootFields.Put("user.group.name", group.Name)
	}
	// Tty things
	event.RootFields.Put("process.interactive",
		tty.InteractiveFromTTY(tty.TTYDev{
			Major: process.Proc.TtyMajor,
			Minor: process.Proc.TtyMinor,
		}))
	if process.Proc.TtyMajor != 0 {
		event.RootFields.Put("process.tty.char_device.major", process.Proc.TtyMajor)
		event.RootFields.Put("process.tty.char_device.minor", process.Proc.TtyMinor)
	}
	// Capabilities
	capEffective, _ := capabilities.FromUint64(process.Proc.CapEffective)
	if len(capEffective) > 0 {
		event.RootFields.Put("process.thread.capabilities.effective", capEffective)
	}
	capPermitted, _ := capabilities.FromUint64(process.Proc.CapPermitted)
	if len(capPermitted) > 0 {
		event.RootFields.Put("process.thread.capabilities.permitted", capPermitted)
	}
	// If we are in the same mount namespace of the process, hash
	// the file. When quark is running on kprobes, there are
	// limitations concerning the full path of the filename, in
	// those cases, the path won't start with a slash.
	if process.Proc.MntInonum == ms.selfMntNsIno && len(process.Filename) > 0 && process.Filename[0] == '/' {
		hashes, err := ms.cachedHasher.HashFile(process.Filename)
		if err != nil {
			processErr = fmt.Errorf("failed to hash executable %v for PID %v: %w",
				process.Filename, process.Pid, err)
			ms.log.Warn(processErr.Error())
		} else {
			for hashType, digest := range hashes {
				fieldName := "process.hash." + string(hashType)
				event.RootFields.Put(fieldName, digest)
			}
		}
	} else {
		ms.log.Debugf("skipping hash %s (inonum %d vs %d)", process.Filename, process.Proc.MntInonum, ms.selfMntNsIno)
	}

	return event, true
}

// wantedEvent filters in only the wanted events from quark.
func wantedEvent(quarkEvent quark.Event) bool {
	const wanted uint64 = quark.QUARK_EV_FORK |
		quark.QUARK_EV_EXEC |
		quark.QUARK_EV_EXIT |
		quark.QUARK_EV_SNAPSHOT
	if quarkEvent.Events&wanted == 0 ||
		quarkEvent.Process.Pid == 2 ||
		quarkEvent.Process.Proc.Ppid == 2 { // skip kthreads

		return false
	}

	return true
}

// actionAndTypeOfEvent computes eventAction and event.type out of a quark.Event.
func actionAndTypeOfEvent(quarkEvent quark.Event) (eventAction, []string) {
	snap := quarkEvent.Events&quark.QUARK_EV_SNAPSHOT != 0
	fork := quarkEvent.Events&quark.QUARK_EV_FORK != 0
	exec := quarkEvent.Events&quark.QUARK_EV_EXEC != 0
	exit := quarkEvent.Events&quark.QUARK_EV_EXIT != 0

	// Calculate event.action
	// If it's a snap, it's existing
	// If it forked + exited and executed or not, we consider ran
	// If it execed + exited we consider stopped
	// If it execed but didn't fork or exit, we consider changed image
	var action eventAction
	if snap {
		action = eventActionExistingProcess
	} else if fork && exit {
		action = eventActionProcessRan
	} else if fork {
		action = eventActionProcessStarted
	} else if exit {
		action = eventActionProcessStopped
	} else if exec {
		action = eventActionProcessChangedImage
	} else {
		action = eventActionProcessError
	}
	// Calculate event.type
	evtype := make([]string, 0, 4)
	if snap {
		evtype = append(evtype, eventActionExistingProcess.Type())
	}
	if fork {
		evtype = append(evtype, eventActionProcessStarted.Type())
	}
	if exec {
		evtype = append(evtype, eventActionProcessChangedImage.Type())
	}
	if exit {
		evtype = append(evtype, eventActionProcessStopped.Type())
	}

	return action, evtype
}

func (ms *QuarkMetricSet) maybeUpdateMetrics(stamp *time.Time) {
	if time.Since(*stamp) < time.Second*5 {
		return
	}

	stats := ms.queue.Stats()
	quarkMetrics.insertions.Set(stats.Insertions)
	quarkMetrics.removals.Set(stats.Removals)
	quarkMetrics.aggregations.Set(stats.Aggregations)
	quarkMetrics.nonAggregations.Set(stats.NonAggregations)
	quarkMetrics.lost.Set(stats.Lost)
	if stats.Backend == quark.QQ_EBPF {
		quarkMetrics.backend.Set("ebpf")
	} else if stats.Backend == quark.QQ_KPROBE {
		quarkMetrics.backend.Set("kprobe")
	} else {
		quarkMetrics.backend.Set("invalid")
	}

	*stamp = time.Now()
}
