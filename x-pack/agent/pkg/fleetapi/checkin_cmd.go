// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// CheckinRequest consists of multiple events reported to fleet ui.
//
// Example:
// POST /api/fleet/agents/a4937110-e53e-11e9-934f-47a8e38a522c/checkin
// {
//   "events": [{
//     "type": "STATE",
//     "subtype": "STARTING",
//     "message": "state changed from STOPPED to STARTING",
//     "timestamp": "2019-10-01T13:42:54.323Z",
//     "payload": {},
//     "data": "{}"
//   }]
// }
type CheckinRequest struct {
	Events []Event `json:"events"`
}

// Event is a single event out of collection of reported events.
type Event struct {
	EventType string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	SubType   string                 `json:"subtype"`
	Message   string                 `json:"message"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Data      string                 `json:"data,omitempty"`
}

// Validate validates the enrollment request before sending it to the API.
func (e *CheckinRequest) Validate() error {
	if len(e.Events) == 0 {
		return errors.New("no events to report")
	}

	return nil
}

// CheckinResponse is a fleets response to checking API request.
//
// Example:
// 	{
// 		"action": "checkin",
// 		"success": true,
// 		"policy": {
// 		},
// 		"actions": []
//  }
type CheckinResponse struct {
	Action  string `json:"action"`
	Success bool   `json:"success"`
}

// Validate validates the response send from the server.
func (e *CheckinResponse) Validate() error {
	var err error

	return err
}

// CheckinCmd is a fleet API command.
type CheckinCmd struct {
	client      clienter
	checkinPath string
}

// NewCheckinCmd creates a new api command.
func NewCheckinCmd(agentID string, client clienter) *CheckinCmd {
	const p = "/api/fleet/agents/%s/checkin"

	return &CheckinCmd{
		client:      client,
		checkinPath: fmt.Sprintf(p, agentID),
	}
}

// Execute enroll the Agent in the Fleet.
func (e *CheckinCmd) Execute(r *CheckinRequest) (*CheckinResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	b, err := json.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "fail to encode the checkin request")
	}

	resp, err := e.client.Send("POST", e.checkinPath, nil, nil, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	checkinResponse := &CheckinResponse{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(checkinResponse); err != nil {
		return nil, errors.Wrap(err, "fail to decode checkin response")
	}

	if err := checkinResponse.Validate(); err != nil {
		return nil, err
	}

	return checkinResponse, nil
}
