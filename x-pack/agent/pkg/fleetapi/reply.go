// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"encoding/json"
	"time"
)

// ReplyAckedAction acks a received action from a checkin call.
type ReplyAckedAction struct {
	ActionID string
	Ts       Time
}

// Type return the type of event.
func (a *ReplyAckedAction) Type() string {
	return "ACTION"
}

// SubType returns "ACKNOWLEDGED".
func (a *ReplyAckedAction) SubType() string {
	return "ACKNOWLEDGED"
}

// Timestamp return when the event was created.
func (a *ReplyAckedAction) Timestamp() time.Time {
	return time.Time(a.Ts)
}

// Message returns the human readable string describing the event.
func (a *ReplyAckedAction) Message() string {
	return "Acknowledge action " + a.ActionID
}

// MarshalJSON custom serialization for an ReplyAckedAction.
func (a *ReplyAckedAction) MarshalJSON() ([]byte, error) {
	e := struct {
		Type     string    `json:"type"`
		Subtype  string    `json:"subtype"`
		ActionID string    `json:"action_id"`
		Ts       time.Time `json:"timestamp"`
		Msg      string    `json:"message,omitempty"`
	}{
		Type:     a.Type(),
		Subtype:  a.SubType(),
		ActionID: a.ActionID,
		Msg:      a.Message(),
		Ts:       a.Timestamp(),
	}

	return json.Marshal(e)
}

// Ack returns an event that represent an acked action.
func Ack(action Action) *ReplyAckedAction {
	return &ReplyAckedAction{
		ActionID: action.ID(),
		Ts:       Time(time.Now()),
	}
}
