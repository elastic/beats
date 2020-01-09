// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
)

func TestAcker(t *testing.T) {
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

	if acker == nil {
		t.Fatal("acker not initialized")
	}

	testID := "ack-test-action-id"
	testAction := &fleetapi.ActionUnknown{ActionBase: &fleetapi.ActionBase{ActionID: testID}}

	ch := client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
		content, err := ioutil.ReadAll(body)
		assert.NoError(t, err)
		cr := &ackRequest{}
		err = json.Unmarshal(content, &cr)
		assert.NoError(t, err)

		assert.EqualValues(t, 1, len(cr.Events))
		ae := cr.Events[0]

		assert.EqualValues(t, "ACTION", ae.Type)
		assert.EqualValues(t, "ACKNOWLEDGED", ae.Subtype)
		assert.EqualValues(t, testID, ae.ActionID)

		resp := wrapStrToResp(http.StatusOK, `{ "actions": [], "success": true }`)
		return resp, nil
	})

	go func() {
		for range ch {
		}
	}()

	if err := acker.Ack(testAction); err != nil {
		t.Fatal(err)
	}
	if err := acker.Commit(); err != nil {
		t.Fatal(err)
	}
}
