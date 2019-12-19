// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/scheduler"
)

type clientCallbackFunc func(headers http.Header, body io.Reader) (*http.Response, error)

type testingClient struct {
	sync.Mutex
	callback clientCallbackFunc
	received chan struct{}
}

func (t *testingClient) Send(
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

func (t *testingClient) Answer(fn clientCallbackFunc) <-chan struct{} {
	t.Lock()
	defer t.Unlock()
	t.callback = fn
	return t.received
}

func newTestingClient() *testingClient {
	return &testingClient{received: make(chan struct{})}
}

type testingDispatcherFunc func(...action) error

type testingDispatcher struct {
	sync.Mutex
	callback testingDispatcherFunc
	received chan struct{}
}

func (t *testingDispatcher) Dispatch(actions ...action) error {
	t.Lock()
	defer t.Unlock()
	defer func() { t.received <- struct{}{} }()
	return t.callback(actions...)
}

func (t *testingDispatcher) Answer(fn testingDispatcherFunc) <-chan struct{} {
	t.Lock()
	defer t.Unlock()
	t.callback = fn
	return t.received
}

func newTestingDispatcher() *testingDispatcher {
	return &testingDispatcher{received: make(chan struct{})}
}

type withGatewayFunc func(*testing.T, *fleetGateway, *testingClient, *testingDispatcher, *scheduler.Stepper)

func withGateway(agentInfo agentInfo, fn withGatewayFunc) func(t *testing.T) {
	return func(t *testing.T) {
		scheduler := scheduler.NewStepper()
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		log, _ := logger.New()

		gateway, err := newFleetGatewayWithScheduler(
			log,
			&fleetGatewaySettings{},
			agentInfo,
			client,
			dispatcher,
			scheduler,
		)

		go gateway.Start()
		defer gateway.Stop()

		require.NoError(t, err)

		fn(t, gateway, client, dispatcher, scheduler)
	}
}

func ackSeq(channels ...<-chan struct{}) <-chan struct{} {
	comm := make(chan struct{})
	go func(comm chan struct{}) {
		for _, c := range channels {
			<-c
		}
		comm <- struct{}{}
	}(comm)
	return comm
}

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

func TestFleetGateway(t *testing.T) {
	agentInfo := &testAgentInfo{}
	t.Run("send no event and receive no action", withGateway(agentInfo, func(
		t *testing.T,
		gateway *fleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
	) {
		received := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				// TODO: assert no events
				resp := wrapStrToResp(http.StatusOK, `{ "actions": [], "success": true }`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...action) error {
				require.Equal(t, 0, len(actions))
				return nil
			}),
		)

		// Synchronize scheduler and acking of calls from the worker go routine.
		scheduler.Next()
		<-received
	}))

	t.Run("Successfully connects and receives a series of actions", withGateway(agentInfo, func(
		t *testing.T,
		gateway *fleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
	) {
		received := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				// TODO: assert no events
				resp := wrapStrToResp(http.StatusOK, `
{
    "actions": [
        {
            "type": "POLICY_CHANGE",
            "id": "id1",
            "data": {
                "policy": {
                    "id": "policy-id"
                }
            }
        },
        {
            "type": "ANOTHER_ACTION",
            "id": "id2"
        }
    ],
    "success": true
}
`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...action) error {
				require.Equal(t, 2, len(actions))
				return nil
			}),
		)

		scheduler.Next()
		<-received
	}))

	// Test the normal time based execution.
	t.Run("Periodically communicates with Fleet", func(t *testing.T) {
		scheduler := scheduler.NewPeriodic(1 * time.Second)
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		log, _ := logger.New()
		gateway, err := newFleetGatewayWithScheduler(
			log,
			&fleetGatewaySettings{},
			agentInfo,
			client,
			dispatcher,
			scheduler,
		)

		go gateway.Start()
		defer gateway.Stop()

		require.NoError(t, err)

		var count int
		for {
			received := ackSeq(
				client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
					// TODO: assert no events
					resp := wrapStrToResp(http.StatusOK, `{ "actions": [], "success": true }`)
					return resp, nil
				}),
				dispatcher.Answer(func(actions ...action) error {
					require.Equal(t, 0, len(actions))
					return nil
				}),
			)

			<-received
			count++
			if count == 5 {
				return
			}
		}
	})

	t.Run("Successfully connects and sends events back to fleet", skip)
}

func skip(t *testing.T) {
	t.SkipNow()
}

type testAgentInfo struct{}

func (testAgentInfo) AgentID() string { return "agent-secret" }
