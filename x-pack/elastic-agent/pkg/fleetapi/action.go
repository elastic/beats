// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

const (
	// ActionTypeUpgrade specifies upgrade action.
	ActionTypeUpgrade = "UPGRADE"
	// ActionTypeUnenroll specifies unenroll action.
	ActionTypeUnenroll = "UNENROLL"
	// ActionTypePolicyChange specifies policy change action.
	ActionTypePolicyChange = "POLICY_CHANGE"
	// ActionTypeSettings specifies change of agent settings.
	ActionTypeSettings = "SETTINGS"
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

// ActionPolicyChange is a request to apply a new
type ActionPolicyChange struct {
	ActionID   string
	ActionType string
	Policy     map[string]interface{} `json:"policy"`
}

func (a *ActionPolicyChange) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ActionID)
	s.WriteString(", type: ")
	s.WriteString(a.ActionType)
	return s.String()
}

// Type returns the type of the Action.
func (a *ActionPolicyChange) Type() string {
	return a.ActionType
}

// ID returns the ID of the Action.
func (a *ActionPolicyChange) ID() string {
	return a.ActionID
}

// ActionUpgrade is a request for agent to upgrade.
type ActionUpgrade struct {
	ActionID   string `json:"id" yaml:"id"`
	ActionType string `json:"type" yaml:"type"`
	Version    string `json:"version" yaml:"version"`
	SourceURI  string `json:"source_uri,omitempty" yaml:"source_uri,omitempty"`
}

func (a *ActionUpgrade) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ActionID)
	s.WriteString(", type: ")
	s.WriteString(a.ActionType)
	return s.String()
}

// Type returns the type of the Action.
func (a *ActionUpgrade) Type() string {
	return a.ActionType
}

// ID returns the ID of the Action.
func (a *ActionUpgrade) ID() string {
	return a.ActionID
}

// ActionUnenroll is a request for agent to unhook from fleet.
type ActionUnenroll struct {
	ActionID   string
	ActionType string
	IsDetected bool
}

func (a *ActionUnenroll) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ActionID)
	s.WriteString(", type: ")
	s.WriteString(a.ActionType)
	return s.String()
}

// Type returns the type of the Action.
func (a *ActionUnenroll) Type() string {
	return a.ActionType
}

// ID returns the ID of the Action.
func (a *ActionUnenroll) ID() string {
	return a.ActionID
}

// ActionSettings is a request to change agent settings.
type ActionSettings struct {
	ActionID   string
	ActionType string
	LogLevel   string `json:"log_level"`
}

func (a *ActionSettings) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ActionID)
	s.WriteString(", type: ")
	s.WriteString(a.ActionType)
	s.WriteString(", log_level: ")
	s.WriteString(a.LogLevel)
	return s.String()
}

// Type returns the type of the Action.
func (a *ActionSettings) Type() string {
	return a.ActionType
}

// ID returns the ID of the Action.
func (a *ActionSettings) ID() string {
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
		case ActionTypePolicyChange:
			action = &ActionPolicyChange{
				ActionID:   response.ActionID,
				ActionType: response.ActionType,
			}
			if err := json.Unmarshal(response.Data, action); err != nil {
				return errors.New(err,
					"fail to decode POLICY_CHANGE action",
					errors.TypeConfig)
			}
		case ActionTypeUnenroll:
			action = &ActionUnenroll{
				ActionID:   response.ActionID,
				ActionType: response.ActionType,
			}
		case ActionTypeUpgrade:
			action = &ActionUpgrade{
				ActionID:   response.ActionID,
				ActionType: response.ActionType,
			}

			if err := json.Unmarshal(response.Data, action); err != nil {
				return errors.New(err,
					"fail to decode UPGRADE_ACTION action",
					errors.TypeConfig)
			}
		case ActionTypeSettings:
			action = &ActionSettings{
				ActionID:   response.ActionID,
				ActionType: response.ActionType,
			}

			if err := json.Unmarshal(response.Data, action); err != nil {
				return errors.New(err,
					"fail to decode SETTINGS_ACTION action",
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
