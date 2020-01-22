// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

var testEventType = EventType("TEST_EVENT")

// Create a custom Event type for testing.
type testEvent struct {
	Message string    `json:"message"`
	Type    EventType `json:"event_type"`
}

func (er *EventRequest) UnmarshalJSON(b []byte) error {
	resp := struct {
		EventType EventType       `json:"type"`
		Event     json.RawMessage `json:"event"`
	}{}

	if err := json.Unmarshal(b, &resp); err != nil {
		return err
	}

	switch resp.EventType {
	case testEventType:
		event := &testEvent{}
		if err := json.Unmarshal(resp.Event, event); err != nil {
			return err
		}
		*er = EventRequest{EventType: resp.EventType, Event: event}
		return nil
	}
	return fmt.Errorf("unknown event type of '%s'", resp.EventType)
}

func (t *testEvent) EventType() EventType {
	return t.Type
}
func (t *testEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(*t)
}

func (t *testEvent) UnmarshalJSON(b []byte) error {
	resp := struct {
		Message string    `json:"message"`
		Type    EventType `json:"type"`
	}{}
	if err := json.Unmarshal(b, &resp); err != nil {
		return err
	}
	*t = testEvent{Message: resp.Message}
	return nil
}

func TestReportEvents(t *testing.T) {
	beatUUID, err := uuid.NewV4()
	if !assert.NoError(t, err) {
		return
	}

	accessToken := "my-enroll-token"

	t.Run("successfully send events", func(t *testing.T) {
		server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check correct path is used
			assert.Equal(t, "/api/beats/"+beatUUID.String()+"/events", r.URL.Path)

			// Check enrollment token is correct
			assert.Equal(t, accessToken, r.Header.Get("kbn-beats-access-token"))

			var response []EventRequest

			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&response)
			if !assert.NoError(t, err) {
				return
			}

			if !assert.Equal(t, 1, len(response)) {
				return
			}

			expected := &testEvent{Message: "OK"}
			received := response[0].Event.(*testEvent)

			if !assert.Equal(t, expected.Message, received.Message) {
				return
			}

			apiResponse := EventAPIResponse{
				Response: []EventResponse{EventResponse{BaseResponse: BaseResponse{Success: true}}},
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(apiResponse)
		}))
		defer server.Close()
		auth := &AuthClient{Client: client, AccessToken: accessToken, BeatUUID: beatUUID}

		events := []*testEvent{&testEvent{Message: "OK"}}

		err = reportEvents(auth, events)
		assert.NoError(t, err)
	})

	t.Run("bubble up any errors", func(t *testing.T) {
		server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			response := BaseResponse{
				Success: false,
				Error: ErrorResponse{
					Message: "bad request",
					Code:    http.StatusBadRequest,
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		auth := &AuthClient{Client: client, AccessToken: accessToken, BeatUUID: beatUUID}

		events := []*testEvent{&testEvent{Message: "OK"}}

		err = reportEvents(auth, events)
		assert.Error(t, err)
	})

	t.Run("assert the response", func(t *testing.T) {
		server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiResponse := EventAPIResponse{
				Response: []EventResponse{
					EventResponse{BaseResponse: BaseResponse{Success: true}},
					EventResponse{BaseResponse: BaseResponse{Success: false}},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(apiResponse)
		}))
		defer server.Close()

		auth := &AuthClient{Client: client, AccessToken: accessToken, BeatUUID: beatUUID}

		events := []*testEvent{
			&testEvent{Message: "testing-1"},
			&testEvent{Message: "testing-2"},
		}

		err = reportEvents(auth, events)
		assert.Error(t, err)
	})

	t.Run("enforce the match of response/request", func(t *testing.T) {
		server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiResponse := EventAPIResponse{
				Response: []EventResponse{
					EventResponse{BaseResponse: BaseResponse{Success: true}},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(apiResponse)
		}))
		defer server.Close()

		auth := &AuthClient{Client: client, AccessToken: accessToken, BeatUUID: beatUUID}

		events := []*testEvent{
			&testEvent{Message: "testing-1"},
			&testEvent{Message: "testing-2"},
		}

		err = reportEvents(auth, events)
		assert.Error(t, err)
	})
}

func reportEvents(client AuthClienter, events []*testEvent) error {
	requests := make([]EventRequest, len(events))
	for idx, err := range events {
		requests[idx] = EventRequest{
			Timestamp: time.Now(),
			EventType: testEventType,
			Event:     err,
		}
	}
	return client.SendEvents(requests)
}
