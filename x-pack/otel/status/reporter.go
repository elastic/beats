// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"errors"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/management/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
)

const inputStatusAttributesKey = "inputs"

// defaultMinReportInterval bounds how often the aggregated status is reported.
// Each report covers every runner, so reporting per runner update is quadratic
// with the number of runners.
const defaultMinReportInterval = time.Second

type runnerState struct {
	state status.Status
	msg   string
}

// raw converts a runnerState to the map reported for its input.
// The format is the same as the healthcheckv2 extension
func (r *runnerState) raw() map[string]any {
	return map[string]any{
		"status": beatStatusToOtelStatus(r.state).String(),
		"error":  r.msg,
	}
}

// RunnerReporter defines an interface that returns a StatusReporter for a specific runner.
// This is used for grouping and managing statuses of multiple runners
type RunnerReporter interface {
	GetReporterForRunner(id string) status.StatusReporter

	// UpdateStatus updates the group status of a runnerReporter
	UpdateStatus(status status.Status, msg string)
}

type reporter struct {
	runnerStates map[string]*runnerState
	host         component.Host
	mtx          sync.Mutex

	minInterval time.Duration
	lastReport  time.Time
	// pending is set while a trailing report is scheduled.
	pending *time.Timer
}

// NewGroupStatusReporter creates a reporter that aggregates the statuses of multiple runners
// and reports the combined status to the parent StatusReporter.
// This is needed because multiple modules can report different statuses, and we want to avoid
// repeatedly flipping the parent's status.
func NewGroupStatusReporter(host component.Host) RunnerReporter {
	return newGroupStatusReporter(host, defaultMinReportInterval)
}

func newGroupStatusReporter(host component.Host, minInterval time.Duration) *reporter {
	return &reporter{
		host:         host,
		runnerStates: make(map[string]*runnerState),
		minInterval:  minInterval,
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
		if rState.state == state && rState.msg == msg {
			return
		}
		rState.msg = msg
		rState.state = state
	} else {
		// add status for the runner to the map, if not preset
		r.runnerStates[id] = &runnerState{
			state: state,
			msg:   msg,
		}
	}

	r.scheduleReportLocked()
}

// scheduleReportLocked reports the aggregated status of all sub-components, at
// most once per minInterval. A report that is due too early is sent once the
// interval has passed, covering every update made in the meantime.
func (r *reporter) scheduleReportLocked() {
	if r.pending != nil {
		return
	}
	wait := r.minInterval - time.Since(r.lastReport)
	if wait <= 0 {
		r.reportLocked()
		return
	}
	r.pending = time.AfterFunc(wait, func() {
		r.mtx.Lock()
		defer r.mtx.Unlock()
		r.pending = nil
		r.reportLocked()
	})
}

func (r *reporter) reportLocked() {
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

	// Build the map in one pass: PutEmptyMap looks the key up linearly, which is
	// quadratic with the number of runners.
	inputStatuses := make(map[string]any, len(r.runnerStates))
	for id, rs := range r.runnerStates {
		inputStatuses[id] = rs.raw()
	}
	if err := evt.Attributes().PutEmptyMap(inputStatusAttributesKey).FromRaw(inputStatuses); err != nil {
		return nil
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
