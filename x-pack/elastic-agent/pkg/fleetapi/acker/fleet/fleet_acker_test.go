// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	repo "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter"
)

func TestAcker(t *testing.T) {
	type ackRequest struct {
		Events []fleetapi.AckEvent `json:"events"`
	}

	log, _ := logger.New("fleet_acker")
	client := newTestingClient()
	agentInfo := &testAgentInfo{}
	acker, err := NewAcker(log, agentInfo, client)
	if err != nil {
		t.Fatal(err)
	}

	if acker == nil {
		t.Fatal("acker not initialized")
	}

	testID := "ack-test-action-id"
	testAction := &fleetapi.ActionUnknown{ActionID: testID}

	ch := client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
		content, err := ioutil.ReadAll(body)
		assert.NoError(t, err)
		cr := &ackRequest{}
		err = json.Unmarshal(content, &cr)
		assert.NoError(t, err)

		assert.EqualValues(t, 1, len(cr.Events))
		assert.EqualValues(t, testID, cr.Events[0].ActionID)

		resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
		return resp, nil
	})

	go func() {
		for range ch {
		}
	}()

	if err := acker.Ack(context.Background(), testAction); err != nil {
		t.Fatal(err)
	}
	if err := acker.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
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

type testStateEvent struct{}

func (testStateEvent) Type() string                    { return repo.EventTypeState }
func (testStateEvent) SubType() string                 { return repo.EventSubTypeInProgress }
func (testStateEvent) Time() time.Time                 { return time.Unix(0, 1) }
func (testStateEvent) Message() string                 { return "hello" }
func (testStateEvent) Payload() map[string]interface{} { return map[string]interface{}{"key": 1} }

func wrapStrToResp(code int, body string) *http.Response {
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", code, http.StatusText(code)),
		StatusCode:    code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          ioutil.NopCloser(bytes.NewBufferString(body)),
		ContentLength: int64(len(body)),
		Header:        make(http.Header, 0),
	}
}
