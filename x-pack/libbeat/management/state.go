// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/x-pack/libbeat/management/api"
)

// StateEvent is a state change notification.
var StateEvent = api.EventType("STATE")

var (
	// Starting is when the Manager is created and no config are currently active.
	Starting = State("STARTING")
	// InProgress we have received a new config from the Remote endpoint and we are trying to apply it.
	InProgress = State("IN_PROGRESS")
	// Running is set when all the config are successfully applied.
	Running = State("RUNNING")
	// Failed is set if an unpack failed, a a blacklisted option is set or when a reload fails.
	Failed = State("FAILED")
	// Stopped is set when CM is shutting down, on close the event reported will flush any pending states.
	Stopped = State("STOPPED")
)

var translateState = map[string]State{
	"STARTING":    Starting,
	"IN_PROGRESS": InProgress,
	"RUNNING":     Running,
	"FAILED":      Failed,
	"STOPPED":     Stopped,
}

// State represents the internal State of the CM Manager, it does not yet represent
// the full status of beats, because if the manager is marked as Failed it is possible that
// Beat is in fact partially working. A failed state represents an error while unpacking the config
// or when a module failed to reload.
type State string

// MarshalJSON marshals a status into a valid JSON document.
func (s *State) MarshalJSON() ([]byte, error) {
	res := struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}{
		Type:    string(*s),
		Message: fmt.Sprintf("State change: %s", *s),
	}
	return json.Marshal(&res)
}

// EventType returns the type of event.
func (s *State) EventType() api.EventType {
	return StateEvent
}

// UnmarshalJSON unmarshals the State.
func (s *State) UnmarshalJSON(b []byte) error {
	raw := struct {
		Type string `json:"type"`
	}{}

	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	v, ok := translateState[raw.Type]
	if !ok {
		return fmt.Errorf("unknown state %s", raw.Type)
	}

	*s = v
	return nil
}

func (s *State) String() string {
	return string(*s)
}
