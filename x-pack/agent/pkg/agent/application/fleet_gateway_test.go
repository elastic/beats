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

	"github.com/stretchr/testify/require"

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
	return t.callback(headers, body)
}

func (t *testingClient) Answer(fn clientCallbackFunc) <-chan struct{} {
	t.Lock()
	defer t.Unlock()
	defer func() { t.received <- struct{}{} }()
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
	defer func() { t.received <- struct{}{} }()
	t.callback = fn
	return t.received
}

func newTestingDispatcher() *testingDispatcher {
	return &testingDispatcher{received: make(chan struct{})}
}

type withGatewayFunc func(*testing.T, *fleetGateway, *testingClient, *testingDispatcher, *scheduler.Stepper)

func withGateway(agentID string, fn withGatewayFunc) func(t *testing.T) {
	return func(t *testing.T) {
		scheduler := scheduler.NewStepper()
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		gateway, err := newFleetGatewayWithScheduler(
			nil,
			&fleetGatewaySettings{},
			agentID,
			client,
			dispatcher,
			scheduler,
		)

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
	agentID := "agent-secret"
	t.Run("send no event and receive no action", withGateway(agentID, func(
		t *testing.T,
		gateway *fleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
	) {
		go gateway.Start()
		defer gateway.Stop()

		received := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				// TODO: assert no events
				resp := wrapStrToResp(http.StatusOK, `{ "actions": [], "success": true, }`)
				return resp, nil
			}),

			dispatcher.Answer(func(actions ...action) error {
				require.Equal(t, 0, len(actions))
				return nil
			}),
		)

		fmt.Println("after received")
		// Synchronize scheduler and acking of calls from the worker go routine.
		scheduler.Next()
		<-received
	}))

	t.Run("Successfully connects and receives a series of actions", skip)
	t.Run("Successfully connects and sends events back to fleet", skip)
	t.Run("Periodically communicates with Fleet", skip)
}

func skip(t *testing.T) {
	t.SkipNow()
}
