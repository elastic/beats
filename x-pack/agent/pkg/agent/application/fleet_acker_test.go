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
)

func TestAcker(t *testing.T) {

	type ackRequest struct {
		Events []actionEvent `json:"events"`
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
	ch := client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
		content, err := ioutil.ReadAll(body)
		assert.NoError(t, err)

		cr := &ackRequest{}
		err = json.Unmarshal(content, &cr)
		assert.NoError(t, err)

		assert.EqualValues(t, 1, len(cr.Events))
		ae := cr.Events[0]

		assert.EqualValues(t, EventTypeAction, ae.Typ)
		assert.EqualValues(t, EventSubtypeACK, ae.Subtype)
		assert.EqualValues(t, testID, ae.ActionID)

		resp := wrapStrToResp(http.StatusOK, `{ "actions": [], "success": true }`)
		return resp, nil
	})

	go func() { <-ch }()

	if err := acker.Ack(testID); err != nil {
		t.Fatal(err)
	}

}
