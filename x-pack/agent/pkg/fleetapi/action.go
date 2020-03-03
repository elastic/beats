// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
)

// Action base interface for all the implemented action from the fleet API.
type Action interface {
	fmt.Stringer
	Type() string
	ID() string
}

// ActionUnknown is an action that is not know by the current version of the Agent and we don't want
// to return an error at parsing time but at execution time we can report or ignore.
//
// NOTE: We only keep the original type and the action id, the payload of the event is dropped, we
// do this to make sure we do not leak any unwanted information.
type ActionUnknown struct {
	originalType string
	ActionID     string
	ActionType   string
}

// Type returns the type of the Action.
func (a *ActionUnknown) Type() string {
	return "UNKNOWN"
}

// ID returns the ID of the Action.
func (a *ActionUnknown) ID() string {
	return a.ActionID
}

func (a *ActionUnknown) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ActionID)
	s.WriteString(", type: ")
	s.WriteString(a.ActionType)
	s.WriteString(" (original type: ")
	s.WriteString(a.OriginalType())
	s.WriteString(")")
	return s.String()
}

// OriginalType returns the original type of the action as returned by the API.
func (a *ActionUnknown) OriginalType() string {
	return a.originalType
}

// ActionConfigChange is a request to apply a new
type ActionConfigChange struct {
	ActionID   string
	ActionType string
	Config     map[string]interface{} `json:"config"`
}

func (a *ActionConfigChange) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ActionID)
	s.WriteString(", type: ")
	s.WriteString(a.ActionType)
	return s.String()
}

// Type returns the type of the Action.
func (a *ActionConfigChange) Type() string {
	return a.ActionType
}

// ID returns the ID of the Action.
func (a *ActionConfigChange) ID() string {
	return a.ActionID
}

// Actions is a list of Actions to executes and allow to unmarshal heterogenous action type.
type Actions []Action

// UnmarshalJSON takes every raw representation of an action and try to decode them.
func (a *Actions) UnmarshalJSON(data []byte) error {
	type r struct {
		ActionType string          `json:"type"`
		ActionID   string          `json:"id"`
		Data       json.RawMessage `json:"data"`
	}

	var responses []r

	if err := json.Unmarshal(data, &responses); err != nil {
		return errors.New(err,
			"fail to decode actions",
			errors.TypeConfig)
	}

	actions := make([]Action, 0, len(responses))
	var action Action

	for _, response := range responses {
		switch response.ActionType {
		case "CONFIG_CHANGE":
			action = &ActionConfigChange{
				ActionID:   response.ActionID,
				ActionType: response.ActionType,
			}
			if err := json.Unmarshal(response.Data, action); err != nil {
				return errors.New(err,
					"fail to decode CONFIG_CHANGE action",
					errors.TypeConfig)
			}
		default:
			action = &ActionUnknown{
				ActionID:     response.ActionID,
				ActionType:   "UNKNOWN",
				originalType: response.ActionType,
			}
		}
		actions = append(actions, action)
	}

	*a = actions
	return nil
}
