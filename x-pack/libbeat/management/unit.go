// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-libs/logp"
)

// unitState is the current state of a unit
type unitState struct {
	state status.Status
	msg   string
}

type clientUnit interface {
	ID() string
	Type() client.UnitType
	Expected() client.Expected
	UpdateState(state client.UnitState, message string, payload map[string]interface{}) error
	RegisterAction(action client.Action)
	UnregisterAction(action client.Action)
	RegisterDiagnosticHook(name string, description string, filename string, contentType string, hook client.DiagnosticHook)
}

// agentUnit implements status.StatusReporter and holds an unitState
// for the input as well as a unitState for each stream of
// the input in when this a client.UnitTypeInput.
type agentUnit struct {
	softDeleted     bool
	mtx             sync.Mutex
	logger          *logp.Logger
	clientUnit      clientUnit
	inputLevelState unitState
	streamIDs       []string
	streamStates    map[string]unitState
}

// getUnitState converts status.Status to client.UnitState
func getUnitState(s status.Status) client.UnitState {
	switch s {
	case status.Unknown:
		// must be started if its unknown
		return client.UnitStateStarting
	case status.Starting:
		return client.UnitStateStarting
	case status.Configuring:
		return client.UnitStateConfiguring
	case status.Running:
		return client.UnitStateHealthy
	case status.Degraded:
		return client.UnitStateDegraded
	case status.Failed:
		return client.UnitStateFailed
	case status.Stopping:
		return client.UnitStateStopping
	case status.Stopped:
		return client.UnitStateStopped
	default:
		// as this is an unknown state, return failed to get some attention
		return client.UnitStateFailed
	}
}

// getUnitState converts status.Status to client.UnitState
func getStatus(s client.UnitState) status.Status {
	switch s {
	case client.UnitStateStarting:
		return status.Starting
	case client.UnitStateConfiguring:
		return status.Configuring
	case client.UnitStateHealthy:
		return status.Running
	case client.UnitStateDegraded:
		return status.Degraded
	case client.UnitStateFailed:
		return status.Failed
	case client.UnitStateStopping:
		return status.Stopping
	case client.UnitStateStopped:
		return status.Stopped
	default:
		return status.Unknown
	}
}

func getStreamStates(expected client.Expected) (map[string]unitState, []string) {
	expectedCfg := expected.Config

	if expectedCfg == nil {
		return nil, nil
	}

	streamStates := make(map[string]unitState, len(expectedCfg.Streams))
	streamIDs := make([]string, len(expectedCfg.Streams))

	for idx, stream := range expectedCfg.Streams {
		streamState := unitState{
			state: status.Unknown,
			msg:   "",
		}

		if id := stream.GetId(); id != "" {
			streamIDs[idx] = id
			streamStates[id] = streamState
			continue
		}

		if cfgName := expectedCfg.GetName(); cfgName != "" {
			id := fmt.Sprintf("%s.[%d]", cfgName, idx)
			streamIDs[idx] = id
			streamStates[id] = streamState
			continue
		}

		id := fmt.Sprintf("%s.[%d]", expectedCfg.GetId(), idx)
		streamIDs[idx] = id
		streamStates[id] = streamState
	}

	return streamStates, streamIDs
}

// newAgentUnit creates a new agentUnit. In case the supplied client.Unit is of type
// client.UnitTypeInput it initializes the streamStates with a unitState.Unknown
func newAgentUnit(cu clientUnit, log *logp.Logger) *agentUnit {
	var (
		streamStates map[string]unitState
		streamIDs    []string
	)

	if cu.Type() == client.UnitTypeInput {
		streamStates, streamIDs = getStreamStates(cu.Expected())
	}

	return &agentUnit{
		clientUnit:   cu,
		logger:       log,
		streamIDs:    streamIDs,
		streamStates: streamStates,
	}
}

// RegisterAction registers action handler for this unit.
func (u *agentUnit) RegisterAction(action client.Action) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	if u.clientUnit == nil {
		return
	}

	u.clientUnit.RegisterAction(action)
}

// UnregisterAction unregisters action handler with the client.
func (u *agentUnit) UnregisterAction(action client.Action) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	if u.clientUnit == nil {
		return
	}

	u.clientUnit.UnregisterAction(action)
}

func (u *agentUnit) RegisterDiagnosticHook(name string, description string, filename string, contentType string, hook client.DiagnosticHook) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	if u.clientUnit == nil {
		return
	}

	u.clientUnit.RegisterDiagnosticHook(name, description, filename, contentType, hook)
}

func (u *agentUnit) Expected() client.Expected {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	if u.clientUnit == nil {
		return client.Expected{}
	}

	return u.clientUnit.Expected()
}

func (u *agentUnit) ID() string {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	if u.clientUnit == nil {
		return ""
	}

	return u.clientUnit.ID()
}

// calcState calculates the current state of the unit.
func (u *agentUnit) calcState() (status.Status, string) {
	// for type output return the unit state directly as it has no streams
	if u.clientUnit.Type() == client.UnitTypeOutput {
		return u.inputLevelState.state, u.inputLevelState.msg
	}

	// if inputLevelState state is not running return the inputLevelState state
	if u.inputLevelState.state != status.Running {
		return u.inputLevelState.state, u.inputLevelState.msg
	}

	// inputLevelState state is marked as running, check the stream states
	reportedStatus := status.Running
	reportedMsg := "Healthy"
	for _, streamState := range u.streamStates {
		switch streamState.state {
		case status.Degraded:
			if reportedStatus != status.Degraded {
				reportedStatus = status.Degraded
				reportedMsg = streamState.msg
			}
		case status.Failed:
			// return the first failed stream
			return streamState.state, streamState.msg
		}
	}

	return reportedStatus, reportedMsg
}

// Type of the unit.
func (u *agentUnit) Type() client.UnitType {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	if u.clientUnit == nil {
		return client.UnitTypeInput
	}

	return u.clientUnit.Type()
}

// UpdateState updates the state for the unit.
func (u *agentUnit) UpdateState(state status.Status, msg string, payload map[string]interface{}) error {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	if u.clientUnit == nil {
		return nil
	}

	if u.inputLevelState.state == state && u.inputLevelState.msg == msg {
		return nil
	}

	u.inputLevelState = unitState{
		state: state,
		msg:   msg,
	}

	state, msg = u.calcState()

	if u.clientUnit.Type() == client.UnitTypeOutput || len(u.streamStates) == 0 {
		return u.clientUnit.UpdateState(getUnitState(state), msg, payload)
	}

	streamsPayload := make(map[string]interface{}, len(u.streamStates))

	for streamID, streamState := range u.streamStates {
		streamsPayload[streamID] = map[string]interface{}{
			"status": getUnitState(streamState.state).String(),
			"error":  streamState.msg,
		}
	}

	if payload == nil {
		payload = make(map[string]interface{})
	}

	payload["streams"] = streamsPayload

	return u.clientUnit.UpdateState(getUnitState(state), msg, payload)
}

// updateStateForStream updates the state for a specific stream in the agent unit.
func (u *agentUnit) updateStateForStream(streamID string, state status.Status, msg string) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	if u.clientUnit == nil || u.streamStates == nil {
		return
	}

	if _, ok := u.streamStates[streamID]; !ok {
		return
	}

	if u.streamStates[streamID].state == state {
		return
	}

	u.streamStates[streamID] = unitState{
		state: state,
		msg:   msg,
	}

	state, msg = u.calcState()

	streamsPayload := make(map[string]interface{}, len(u.streamStates))

	for id, streamState := range u.streamStates {
		streamsPayload[id] = map[string]interface{}{
			"status": getUnitState(streamState.state).String(),
			"error":  streamState.msg,
		}
	}

	payload := map[string]interface{}{
		"streams": streamsPayload,
	}

	if err := u.clientUnit.UpdateState(getUnitState(state), msg, payload); err != nil {
		u.logger.Warnf("failed to update state for input %s: %v", u.ID(), err)
	}
}

func (u *agentUnit) update(cu *client.Unit) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	u.softDeleted = false
	u.clientUnit = cu

	inputStatus := getStatus(cu.Expected().State)
	if u.inputLevelState.state != inputStatus {
		u.inputLevelState = unitState{
			state: inputStatus,
		}
	}

	newStreamStates, newStreamIDs := getStreamStates(cu.Expected())

	for key, state := range newStreamStates {
		if _, exists := u.streamStates[key]; exists {
			continue
		}

		u.streamStates[key] = state
	}

	for key := range u.streamStates {
		if _, exists := newStreamStates[key]; !exists {
			delete(u.streamStates, key)
		}
	}

	switch {
	case len(newStreamIDs) != len(u.streamIDs):
		u.streamIDs = newStreamIDs
	default:
		for idx, streamID := range u.streamIDs {
			if newStreamIDs[idx] != streamID {
				u.streamIDs = newStreamIDs
				break
			}
		}
	}
}

func (u *agentUnit) markAsDeleted() {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	u.softDeleted = true
}

// GetReporterForStreamByIndex returns a status reporter for the stream at the given index.
// Note if the index is out of range it returns nil. It is up to the caller to check the return value.
func (u *agentUnit) GetReporterForStreamByIndex(idx int) status.StatusReporter {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	if idx >= len(u.streamIDs) {
		return nil
	}

	return &streamStatusReporter{
		id:   u.streamIDs[idx],
		unit: u,
	}
}

// streamStatusReporter implements status.StatusReporter
type streamStatusReporter struct {
	id   string
	unit *agentUnit
}

// UpdateStatus updates the status of the stream unit.
func (s *streamStatusReporter) UpdateStatus(state status.Status, msg string) {
	s.unit.updateStateForStream(s.id, state, msg)
}

// ID of the stream unit.
func (s *streamStatusReporter) ID() string {
	return s.id
}
