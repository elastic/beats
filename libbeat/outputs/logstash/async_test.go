package logstash

import (
	"fmt"

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

func TestAsyncCloseAfterWindowSize(t *testing.T) {
	testCloseAfterWindowSize(t, makeAsyncTestClient)
}

func TestAsyncMultiFailMaxTimeouts(t *testing.T) {
	testMultiFailMaxTimeouts(t, makeAsyncTestClient)
}

func makeAsyncTestClient(conn TransportClient) testClientDriver {
	return newAsyncTestDriver(newAsyncTestClient(conn))
}

func newAsyncTestClient(conn TransportClient) *asyncClient {
	c, err := newAsyncLumberjackClient(conn, 3, testMaxWindowSize, 100*time.Millisecond)
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
					fmt.Printf("response: batch=%v, err=%v\n", len(events), err)

					n := len(cmd.events) - len(events)
					ret := testClientReturn{n, err}
					resp <- ret
				}

				fmt.Printf("publish events: batch=%v", len(cmd.events))
				err := driver.client.AsyncPublishEvents(cb, cmd.events)
				fmt.Println("async publish returned with: ", err)

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
