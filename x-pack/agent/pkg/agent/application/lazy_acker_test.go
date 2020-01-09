// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
)

func TestLazyAcker(t *testing.T) {
	type serializedEvent struct {
		Type     string `json:"type"`
		Subtype  string `json:"subtype"`
		ActionID string `json:"action_id"`
	}

	type ackRequest struct {
		Events []serializedEvent `json:"events"`
	}

	log, _ := logger.New()
	client := newTestingClient()
	agentInfo := &testAgentInfo{}
	acker, err := newActionAcker(log, agentInfo, client)
	if err != nil {
		t.Fatal(err)
	}

	lacker := newLazyAcker(acker)

	if acker == nil {
		t.Fatal("acker not initialized")
	}

	testID1 := "ack-test-action-id"
	testID2 := testID1 + "2"
	testID3 := testID1 + "3"
	testAction1 := &fleetapi.ActionUnknown{ActionBase: &fleetapi.ActionBase{ActionID: testID1}}
	testAction2 := &actionImmediate{ActionBase: &fleetapi.ActionBase{ActionID: testID2}}
	testAction3 := &fleetapi.ActionUnknown{ActionBase: &fleetapi.ActionBase{ActionID: testID3}}

	ch := client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
		content, err := ioutil.ReadAll(body)
		assert.NoError(t, err)
		cr := &ackRequest{}
		err = json.Unmarshal(content, &cr)
		assert.NoError(t, err)

		if len(cr.Events) == 0 {
			t.Fatal("expected events but got none")
		}
		if cr.Events[0].ActionID == testID1 {
			assert.EqualValues(t, 2, len(cr.Events))

			assert.EqualValues(t, "ACTION", cr.Events[0].Type)
			assert.EqualValues(t, "ACKNOWLEDGED", cr.Events[0].Subtype)
			assert.EqualValues(t, testID1, cr.Events[0].ActionID)

			assert.EqualValues(t, "ACTION", cr.Events[1].Type)
			assert.EqualValues(t, "ACKNOWLEDGED", cr.Events[1].Subtype)
			assert.EqualValues(t, testID2, cr.Events[1].ActionID)

		} else {
			assert.EqualValues(t, 1, len(cr.Events))

			assert.EqualValues(t, "ACTION", cr.Events[0].Type)
			assert.EqualValues(t, "ACKNOWLEDGED", cr.Events[0].Subtype)
			assert.EqualValues(t, testID3, cr.Events[0].ActionID)
		}

		resp := wrapStrToResp(http.StatusOK, `{ "actions": [], "success": true }`)
		return resp, nil
	})

	go func() {
		for range ch {
		}
	}()

	if err := lacker.Ack(testAction1); err != nil {
		t.Fatal(err)
	}
	if err := lacker.Ack(testAction2); err != nil {
		t.Fatal(err)
	}
	if err := lacker.Ack(testAction3); err != nil {
		t.Fatal(err)
	}
	if err := lacker.Commit(); err != nil {
		t.Fatal(err)
	}

}

type actionImmediate struct {
	*fleetapi.ActionBase
	originalType string
}

// Type returns the type of the Action.
func (a *actionImmediate) Type() string {
	return "IMMEDIATE"
}

func (a *actionImmediate) ForceAck() {}

func (a *actionImmediate) String() string {
	var s strings.Builder
	s.WriteString("action_id: ")
	s.WriteString(a.ID())
	s.WriteString(", type: ")
	s.WriteString(a.Type())
	s.WriteString(" (original type: ")
	s.WriteString(a.OriginalType())
	s.WriteString(")")
	return s.String()
}

// OriginalType returns the original type of the action as returned by the API.
func (a *actionImmediate) OriginalType() string {
	return a.originalType
}
