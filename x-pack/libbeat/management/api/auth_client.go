// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gofrs/uuid"
	"github.com/joeshaw/multierror"
)

// EventType is the type of event that the events endpoint can understand.
type EventType string

// Event is the interface for the events to be send to the event endpoint.
type Event interface {
	json.Marshaler
	EventType() EventType
}

// EventRequest is the data send to the CM event endpoint.
type EventRequest struct {
	Timestamp time.Time `json:"timestamp"`
	EventType EventType `json:"type"`
	Event     Event     `json:"event"`
}

// EventAPIResponse is the top level response for the events endpoints.
type EventAPIResponse struct {
	BaseResponse
	Response []EventResponse `json:"results"`
}

// EventResponse is the indiviual response for each event request.
type EventResponse struct {
	BaseResponse
}

// AuthClienter is the interface exposed by the auth client and is useful for testing without calling
// a remote endpoint.
type AuthClienter interface {
	// SendEvents takes a slices of event request and send them to the endpoint.
	SendEvents([]EventRequest) error

	// Configuration retrieves the list of configuration blocks from Kibana
	Configuration() (ConfigBlocks, error)
}

// AuthClient is a authenticated client to the CM endpoint and exposes the calls that require
// the clients to pass credentials (UUID and AccessToken).
type AuthClient struct {
	Client      *Client
	BeatUUID    uuid.UUID
	AccessToken string
}

func (c AuthClient) headers() http.Header {
	headers := http.Header{}
	headers.Set("kbn-beats-access-token", c.AccessToken)
	return headers
}

// SendEvents send a list of events to Kibana.
func (c *AuthClient) SendEvents(requests []EventRequest) error {
	sort.SliceStable(requests, func(i, j int) bool {
		return requests[i].Timestamp.Before(requests[j].Timestamp)
	})

	resp := EventAPIResponse{}
	url := fmt.Sprintf("/api/beats/%s/events", c.BeatUUID)
	statusCode, err := c.Client.request("POST", url, requests, c.headers(), &resp)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return fmt.Errorf(
			"invalid response code while sending events, expected 200 and received %d",
			statusCode,
		)
	}

	if len(resp.Response) != len(requests) {
		return fmt.Errorf(
			"number of response and the request do not match, expecting %d and received %d",
			len(requests),
			len(resp.Response),
		)
	}

	// Loop through the responses and see if all items are marked as `success` we assume the response
	// are in the same order as the sending order.
	//
	// We could add logic later to retry them, currently if sending error fails it's probably because
	// Kibana is not answering and the next fetch will probably fails.
	var errors multierror.Errors
	for _, response := range resp.Response {
		if !response.Success {
			errors = append(errors, fmt.Errorf("error sending event, reason: %+v", response.Error.Message))
		}
	}

	return errors.Err()
}
