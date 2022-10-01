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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/gateway"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	noopacker "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/acker/noop"
	repo "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter"
	fleetreporter "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter/fleet"
	fleetreporterConfig "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter/fleet/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/scheduler"
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
	return &testingClient{received: make(chan struct{}, 1)}
}

type testingDispatcherFunc func(...fleetapi.Action) error

type testingDispatcher struct {
	sync.Mutex
	callback testingDispatcherFunc
	received chan struct{}
}

func (t *testingDispatcher) Dispatch(acker store.FleetAcker, actions ...fleetapi.Action) error {
	t.Lock()
	defer t.Unlock()
	defer func() { t.received <- struct{}{} }()
	// Get a dummy context.
	ctx := context.Background()

	// In context of testing we need to abort on error.
	if err := t.callback(actions...); err != nil {
		return err
	}

	// Ack everything and commit at the end.
	for _, action := range actions {
		acker.Ack(ctx, action)
	}
	acker.Commit(ctx)

	return nil
}

func (t *testingDispatcher) Answer(fn testingDispatcherFunc) <-chan struct{} {
	t.Lock()
	defer t.Unlock()
	t.callback = fn
	return t.received
}

func newTestingDispatcher() *testingDispatcher {
	return &testingDispatcher{received: make(chan struct{}, 1)}
}

type withGatewayFunc func(*testing.T, gateway.FleetGateway, *testingClient, *testingDispatcher, *scheduler.Stepper, repo.Backend)

func withGateway(agentInfo agentInfo, settings *fleetGatewaySettings, fn withGatewayFunc) func(t *testing.T) {
	return func(t *testing.T) {
		scheduler := scheduler.NewStepper()
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		log, _ := logger.New("fleet_gateway", false)
		rep := getReporter(agentInfo, log, t)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		diskStore := storage.NewDiskStore(paths.AgentStateStoreFile())
		stateStore, err := store.NewStateStore(log, diskStore)
		require.NoError(t, err)

		gateway, err := newFleetGatewayWithScheduler(
			ctx,
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			rep,
			noopacker.NewAcker(),
			&noopController{},
			stateStore,
		)

		require.NoError(t, err)

		fn(t, gateway, client, dispatcher, scheduler, rep)
	}
}

func ackSeq(channels ...<-chan struct{}) func() {
	return func() {
		for _, c := range channels {
			<-c
		}
	}
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
		Header:        make(http.Header),
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
		gateway gateway.FleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
		rep repo.Backend,
	) {
		waitFn := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Equal(t, 0, len(actions))
				return nil
			}),
		)
		gateway.Start()

		// Synchronize scheduler and acking of calls from the worker go routine.
		scheduler.Next()
		waitFn()
	}))

	t.Run("Successfully connects and receives a series of actions", withGateway(agentInfo, settings, func(
		t *testing.T,
		gateway gateway.FleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
		rep repo.Backend,
	) {
		waitFn := ackSeq(
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
		]
	}
	`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Equal(t, 2, len(actions))
				return nil
			}),
		)
		gateway.Start()

		scheduler.Next()
		waitFn()
	}))

	// Test the normal time based execution.
	t.Run("Periodically communicates with Fleet", func(t *testing.T) {
		scheduler := scheduler.NewPeriodic(150 * time.Millisecond)
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log, _ := logger.New("tst", false)

		diskStore := storage.NewDiskStore(paths.AgentStateStoreFile())
		stateStore, err := store.NewStateStore(log, diskStore)
		require.NoError(t, err)

		gateway, err := newFleetGatewayWithScheduler(
			ctx,
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			getReporter(agentInfo, log, t),
			noopacker.NewAcker(),
			&noopController{},
			stateStore,
		)

		require.NoError(t, err)

		waitFn := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Equal(t, 0, len(actions))
				return nil
			}),
		)

		gateway.Start()

		var count int
		for {
			waitFn()
			count++
			if count == 4 {
				return
			}
		}
	})

	t.Run("send event and receive no action", withGateway(agentInfo, settings, func(
		t *testing.T,
		gateway gateway.FleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
		rep repo.Backend,
	) {
		rep.Report(context.Background(), &testStateEvent{})
		waitFn := ackSeq(
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

				resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Equal(t, 0, len(actions))
				return nil
			}),
		)
		gateway.Start()

		// Synchronize scheduler and acking of calls from the worker go routine.
		scheduler.Next()
		waitFn()
	}))

	t.Run("Test the wait loop is interruptible", func(t *testing.T) {
		// 20mins is the double of the base timeout values for golang test suites.
		// If we cannot interrupt we will timeout.
		d := 20 * time.Minute
		scheduler := scheduler.NewPeriodic(d)
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		ctx, cancel := context.WithCancel(context.Background())
		log, _ := logger.New("tst", false)

		diskStore := storage.NewDiskStore(paths.AgentStateStoreFile())
		stateStore, err := store.NewStateStore(log, diskStore)
		require.NoError(t, err)

		gateway, err := newFleetGatewayWithScheduler(
			ctx,
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
			noopacker.NewAcker(),
			&noopController{},
			stateStore,
		)

		require.NoError(t, err)

		ch1 := dispatcher.Answer(func(actions ...fleetapi.Action) error { return nil })
		ch2 := client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
			resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
			return resp, nil
		})

		gateway.Start()

		// Silently dispatch action.
		go func() {
			for range ch1 {
			}
		}()

		// Make sure that all API calls to the checkin API are successfull, the following will happen:

		// block on the first call.
		<-ch2

		go func() {
			// drain the channel
			for range ch2 {
			}
		}()

		// 1. Gateway will check the API on boot.
		// 2. WaitTick() will block for 20 minutes.
		// 3. Stop will should unblock the wait.
		cancel()
	})

}

func TestRetriesOnFailures(t *testing.T) {
	agentInfo := &testAgentInfo{}
	settings := &fleetGatewaySettings{
		Duration: 5 * time.Second,
		Backoff:  backoffSettings{Init: 100 * time.Millisecond, Max: 5 * time.Second},
	}

	t.Run("When the gateway fails to communicate with the checkin API we will retry",
		withGateway(agentInfo, settings, func(
			t *testing.T,
			gateway gateway.FleetGateway,
			client *testingClient,
			dispatcher *testingDispatcher,
			scheduler *scheduler.Stepper,
			rep repo.Backend,
		) {
			fail := func(_ http.Header, _ io.Reader) (*http.Response, error) {
				return wrapStrToResp(http.StatusInternalServerError, "something is bad"), nil
			}
			clientWaitFn := client.Answer(fail)
			gateway.Start()

			rep.Report(context.Background(), &testStateEvent{})

			// Initial tick is done out of bound so we can block on channels.
			scheduler.Next()

			// Simulate a 500 errors for the next 3 calls.
			<-clientWaitFn
			<-clientWaitFn
			<-clientWaitFn

			// API recover
			waitFn := ackSeq(
				client.Answer(func(_ http.Header, body io.Reader) (*http.Response, error) {
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

					resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
					return resp, nil
				}),

				dispatcher.Answer(func(actions ...fleetapi.Action) error {
					require.Equal(t, 0, len(actions))
					return nil
				}),
			)

			waitFn()
		}))

	t.Run("The retry loop is interruptible",
		withGateway(agentInfo, &fleetGatewaySettings{
			Duration: 0 * time.Second,
			Backoff:  backoffSettings{Init: 10 * time.Minute, Max: 20 * time.Minute},
		}, func(
			t *testing.T,
			gateway gateway.FleetGateway,
			client *testingClient,
			dispatcher *testingDispatcher,
			scheduler *scheduler.Stepper,
			rep repo.Backend,
		) {
			fail := func(_ http.Header, _ io.Reader) (*http.Response, error) {
				return wrapStrToResp(http.StatusInternalServerError, "something is bad"), nil
			}
			waitChan := client.Answer(fail)
			gateway.Start()

			rep.Report(context.Background(), &testStateEvent{})

			// Initial tick is done out of bound so we can block on channels.
			scheduler.Next()

			// Fail to enter retry loop, all other calls will fails and will force to wait on big initial
			// delay.
			<-waitChan

			// non-obvious but withGateway on return will stop the gateway before returning and we should
			// exit the retry loop. The init value of the backoff is set to exceed the test default timeout.
		}))
}

func getReporter(info agentInfo, log *logger.Logger, t *testing.T) *fleetreporter.Reporter {
	fleetR, err := fleetreporter.NewReporter(info, log, fleetreporterConfig.DefaultConfig())
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
