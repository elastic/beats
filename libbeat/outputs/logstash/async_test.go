// +build !integration

package logstash

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type testAsyncDriver struct {
	client  mode.AsyncProtocolClient
	ch      chan testDriverCommand
	returns []testClientReturn
	wg      sync.WaitGroup
}

func TestAsyncSendZero(t *testing.T) {
	testSendZero(t, makeAsyncTestClient)
}

func TestAsyncSimpleEvent(t *testing.T) {
	testSimpleEvent(t, makeAsyncTestClient)
}

func TestAsyncStructuredEvent(t *testing.T) {
	testStructuredEvent(t, makeAsyncTestClient)
}

func TestAsyncMultiFailMaxTimeouts(t *testing.T) {
	testMultiFailMaxTimeouts(t, makeAsyncTestClient)
}

func makeAsyncTestClient(conn *transport.Client) testClientDriver {
	return newAsyncTestDriver(newAsyncTestClient(conn))
}

func newAsyncTestClient(conn *transport.Client) *asyncClient {
	c, err := newAsyncLumberjackClient(conn,
		1, 3, testMaxWindowSize, 100*time.Millisecond, "testbeat")
	if err != nil {
		panic(err)
	}
	c.Connect(100 * time.Millisecond)
	return c
}

func newAsyncTestDriver(client mode.AsyncProtocolClient) *testAsyncDriver {
	driver := &testAsyncDriver{
		client:  client,
		ch:      make(chan testDriverCommand, 1),
		returns: nil,
	}

	resp := make(chan testClientReturn, 1)

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
				driver.client.Connect(1 * time.Second)
			case driverCmdClose:
				driver.client.Close()
			case driverCmdPublish:
				cb := func(events []common.MapStr, err error) {
					n := len(cmd.events) - len(events)
					ret := testClientReturn{n, err}
					resp <- ret
				}

				err := driver.client.AsyncPublishEvents(cb, cmd.events)
				if err != nil {
					driver.returns = append(driver.returns, testClientReturn{0, err})
				} else {
					r := <-resp
					driver.returns = append(driver.returns, r)
				}
			}
		}
	}()

	return driver
}

func (t *testAsyncDriver) Close() {
	t.ch <- testDriverCommand{code: driverCmdClose}
}

func (t *testAsyncDriver) Connect() {
	t.ch <- testDriverCommand{code: driverCmdConnect}
}

func (t *testAsyncDriver) Stop() {
	if t.ch != nil {
		t.ch <- testDriverCommand{code: driverCmdQuit}
		t.wg.Wait()
		close(t.ch)
		t.client.Close()
		t.ch = nil
	}
}

func (t *testAsyncDriver) Publish(events []common.MapStr) {
	t.ch <- testDriverCommand{code: driverCmdPublish, events: events}
}

func (t *testAsyncDriver) Returns() []testClientReturn {
	return t.returns
}
