package fleetapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Action base interface for all the implemented action from the fleet API.
type Action interface {
	fmt.Stringer
	Type() string
	ID() string
}

// BaseAction is the base of all actions to be executed.
type BaseAction struct {
	ActionID   string
	ActionType string
}

// Type returns the action type.
func (a *BaseAction) Type() string {
	return a.ActionType
}

// ID returns the action ID.
func (a *BaseAction) ID() string {
	return a.ActionID
}

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
	return "UNKNOWN"
}

func (a *UnknownAction) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ID())
	s.WriteString(", type: ")
	s.WriteString(a.Type())
	s.WriteString(" (original type: ")
	s.WriteString(a.OriginalType())
	s.WriteString(")")
	return s.String()
}

// OriginalType returns the original type of the action as returned by the API.
func (a *UnknownAction) OriginalType() string {
	return a.originalType
}

// PolicyChangeAction is a request to apply a new
type PolicyChangeAction struct {
	*BaseAction
	Policy map[string]interface{} `json:"policy"`
}

func (a *PolicyChangeAction) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ID())
	s.WriteString(", type: ")
	s.WriteString(a.Type())
	return s.String()
}

// Actions is a list of Actions to executes and allow to unmarshal heterogenous action type.
type Actions []Action

// UnmarshalJSON takes every raw representation of an action and try to decode them.
func (a *Actions) UnmarshalJSON(data []byte) error {
	type actionResponse struct {
		ActionType string          `json:"type"`
		ActionID   string          `json:"id"`
		Data       json.RawMessage `json:"data"`
	}

	var responses []actionResponse

	if err := json.Unmarshal(data, &responses); err != nil {
		return errors.Wrap(err, "fail to decode actions")
	}

	actions := make([]Action, 0, len(responses))
	var action Action

	for _, response := range responses {
		switch response.ActionType {
		case "POLICY_CHANGE":
			action = &PolicyChangeAction{
				BaseAction: &BaseAction{
					ActionID:   response.ActionID,
					ActionType: response.ActionType,
				},
			}
			if err := json.Unmarshal(response.Data, action); err != nil {
				return errors.Wrap(err, "fail to decode POLICY_CHANGE action")
			}
		default:
			action = &UnknownAction{
				BaseAction:   &BaseAction{ActionID: response.ActionID, ActionType: "UNKNOWN"},
				originalType: response.ActionType,
			}
		}
		actions = append(actions, action)
	}

	*a = actions
	return nil
}
