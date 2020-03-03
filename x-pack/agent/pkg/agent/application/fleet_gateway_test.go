// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
	repo "github.com/elastic/beats/v7/x-pack/agent/pkg/reporter"
	fleetreporter "github.com/elastic/beats/v7/x-pack/agent/pkg/reporter/fleet"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/scheduler"
)

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
	return &testingClient{received: make(chan struct{})}
}

type testingDispatcherFunc func(...action) error

type testingDispatcher struct {
	sync.Mutex
	callback testingDispatcherFunc
	received chan struct{}
}

func (t *testingDispatcher) Dispatch(acker fleetAcker, actions ...action) error {
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

type withGatewayFunc func(*testing.T, *fleetGateway, *testingClient, *testingDispatcher, *scheduler.Stepper, repo.Backend)

func withGateway(agentInfo agentInfo, settings *fleetGatewaySettings, fn withGatewayFunc) func(t *testing.T) {
	return func(t *testing.T) {
		scheduler := scheduler.NewStepper()
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		log, _ := logger.New()
		rep := getReporter(agentInfo, log, t)

		gateway, err := newFleetGatewayWithScheduler(
			context.Background(),
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			rep,
			newNoopAcker(),
		)

		go gateway.Start()
		defer gateway.Stop()

		require.NoError(t, err)

		fn(t, gateway, client, dispatcher, scheduler, rep)
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
	settings := &fleetGatewaySettings{
		Duration: 5 * time.Second,
		Backoff:  backoffSettings{Init: 1 * time.Second, Max: 5 * time.Second},
	}

	t.Run("send no event and receive no action", withGateway(agentInfo, settings, func(
		t *testing.T,
		gateway *fleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
		rep repo.Backend,
	) {
		received := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
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

	t.Run("Successfully connects and receives a series of actions", withGateway(agentInfo, settings, func(
		t *testing.T,
		gateway *fleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
		rep repo.Backend,
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
			context.Background(),
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			getReporter(agentInfo, log, t),
			newNoopAcker(),
		)

		go gateway.Start()
		defer gateway.Stop()

		require.NoError(t, err)

		var count int
		for {
			received := ackSeq(
				client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
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

	t.Run("send event and receive no action", withGateway(agentInfo, settings, func(
		t *testing.T,
		gateway *fleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
		rep repo.Backend,
	) {
		rep.Report(context.Background(), &testStateEvent{})
		received := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				cr := &request{}
				content, err := ioutil.ReadAll(body)
				if err != nil {
					t.Fatal(err)
				}
				err = json.Unmarshal(content, &cr)
				if err != nil {
					t.Fatal(err)
				}

				require.Equal(t, 1, len(cr.Events))

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

	t.Run("Test the wait loop is interruptible", func(t *testing.T) {
		d := 10 * time.Minute
		scheduler := scheduler.NewPeriodic(d)
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		log, _ := logger.New()
		gateway, err := newFleetGatewayWithScheduler(
			context.Background(),
			log,
			&fleetGatewaySettings{
				Duration: d,
				Backoff:  backoffSettings{Init: 1 * time.Second, Max: 30 * time.Second},
			},
			agentInfo,
			client,
			dispatcher,
			scheduler,
			getReporter(agentInfo, log, t),
			newNoopAcker(),
		)

		go gateway.Start()
		defer gateway.Stop()

		require.NoError(t, err)

		// Silently dispatch action.
		go func() { <-dispatcher.Answer(func(actions ...action) error { return nil }) }()

		// Make sure that all API calls to the checkin API are successfull, the following will happen:
		// 1. Gateway -> checking api.
		// 2. WaitTick() will block for 10 minutes.
		// 3. Stop will unblock the Wait.
		<-client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
			resp := wrapStrToResp(http.StatusOK, `{ "actions": [], "success": true }`)
			return resp, nil
		})
	})
}

func TestRetriesOnFailures(t *testing.T) {
	agentInfo := &testAgentInfo{}
	settings := &fleetGatewaySettings{
		Duration: 5 * time.Second,
		Backoff:  backoffSettings{Init: 1 * time.Second, Max: 5 * time.Second},
	}

	t.Run("When the gateway fails to communicate with the checkin API we will retry",
		withGateway(agentInfo, settings, func(
			t *testing.T,
			gateway *fleetGateway,
			client *testingClient,
			dispatcher *testingDispatcher,
			scheduler *scheduler.Stepper,
			rep repo.Backend,
		) {
			rep.Report(context.Background(), &testStateEvent{})

			fail := func(_ http.Header, _ io.Reader) (*http.Response, error) {
				return wrapStrToResp(http.StatusInternalServerError, "something is bad"), nil
			}

			// Initial tick is done out of bound so we can block on channels.
			go scheduler.Next()

			// Simulate a 500 errors for the next 3 calls.
			<-client.Answer(fail)
			<-client.Answer(fail)
			<-client.Answer(fail)

			// API recover
			received := ackSeq(
				client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
					cr := &request{}
					content, err := ioutil.ReadAll(body)
					if err != nil {
						t.Fatal(err)
					}
					err = json.Unmarshal(content, &cr)
					if err != nil {
						t.Fatal(err)
					}

					require.Equal(t, 1, len(cr.Events))

					resp := wrapStrToResp(http.StatusOK, `{ "actions": [], "success": true }`)
					return resp, nil
				}),

				dispatcher.Answer(func(actions ...action) error {
					require.Equal(t, 0, len(actions))
					return nil
				}),
			)

			<-received
		}))

	t.Run("The retry loop is interruptible",
		withGateway(agentInfo, &fleetGatewaySettings{
			Duration: 0 * time.Second,
			Backoff:  backoffSettings{Init: 10 * time.Minute, Max: 20 * time.Minute},
		}, func(
			t *testing.T,
			gateway *fleetGateway,
			client *testingClient,
			dispatcher *testingDispatcher,
			scheduler *scheduler.Stepper,
			rep repo.Backend,
		) {
			rep.Report(context.Background(), &testStateEvent{})

			fail := func(_ http.Header, _ io.Reader) (*http.Response, error) {
				return wrapStrToResp(http.StatusInternalServerError, "something is bad"), nil
			}

			// Initial tick is done out of bound so we can block on channels.
			go scheduler.Next()

			// Fail to enter retry loop, all other calls will fails and will force to wait on big initial
			// delay.
			<-client.Answer(fail)

			// non-obvious but withGateway on return will stop the gateway before returning and we should
			// exit the retry loop. The init value of the backoff is set to exceed the test default timeout.
		}))
}

func getReporter(info agentInfo, log *logger.Logger, t *testing.T) *fleetreporter.Reporter {
	fleetR, err := fleetreporter.NewReporter(info, log, fleetreporter.DefaultFleetManagementConfig())
	if err != nil {
		t.Fatal(errors.Wrap(err, "fail to create reporters"))
	}

	return fleetR
}

type testAgentInfo struct{}

func (testAgentInfo) AgentID() string { return "agent-secret" }

type testStateEvent struct{}

func (testStateEvent) Type() string                    { return repo.EventTypeState }
func (testStateEvent) SubType() string                 { return repo.EventSubTypeInProgress }
func (testStateEvent) Time() time.Time                 { return time.Unix(0, 1) }
func (testStateEvent) Message() string                 { return "hello" }
func (testStateEvent) Payload() map[string]interface{} { return map[string]interface{}{"key": 1} }

type request struct {
	Events []interface{} `json:"events"`
}
