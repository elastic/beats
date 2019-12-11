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

	"github.com/elastic/beats/x-pack/agent/pkg/scheduler"
)

type clientCallbackFunc func(method string,
	headers http.Header,
	body io.Reader,
) (*http.Response, error)

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
	m.Lock()
	defer m.Unlock()
	return m.callback(headers, body)
}

func (t *testingClient) Answer(fn clientCallbackFunc) <-chan struct{} {
	m.Lock()
	defer m.Unlock()
	defer func() { t.received <- struct{}{} }()
	m.callback = fn
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

func (t *testingDispatcher) Dispatch(actions ...Action) error {
	t.Lock()
	defer t.Unlock()
	defer func() { t.received <- struct{}{} }()
	return t.callback(actions...)
}

func (t *testingDispatcher) Answer(fn testingDispatcher) <-chan struct{} {
	t.Lock()
	defer t.Unlock()
	t.callback = fn
}

func newTestingDispatcher() *testingDispatcher {
	return &testingDispatcher{received: make(chan struct{})}
}

type withGatewayFunc func(*fleetGateway, *testingClient, *testingDispatcher, *scheduler.Stepper)

func withGateway(fn withGatewayFunc) func(t *testing.T) {
	scheduler := scheduler.NewStepper()
	client := newTestingClient()
	dispatcher := newTestingDispatcher()

	gateway, err := newFleetGatewayWithScheduler(
		nil,
		Settings{period: 5 * time.Second},
		agentID,
		client,
		dispatcher,
		scheduler,
	)

	require.NoError(t)

	fn(gateway, client, dispatcher, scheduler)
}

func ackSeq(channels ...chan struct{}) chan struct{} {
	comm := make(chan struct{})
	go func(comm chan struct{}) {
		var c int
		for _, c := range channels {
			<-c
			c++
		}
		close(comm)
	}(comm)
	return comm
}

func wrapStrToResp(code int, body string) *http.Response {
	t := &http.Response{
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
	agentID = "agent-secret"

	t.Run("Successfully connects send no event and receives no action", withGateway(func(
		gateway *fleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Scheduler,
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
				return require.Equal(t, 0, len(actions))
			}),
		)

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
