package logstash

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

type testSyncDriver struct {
	client  mode.ProtocolClient
	ch      chan testDriverCommand
	returns []testClientReturn
}

func TestClientSendZero(t *testing.T) {
	testSendZero(t, makeTestClient)
}

func TestClientSimpleEvent(t *testing.T) {
	testSimpleEvent(t, makeTestClient)
}

func TestClientStructuredEvent(t *testing.T) {
	testStructuredEvent(t, makeTestClient)
}

func makeTestClient(conn TransportClient) testClient {
	return newClientTestDriver(newLumberjackTestClient(conn))
}

func newClientTestDriver(client mode.ProtocolClient) *testSyncDriver {
	driver := &testSyncDriver{
		client:  client,
		ch:      make(chan testDriverCommand),
		returns: nil,
	}

	go func() {
		for {
			cmd, ok := <-driver.ch
			if !ok {
				return
			}

			switch cmd.code {
			case driverCmdQuit:
				close(driver.ch)
				return
			case driverCmdPublish:
				events, err := driver.client.PublishEvents(cmd.events)
				n := len(cmd.events) - len(events)
				driver.returns = append(driver.returns, testClientReturn{n, err})
			}
		}
	}()

	return driver
}

func (t *testSyncDriver) Stop() {
	t.ch <- testDriverCommand{code: driverCmdQuit}
}

func (t *testSyncDriver) Publish(events []common.MapStr) {
	t.ch <- testDriverCommand{code: driverCmdPublish, events: events}
}

func (t *testSyncDriver) Returns() []testClientReturn {
	return t.returns
}
