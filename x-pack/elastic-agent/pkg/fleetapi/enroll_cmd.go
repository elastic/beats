// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/client"
)

// EnrollType is the type of enrollment to do with the elastic-agent.
type EnrollType string

// ErrTooManyRequests is received when the remote server is overloaded.
var ErrTooManyRequests = errors.New("too many requests received (429)")

// ErrConnRefused is returned when the connection to the server is refused.
var ErrConnRefused = errors.New("connection refused")

const (
	// PermanentEnroll is default enrollment type, by default an Agent is permanently enroll to Agent.
	PermanentEnroll = EnrollType("PERMANENT")
)

var mapEnrollType = map[string]EnrollType{
	"PERMANENT": PermanentEnroll,
}

var reverseMapEnrollType = make(map[EnrollType]string)

func init() {
	for k, v := range mapEnrollType {
		reverseMapEnrollType[v] = k
	}
}

// UnmarshalJSON unmarshal an enrollment type.
func (p *EnrollType) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) <= 2 {
		return errors.New("invalid enroll type received")
	}
	s = s[1 : len(s)-1]
	v, ok := mapEnrollType[s]
	if !ok {
		return fmt.Errorf("value of '%s' is an invalid enrollment type, supported type is 'PERMANENT'", s)
	}

	*p = v

	return nil
}

// MarshalJSON marshal an enrollType.
func (p EnrollType) MarshalJSON() ([]byte, error) {
	v, ok := reverseMapEnrollType[p]
	if !ok {
		return nil, errors.New("cannot serialize unknown type")
	}

	return json.Marshal(v)
}

// EnrollRequest is the data required to enroll the elastic-agent into Fleet.
//
// Example:
// POST /api/fleet/agents/enroll
// {
// 	"type": "PERMANENT",
//   "metadata": {
// 	  "local": { "os": "macos"},
// 	  "user_provided": { "region": "us-east"}
//   }
// }
type EnrollRequest struct {
	EnrollAPIKey string     `json:"-"`
	Type         EnrollType `json:"type"`
	SharedID     string     `json:"sharedId,omitempty"`
	Metadata     Metadata   `json:"metadata"`
}

// Metadata is a all the metadata send or received from the elastic-agent.
type Metadata struct {
	Local        *info.ECSMeta          `json:"local"`
	UserProvided map[string]interface{} `json:"user_provided"`
}

// Validate validates the enrollment request before sending it to the API.
func (e *EnrollRequest) Validate() error {
	var err error

	if len(e.EnrollAPIKey) == 0 {
		err = multierror.Append(err, errors.New("missing enrollment api key"))
	}

	if len(e.Type) == 0 {
		err = multierror.Append(err, errors.New("missing enrollment type"))
	}

	return err
}

// EnrollResponse is the data received after enrolling an Agent into fleet.
//
// Example:
// {
//   "action": "created",
//   "item": {
//     "id": "a4937110-e53e-11e9-934f-47a8e38a522c",
//     "active": true,
//     "policy_id": "default",
//     "type": "PERMANENT",
//     "enrolled_at": "2019-10-02T18:01:22.337Z",
//     "user_provided_metadata": {},
//     "local_metadata": {},
//     "actions": [],
//     "access_api_key": "API_KEY"
//   }
// }
type EnrollResponse struct {
	Action string             `json:"action"`
	Item   EnrollItemResponse `json:"item"`
}

// EnrollItemResponse item response.
type EnrollItemResponse struct {
	ID                   string                 `json:"id"`
	Active               bool                   `json:"active"`
	PolicyID             string                 `json:"policy_id"`
	Type                 EnrollType             `json:"type"`
	EnrolledAt           time.Time              `json:"enrolled_at"`
	UserProvidedMetadata map[string]interface{} `json:"user_provided_metadata"`
	LocalMetadata        map[string]interface{} `json:"local_metadata"`
	Actions              []interface{}          `json:"actions"`
	AccessAPIKey         string                 `json:"access_api_key"`
}

// Validate validates the response send from the server.
func (e *EnrollResponse) Validate() error {
	var err error

	if len(e.Item.ID) == 0 {
		err = multierror.Append(err, errors.New("missing ID"))
	}

	if len(e.Item.Type) == 0 {
		err = multierror.Append(err, errors.New("missing enrollment type"))
	}

	if len(e.Item.AccessAPIKey) == 0 {
		err = multierror.Append(err, errors.New("access api key is missing"))
	}

	return err
}

// EnrollCmd is the command to be executed to enroll an elastic-agent into Fleet.
type EnrollCmd struct {
	client client.Sender
}

// Execute enroll the Agent in the Fleet.
func (e *EnrollCmd) Execute(ctx context.Context, r *EnrollRequest) (*EnrollResponse, error) {
	const p = "/api/fleet/agents/enroll"
	const key = "Authorization"
	const prefix = "ApiKey "

	if err := r.Validate(); err != nil {
		return nil, err
	}

	headers := map[string][]string{
		key: []string{prefix + r.EnrollAPIKey},
	}

	b, err := json.Marshal(r)
	if err != nil {
		return nil, errors.New(err, "fail to encode the enrollment request")
	}

	resp, err := e.client.Send(ctx, "POST", p, nil, headers, bytes.NewBuffer(b))
	if err != nil {
		// connection refused is returned as a clean type
		switch et := err.(type) {
		case *url.Error:
			err = et.Err
		}
		switch err.(type) {
		case *net.OpError:
			return nil, ErrConnRefused
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrTooManyRequests
	}

	if resp.StatusCode != http.StatusOK {
		return nil, client.ExtractError(resp.Body)
	}

	enrollResponse := &EnrollResponse{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(enrollResponse); err != nil {
		return nil, errors.New(err, "fail to decode enrollment response")
	}

	if err := enrollResponse.Validate(); err != nil {
		return nil, err
	}

	return enrollResponse, nil
}

// NewEnrollCmd creates a new EnrollCmd.
func NewEnrollCmd(client client.Sender) *EnrollCmd {
	return &EnrollCmd{client: client}
}
