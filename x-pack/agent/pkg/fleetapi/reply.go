package fleetapi

import (
	"strings"
	"time"
)

// UnknownAction is an action that is not know by the current version of the Agent and we don't want
// to return an error at parsing time but at execution time we can report or ignore.
//
// NOTE: We only keep the original type and the action id, the payload of the event is dropped, we
// do this to make sure we do not leak any unwanted information.
type UnknownAction struct {
	*BaseAction
	originalType string
}

// Type returns the type of the Action.
func (a *UnknownAction) Type() string {
	return "Action"
}

// SubType returns the subtype of the action.
func (a *UnknownAction) SubType() {
	return "UNKNOWN"
}

func (a *UnknownAction) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ID())
	s.WriteString(", type: ")
	s.WriteString(a.Type())
	s.WriteString(", sub type: ")
	s.WriteString(a.SubType())
	s.WriteString(" (original type: ")
	s.WriteString(a.OriginalType())
	s.WriteString(")")
	return s.String()
}

// OriginalType returns the original type of the action as returned by the API.
func (a *UnknownAction) OriginalType() string {
	return a.originalType
}

// AckedAction represents a event to be send to the next checkin that will Ack an action.
type AckedAction struct {
	EventType string    `json:"type"`
	ActionID  string    `json:"action_id"`
	Ts        time.Time `json:"timestamp"`
	Msg       string    `json:"message,omitempty"`
}

// Type return the type of event.
func (a *AckedAction) Type() string {
	return a.EventType
}

// Timestamp return when the event was created.
func (a *AckedAction) Timestamp() time.Time {
	return a.Ts
}

// Message returns the human readable string describing the event.
func (a *AckedAction) Message() string {
	return a.Msg
}

// Ack returns an event that represent an acked action.
func Ack(action Action) *AckedAction {
	const t = "ACTION_ACKNOWLEDGED"
	const st = "ACKNOWLEDGED"

	return &AckedAction{
		EventType:    t,
		EventSubType: st,
		ActionID:     action.ID(),
		Msg:          "Acknowledge action " + action.ID(),
		Ts:           time.Now(),
	}
}
