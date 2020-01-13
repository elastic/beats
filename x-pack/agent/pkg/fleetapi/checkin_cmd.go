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

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
)

const checkingPath = "/api/fleet/agents/%s/checkin"

// CheckinRequest consists of multiple events reported to fleet ui.
type CheckinRequest struct {
	Events   []SerializableEvent    `json:"events"`
	Metadata map[string]interface{} `json:"local_metadata,omitempty"`
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
	client clienter
	info   agentInfo
}

type agentInfo interface {
	AgentID() string
}

// NewCheckinCmd creates a new api command.
func NewCheckinCmd(info agentInfo, client clienter) *CheckinCmd {
	return &CheckinCmd{
		client: client,
		info:   info,
	}
}

// Execute enroll the Agent in the Fleet.
func (e *CheckinCmd) Execute(r *CheckinRequest) (*CheckinResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	b, err := json.Marshal(r)
	if err != nil {
		return nil, errors.New(err,
			"fail to encode the checkin request",
			errors.TypeUnexpected)
	}

	cp := fmt.Sprintf(checkingPath, e.info.AgentID())
	resp, err := e.client.Send("POST", cp, nil, nil, bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.New(err,
			"fail to checkin to fleet",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, cp))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, extract(resp.Body)
	}

	checkinResponse := &CheckinResponse{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(checkinResponse); err != nil {
		return nil, errors.New(err,
			"fail to decode checkin response",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, cp))
	}

	if err := checkinResponse.Validate(); err != nil {
		return nil, err
	}

	return checkinResponse, nil
}
