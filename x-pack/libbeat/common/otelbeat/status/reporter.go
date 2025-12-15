// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"errors"
	"sync"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/management/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
)

const inputStatusAttributesKey = "inputs"

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
	GetReporterForRunner(id uint64) status.StatusReporter

	// UpdateStatus updates the group status of a runnerReporter
	UpdateStatus(status status.Status, msg string)
}

type reporter struct {
	runnerStates map[string]*runnerState
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
		runnerStates: make(map[string]*runnerState),
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

	// report status to parent reporter
	r.UpdateStatus()
}

func (r *reporter) UpdateStatus() {
	evt := r.calculateOtelStatus()
	oppositeStatus := getOppositeStatus(evt.Status())
	if oppositeStatus != componentstatus.StatusNone {
		// emit a dummy event first to ensure the otel core framework acknowledges the change
		// workaround for https://github.com/open-telemetry/opentelemetry-collector/issues/14282
		dummyEvt := componentstatus.NewEvent(oppositeStatus)
		componentstatus.ReportStatus(r.host, dummyEvt)
	}
	componentstatus.ReportStatus(r.host, evt)
}

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

// getOppositeStatus returns the opposite status of the given status, and None if no such status exists.
func getOppositeStatus(status componentstatus.Status) componentstatus.Status {
	switch status {
	case componentstatus.StatusOK:
		return componentstatus.StatusRecoverableError
	case componentstatus.StatusRecoverableError:
		return componentstatus.StatusOK
	default:
		return componentstatus.StatusNone
	}
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
