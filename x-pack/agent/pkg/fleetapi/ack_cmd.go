// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
)

const ackPath = "/api/fleet/agents/%s/acks"

// AckRequest consists of multiple actions acked to fleet ui.
// POST /agents/{agentId}/acks
// Authorization: ApiKey {AgentAccessApiKey}
// {
//   "action_ids": ["id1"]
// }
type AckRequest struct {
	Actions []string `json:"action_ids"`
}

// Validate validates the enrollment request before sending it to the API.
func (e *AckRequest) Validate() error {
	return nil
}

// AckResponse is the response send back from the server.
// 200
// {
// 	 "action": "acks",
// 	 "success": true
// }
type AckResponse struct {
	Action  string `json:"action"`
	Success bool   `json:"success"`
}

// Validate validates the response send from the server.
func (e *AckResponse) Validate() error {
	return nil
}

// AckCmd is a fleet API command.
type AckCmd struct {
	client clienter
	info   agentInfo
}

// NewAckCmd creates a new api command.
func NewAckCmd(info agentInfo, client clienter) *AckCmd {
	return &AckCmd{
		client: client,
		info:   info,
	}
}

// Execute ACK of actions to the Fleet.
func (e *AckCmd) Execute(r *AckRequest) (*AckResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	b, err := json.Marshal(r)
	if err != nil {
		return nil, errors.New(err,
			"fail to encode the ack request",
			errors.TypeUnexpected)
	}

	ap := fmt.Sprintf(ackPath, e.info.AgentID())
	resp, err := e.client.Send("POST", ap, nil, nil, bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.New(err,
			"fail to ack to fleet",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, ap))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, extract(resp.Body)
	}

	ackResponse := &AckResponse{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(ackResponse); err != nil {
		return nil, errors.New(err,
			"fail to decode ack response",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, ap))
	}

	if err := ackResponse.Validate(); err != nil {
		return nil, err
	}

	return ackResponse, nil
}
