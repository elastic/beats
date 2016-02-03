package logstash

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/mode"
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

func makeAsyncTestClient(conn TransportClient) testClient {
	return newAsyncTestDriver(newAsyncTestClient(conn))
}

func newAsyncTestClient(conn TransportClient) *asyncClient {
	c, err := newAsyncLumberjackClient(conn, 3, testMaxWindowSize, 5*time.Second)
	if err != nil {
		panic(err)
	}
	return c
}

func newAsyncTestDriver(client mode.AsyncProtocolClient) *testAsyncDriver {
	driver := &testAsyncDriver{
		client:  client,
		ch:      make(chan testDriverCommand, 1),
		returns: nil,
	}

	client.Connect(100 * time.Millisecond)

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
			case driverCmdPublish:
				cb := func(events []common.MapStr, err error) {
					n := len(cmd.events) - len(events)
					resp <- testClientReturn{n, err}
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

func (t *testAsyncDriver) Stop() {
	t.ch <- testDriverCommand{code: driverCmdQuit}
	t.wg.Wait()
	close(t.ch)
}

func (t *testAsyncDriver) Publish(events []common.MapStr) {
	t.ch <- testDriverCommand{code: driverCmdPublish, events: events}
}

func (t *testAsyncDriver) Returns() []testClientReturn {
	return t.returns
}
