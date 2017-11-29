// +build !integration

package logstash

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/outest"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/libbeat/outputs/transport/transptest"
)

type testSyncDriver struct {
	client  outputs.NetworkClient
	ch      chan testDriverCommand
	returns []testClientReturn
	wg      sync.WaitGroup
}

type clientServer struct {
	*transptest.MockServer
}

func TestClientSendZero(t *testing.T) {
	testSendZero(t, makeTestClient)
}

func TestClientSimpleEvent(t *testing.T) {
	testSimpleEvent(t, makeTestClient)
}

func TestClientSimpleEventTTL(t *testing.T) {
	testSimpleEventWithTTL(t, makeTestClient)
}

func TestClientStructuredEvent(t *testing.T) {
	testStructuredEvent(t, makeTestClient)
}

func newClientServerTCP(t *testing.T, to time.Duration) *clientServer {
	return &clientServer{transptest.NewMockServerTCP(t, to, "", nil)}
}

func makeTestClient(conn *transport.Client) testClientDriver {
	config := defaultConfig
	config.Timeout = 1 * time.Second
	config.TTL = 5 * time.Second
	client, err := newSyncClient(beat.Info{}, conn, outputs.NewNilObserver(), &config)
	if err != nil {
		panic(err)
	}

	return newClientTestDriver(client)
}

func newClientTestDriver(client outputs.NetworkClient) *testSyncDriver {
	driver := &testSyncDriver{
		client:  client,
		ch:      make(chan testDriverCommand),
		returns: nil,
	}

	driver.wg.Add(1)
	go func() {
		defer driver.wg.Done()

		for {
			cmd, ok := <-driver.ch
			if !ok {
				return
			}

			switch cmd.code {
			case driverCmdQuit:
				return
			case driverCmdConnect:
				driver.client.Connect()
			case driverCmdClose:
				driver.client.Close()
			case driverCmdPublish:
				err := driver.client.Publish(cmd.batch)
				driver.returns = append(driver.returns, testClientReturn{cmd.batch, err})
			}
		}
	}()

	return driver
}

func (t *testSyncDriver) Stop() {
	if t.ch != nil {
		t.ch <- testDriverCommand{code: driverCmdQuit}
		t.wg.Wait()
		close(t.ch)
		t.client.Close()
		t.ch = nil
	}
}

func (t *testSyncDriver) Connect() {
	t.ch <- testDriverCommand{code: driverCmdConnect}
}

func (t *testSyncDriver) Close() {
	t.ch <- testDriverCommand{code: driverCmdClose}
}

func (t *testSyncDriver) Publish(batch *outest.Batch) {
	t.ch <- testDriverCommand{code: driverCmdPublish, batch: batch}
}

func (t *testSyncDriver) Returns() []testClientReturn {
	return t.returns
}
