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
	GetReporterForRunner(id uint64) status.StatusReporter
}

type reporter struct {
	runnerStates map[uint64]runnerState
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
		runnerStates: make(map[uint64]runnerState),
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
		r.runnerStates = make(map[uint64]runnerState)
	}

	// add status for the runner to the map
	r.runnerStates[id] = runnerState{
		state: state,
		msg:   msg,
	}

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
