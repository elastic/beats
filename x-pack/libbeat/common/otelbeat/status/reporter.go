// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"errors"
	"sync"

	"github.com/elastic/beats/v7/libbeat/management/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
)

type runnerState struct {
	state status.Status
	msg   string
}

// RunnerReporter defines an interface that returns a StatusReporter for a specific runner.
// This is used for grouping and managing statuses of multiple runners
type RunnerReporter interface {
<<<<<<< HEAD
	GetReporterForRunner(id uint64) status.StatusReporter
=======
	GetReporterForRunner(id string) status.StatusReporter

	// UpdateStatus updates the group status of a runnerReporter
	UpdateStatus(status status.Status, msg string)
>>>>>>> 2ac081b10 ([beatreceiver] fix status reporting (#47936))
}

type reporter struct {
	runnerStates map[uint64]*runnerState
	host         component.Host
	mtx          sync.Mutex
}

// NewGroupStatusReporter creates a reporter that aggregates the statuses of multiple runners
// and reports the combined status to the parent StatusReporter.
// This is needed because multiple modules can report different statuses, and we want to avoid
// repeatedly flipping the parent's status.
func NewGroupStatusReporter(host component.Host) RunnerReporter {
	return &reporter{
		host:         host,
		runnerStates: make(map[uint64]*runnerState),
	}
}

func (r *reporter) GetReporterForRunner(id uint64) status.StatusReporter {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return &subReporter{
		id: id,
		r:  r,
	}
}

func (r *reporter) updateStatusForRunner(id uint64, state status.Status, msg string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if r.runnerStates == nil {
		r.runnerStates = make(map[uint64]*runnerState)
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

<<<<<<< HEAD
	// calculate the aggregate state of beat based on the module states
	calcState, calcMsg := r.calculateState()

	// report status to parent reporter
	r.UpdateStatus(calcState, calcMsg)
}

func (r *reporter) UpdateStatus(s status.Status, msg string) {
	var evt *componentstatus.Event
	switch s {
	case status.Starting:
		evt = componentstatus.NewEvent(componentstatus.StatusStarting)
	case status.Running:
		evt = componentstatus.NewEvent(componentstatus.StatusOK)
	case status.Degraded:
		evt = componentstatus.NewRecoverableErrorEvent(errors.New(msg))
	case status.Failed:
		evt = componentstatus.NewPermanentErrorEvent(errors.New(msg))
	case status.Stopping:
		evt = componentstatus.NewEvent(componentstatus.StatusStopped)
	case status.Stopped:
		evt = componentstatus.NewEvent(componentstatus.StatusStopped)
	default:
		return
	}

	componentstatus.ReportStatus(r.host, evt)
}

func (r *reporter) calculateState() (status.Status, string) {
=======
	// report aggregated status for all sub-components
	evt := r.calculateOtelStatus()
	r.emitDummyStatus(evt)
	componentstatus.ReportStatus(r.host, evt)
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
	r.emitDummyStatus(evt)
	componentstatus.ReportStatus(r.host, evt)
}

func (r *reporter) emitDummyStatus(evt *componentstatus.Event) {
	oppositeStatus := getOppositeStatus(evt.Status())
	if oppositeStatus != componentstatus.StatusNone {
		// emit a dummy event first to ensure the otel core framework acknowledges the change
		// workaround for https://github.com/open-telemetry/opentelemetry-collector/issues/14282
		dummyEvt := componentstatus.NewEvent(oppositeStatus)
		componentstatus.ReportStatus(r.host, dummyEvt)
	}
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

	inputStatusesPdata := evt.Attributes().PutEmptyMap(inputStatusAttributesKey)

	for id, rs := range r.runnerStates {
		inputStatePdata := toPdata(rs)
		m := inputStatusesPdata.PutEmptyMap(id)
		inputStatePdata.MoveTo(m)
	}

	return evt
}

func (r *reporter) calculateAggregateState() (status.Status, string) {
>>>>>>> 2ac081b10 ([beatreceiver] fix status reporting (#47936))
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
		}
	}
	return reportedState, reportedMsg
}

// subReporter implements status.StatusReporter
type subReporter struct {
	id uint64
	r  *reporter
}

func (m *subReporter) UpdateStatus(status status.Status, msg string) {
	// report status to its parent
	m.r.updateStatusForRunner(m.id, status, msg)
}
