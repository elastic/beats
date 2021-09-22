// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lazy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/acker/fleet"
)

func TestLazyAcker(t *testing.T) {
	type ackRequest struct {
		Events []fleetapi.AckEvent `json:"events"`
	}

	log, _ := logger.New("", false)
	client := newTestingClient()
	agentInfo := &testAgentInfo{}
	acker, err := fleet.NewAcker(log, agentInfo, client)
	if err != nil {
		t.Fatal(err)
	}

	lacker := NewAcker(acker, log)

	if acker == nil {
		t.Fatal("acker not initialized")
	}

	testID1 := "ack-test-action-id"
	testID2 := testID1 + "2"
	testID3 := testID1 + "3"
	testAction1 := &fleetapi.ActionUnknown{ActionID: testID1}
	testAction2 := &actionImmediate{ActionID: testID2}
	testAction3 := &fleetapi.ActionUnknown{ActionID: testID3}

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
			assert.EqualValues(t, testID1, cr.Events[0].ActionID)
			assert.EqualValues(t, testID2, cr.Events[1].ActionID)

		} else {
			assert.EqualValues(t, 1, len(cr.Events))
		}

		resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
		return resp, nil
	})

	go func() {
		for range ch {
		}
	}()
	c := context.Background()

	if err := lacker.Ack(c, testAction1); err != nil {
		t.Fatal(err)
	}
	if err := lacker.Ack(c, testAction1); err != nil {
		t.Fatal(err)
	}
	if err := lacker.Ack(c, testAction2); err != nil {
		t.Fatal(err)
	}
	if err := lacker.Ack(c, testAction2); err != nil {
		t.Fatal(err)
	}
	if err := lacker.Ack(c, testAction3); err != nil {
		t.Fatal(err)
	}
	if err := lacker.Ack(c, testAction3); err != nil {
		t.Fatal(err)
	}
	if err := lacker.Commit(c); err != nil {
		t.Fatal(err)
	}

}

type actionImmediate struct {
	ActionID     string
	ActionType   string
	originalType string
}

// Type returns the type of the Action.
func (a *actionImmediate) Type() string {
	return "IMMEDIATE"
}

func (a *actionImmediate) ID() string {
	return a.ActionID
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

type clientCallbackFunc func(headers http.Header, body io.Reader) (*http.Response, error)

type testingClient struct {
	sync.Mutex
	callback clientCallbackFunc
	received chan struct{}
}

func (t *testingClient) Send(
	_ context.Context,
	method string,
	path string,
	params url.Values,
	headers http.Header,
	body io.Reader,
) (*http.Response, error) {
	t.Lock()
	defer t.Unlock()
	defer func() { t.received <- struct{}{} }()
	return t.callback(headers, body)
}

func (t *testingClient) URI() string {
	return "http://localhost"
}

func (t *testingClient) Answer(fn clientCallbackFunc) <-chan struct{} {
	t.Lock()
	defer t.Unlock()
	t.callback = fn
	return t.received
}

func newTestingClient() *testingClient {
	return &testingClient{received: make(chan struct{}, 1)}
}

type testAgentInfo struct{}

func (testAgentInfo) AgentID() string { return "agent-secret" }

func wrapStrToResp(code int, body string) *http.Response {
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", code, http.StatusText(code)),
		StatusCode:    code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          ioutil.NopCloser(bytes.NewBufferString(body)),
		ContentLength: int64(len(body)),
		Header:        make(http.Header),
	}
}
