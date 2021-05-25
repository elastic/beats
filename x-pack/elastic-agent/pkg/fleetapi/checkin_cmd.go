// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/client"
)

const checkingPath = "/api/fleet/agents/%s/checkin"

// CheckinRequest consists of multiple events reported to fleet ui.
type CheckinRequest struct {
	Status   string              `json:"status"`
	AckToken string              `json:"ack_token,omitempty"`
	Events   []SerializableEvent `json:"events"`
	Metadata *info.ECSMeta       `json:"local_metadata,omitempty"`
}

// SerializableEvent is a representation of the event to be send to the Fleet Server API via the checkin
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
	AckToken string  `json:"ack_token"`
	Actions  Actions `json:"actions"`
}

// Validate validates the response send from the server.
func (e *CheckinResponse) Validate() error {
	return nil
}

// CheckinCmd is a fleet API command.
type CheckinCmd struct {
	client client.Sender
	info   agentInfo
}

type agentInfo interface {
	AgentID() string
}

// NewCheckinCmd creates a new api command.
func NewCheckinCmd(info agentInfo, client client.Sender) *CheckinCmd {
	return &CheckinCmd{
		client: client,
		info:   info,
	}
}

// Execute enroll the Agent in the Fleet Server.
func (e *CheckinCmd) Execute(ctx context.Context, r *CheckinRequest) (*CheckinResponse, error) {
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
	resp, err := e.client.Send(ctx, "POST", cp, nil, nil, bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.New(err,
			"fail to checkin to fleet-server",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, cp))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, client.ExtractError(resp.Body)
	}

	rs, _ := ioutil.ReadAll(resp.Body)

	checkinResponse := &CheckinResponse{}
	decoder := json.NewDecoder(bytes.NewReader(rs))
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
