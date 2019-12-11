package application

import (
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
)

type clientCallbackFunc func(method string,
	path string,
	params url.Values,
	headers http.Header,
	body io.Reader,
) (*http.Response, error)

type testingClient struct {
	sync.Mutex
	callback clientCallbackFunc
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
	return m.callback()
}

func (t *testingClient) SetCallback(fn clientCallbackFunc) {
	m.Lock()
	defer m.Unlock()
	m.callback = fn
}

type testingDispatcherFunc func(...action) error

type testingDispatcher struct {
	sync.Mutext
	callback testingDispatcherFunc
}

func (t *testingDispatcher) Dispatch(actions ...Action) error {
	t.Lock()
	defer t.Unlock()
	return t.callback(actions...)
}

func (t *testingDispatcher) SetCallback(fn testingDispatcher) {
	t.Lock()
	defer t.Unlock()
	t.callback = fn
}

func TestFleetGateway(t *testing.T) {
	agentID = "agent-secret"

	t.Run("Successfully connects send no event and receives no action", func(t *testing.T) {

		// 		scheduler := scheduler.NewStepper()
		// 		client := mockClientWithSequence{t: t}

		// 		gateway, err := newFleetGatewayWithScheduler(
		// 			nil,
		// 			Settings{period: 5 * time.Second},
		// 			agentID,
		// 			client,
		// 		)

		// 		require.NoError(t)
		// 		gateway.Start()
		// defer gateway.Stop()

		// 		// TODO:
		// 		// We could could pass the ticker instead to the instance so we actually really check the tick.
		// 		// instead of calling the.

	})

	t.Run("Successfully connects and receives a series of actions", skip)
	t.Run("Successfully connects and sends events back to fleet", skip)
	t.Run("Periodically communicates with Fleet", skip)
}

type actionDispatcher struct {
	err             error
	recordedActions chan []action
}

func (m *mockActionDispatcher) Dispatch(actions ...action) error {
	m.actions = actions
	return m.err
}
func skip(t *testing.T) {
	t.SkipNow()
}
