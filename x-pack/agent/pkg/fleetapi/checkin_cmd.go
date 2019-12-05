// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// CheckinRequest consists of multiple events reported to fleet ui.
type CheckinRequest struct {
	Events []SerializableEvent `json:"events"`
}

// SerializableEvent is a representation of the event to be send to the Fleet API via the checkin
// endpoint, we are liberal into what we accept to be send you only need a type and be able to be
// serialized into JSON.
type SerializableEvent interface {
	// Type return the type of the event, this must be included in the serialized document.
	Type() string

	// Timestamp is used to keep track when the event was created in the system.
	Timestamp() time.Time

	// Message is a human readable string to explain what the event does, this would be displayed in
	// the UI as a string of text.
	Message() string
}

// Validate validates the enrollment request before sending it to the API.
func (e *CheckinRequest) Validate() error {
	return nil
}

// CheckinResponse is the response send back from the server which contains all the action that
// need to be executed or proxy to running processes.
type CheckinResponse struct {
	Actions Actions `json:"actions"`
	Success bool    `json:"success"`
}

// Validate validates the response send from the server.
func (e *CheckinResponse) Validate() error {
	return nil
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
		return nil, errors.Wrap(err, "fail to checkin to fleet")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, extract(resp.Body)
	}

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
