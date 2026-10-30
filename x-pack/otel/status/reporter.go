// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"errors"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/management/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
)

// InputStatusAttributesKey is the event attribute holding the status of each
// runner. Only events aggregated from runner statuses carry it; events from
// RunnerReporter.UpdateStatus, which report the status of the whole group,
// never do.
const InputStatusAttributesKey = "inputs"

type runnerState struct {
	state status.Status
	msg   string
}

// toPdata converts a runnerState to a pdata.Map
// The format is the same as the healthcheckv2 extension
func toPdata(r *runnerState) pcommon.Map {
	pcommonMap := pcommon.NewMap()
	pcommonMap.PutStr("status", beatStatusToOtelStatus(r.state).String())
	pcommonMap.PutStr("error", r.msg)
	return pcommonMap
}

// RunnerReporter defines an interface that returns a StatusReporter for a specific runner.
// This is used for grouping and managing statuses of multiple runners
type RunnerReporter interface {
	GetReporterForRunner(id string) status.StatusReporter

	// UpdateStatus updates the group status of a runnerReporter
	UpdateStatus(status status.Status, msg string)
}

// reportInterval is the minimum time between two status reports aggregated from
// the runners. Each report carries the status of every runner, so reporting
// every change made a burst of changes, like starting many runners, quadratic.
// A change after a quiet interval is reported right away; changes within the
// interval are coalesced into one report at its end.
const reportInterval = 500 * time.Millisecond

type reporter struct {
	runnerStates map[string]*runnerState
	host         component.Host
	mtx          sync.Mutex

	interval   time.Duration
	lastReport time.Time
	// pending is the report scheduled at the end of the current interval, if any.
	pending *time.Timer
	// dirty is set when a runner status changed since the last report. A
	// scheduled report is skipped if a newer report made it redundant.
	dirty bool
}

// NewGroupStatusReporter creates a reporter that aggregates the statuses of multiple runners
// and reports the combined status to the parent StatusReporter.
// This is needed because multiple modules can report different statuses, and we want to avoid
// repeatedly flipping the parent's status.
func NewGroupStatusReporter(host component.Host) RunnerReporter {
	return newGroupStatusReporter(host, reportInterval)
}

func newGroupStatusReporter(host component.Host, interval time.Duration) *reporter {
	return &reporter{
		host:         host,
		runnerStates: make(map[string]*runnerState),
		interval:     interval,
	}
}

func (r *reporter) GetReporterForRunner(id string) status.StatusReporter {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return &subReporter{
		id: id,
		r:  r,
	}
}

func (r *reporter) updateStatusForRunner(id string, state status.Status, msg string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if r.runnerStates == nil {
		r.runnerStates = make(map[string]*runnerState)
	}
	if rState, ok := r.runnerStates[id]; ok {
		rState.msg = msg
		rState.state = state
	} else {
		// add status for the runner to the map, if not preset
		r.runnerStates[id] = &runnerState{
			state: state,
			msg:   msg,
		}
	}

	// report aggregated status for all sub-components
	r.dirty = true
	r.reportLocked()
}

// reportLocked reports the aggregated status of the runners now, or at the end
// of the current interval if a report was sent recently. r.mtx must be held.
func (r *reporter) reportLocked() {
	if r.pending != nil {
		// the scheduled report will include this change
		return
	}
	if wait := r.interval - time.Since(r.lastReport); wait > 0 {
		r.pending = time.AfterFunc(wait, func() {
			r.mtx.Lock()
			defer r.mtx.Unlock()
			r.pending = nil
			if r.dirty {
				r.sendLocked()
			}
		})
		return
	}
	r.sendLocked()
}

func (r *reporter) sendLocked() {
	r.dirty = false
	r.lastReport = time.Now()
	componentstatus.ReportStatus(r.host, r.calculateOtelStatus())
}

// UpdateStatus reports the overall status of the group.
// This is useful to report any failures encountered before a runner is initialized.
// Note: This will override all sub-reporter statuses if any
func (r *reporter) UpdateStatus(status status.Status, msg string) {
	otelStatus := beatStatusToOtelStatus(status)
	if otelStatus == componentstatus.StatusNone {
		return
	}
	var eventBuilderOpts []componentstatus.EventBuilderOption
	if componentstatus.StatusIsError(otelStatus) {
		eventBuilderOpts = append(eventBuilderOpts, componentstatus.WithError(errors.New(msg)))
	}
	evt := componentstatus.NewEvent(otelStatus, eventBuilderOpts...)

	r.mtx.Lock()
	defer r.mtx.Unlock()
	// a scheduled report would override this status with runner statuses older
	// than it; only runner changes made after it should be reported
	r.dirty = false
	componentstatus.ReportStatus(r.host, evt)
}

// calculateOtelStatus aggregates the statuses of all runners
func (r *reporter) calculateOtelStatus() *componentstatus.Event {
	var evt *componentstatus.Event
	s, msg := r.calculateAggregateState()
	otelStatus := beatStatusToOtelStatus(s)
	if otelStatus == componentstatus.StatusNone {
		return nil
	}
	var eventBuilderOpts []componentstatus.EventBuilderOption
	if componentstatus.StatusIsError(otelStatus) {
		eventBuilderOpts = append(eventBuilderOpts, componentstatus.WithError(errors.New(msg)))
	}
	evt = componentstatus.NewEvent(otelStatus, eventBuilderOpts...)

	inputStatusesPdata := evt.Attributes().PutEmptyMap(InputStatusAttributesKey)

	for id, rs := range r.runnerStates {
		inputStatePdata := toPdata(rs)
		m := inputStatusesPdata.PutEmptyMap(id)
		inputStatePdata.MoveTo(m)
	}

	return evt
}

func (r *reporter) calculateAggregateState() (status.Status, string) {
	reportedState := status.Running
	reportedMsg := ""

	for _, s := range r.runnerStates {
		switch s.state {
		case status.Degraded:
			if reportedState != status.Degraded {
				reportedState = status.Degraded
				reportedMsg = s.msg
			}
		case status.Failed:
			// we've encountered a failed runner.
			// short-circuit and return, as Failed state takes precedence over other states
			return s.state, s.msg
		default:
		}
	}
	return reportedState, reportedMsg
}

// subReporter implements status.StatusReporter
type subReporter struct {
	id string
	r  *reporter
}

func (m *subReporter) UpdateStatus(status status.Status, msg string) {
	// report status to its parent
	m.r.updateStatusForRunner(m.id, status, msg)
}

// beatStatusToOtelStatus converts a beat status to an otel status.
func beatStatusToOtelStatus(beatStatus status.Status) componentstatus.Status {
	switch beatStatus {
	case status.Starting:
		return componentstatus.StatusStarting
	case status.Running:
		return componentstatus.StatusOK
	case status.Degraded:
		return componentstatus.StatusRecoverableError
	case status.Configuring:
		return componentstatus.StatusOK
	case status.Failed:
		return componentstatus.StatusPermanentError
	case status.Stopping:
		return componentstatus.StatusStopping
	case status.Stopped:
		return componentstatus.StatusStopped
	default:
		return componentstatus.StatusNone
	}
}
