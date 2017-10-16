// +build !integration

package logstash

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type testAsyncDriver struct {
	client  mode.AsyncProtocolClient
	ch      chan testDriverCommand
	returns []testClientReturn
	wg      sync.WaitGroup
}

func TestAsync(t *testing.T) {
	tests := []struct {
		name   string
		runner func(*testing.T, clientFactory)
	}{
		{"sendZero", testSendZero},
		{"simpleEvent", testSimpleEvent},
		{"structuredEvent", testStructuredEvent},
		{"multiFailMaxTimeouts", testMultiFailMaxTimeouts},
	}

	settings := []map[string]interface{}{
		nil,
		map[string]interface{}{
			"slow_start": false,
		},
		map[string]interface{}{
			"slow_start": true,
		},
		map[string]interface{}{
			"slow_start":    true,
			"pipelining":    5,
			"bulk_max_size": 8,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, s := range settings {
				s := s
				t.Run(fmt.Sprintf("%v", s), func(t *testing.T) {
					test.runner(t, makeAsyncTestClient(s))
				})
			}
		})
	}
}

func makeAsyncTestClient(settings map[string]interface{}) func(*transport.Client) testClientDriver {
	return func(conn *transport.Client) testClientDriver {
		return newAsyncTestDriver(newAsyncTestClient(conn, settings))
	}
}

func newAsyncTestClient(conn *transport.Client, settings map[string]interface{}) *asyncClient {
	config, err := common.NewConfigFrom(settings)
	if err != nil {
		panic(err)
	}

	lsCfg := defaultConfig
	lsCfg.Index = "testbeat"
	lsCfg.BulkMaxSize = testMaxWindowSize
	lsCfg.Timeout = 100 * time.Millisecond
	lsCfg.Pipelining = 2
	lsCfg.SlowStart = true
	if err := config.Unpack(&lsCfg); err != nil {
		panic(err)
	}

	c, err := newAsyncLumberjackClient(conn, &lsCfg)
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
				cb := func(data []outputs.Data, err error) {
					n := len(cmd.data) - len(data)
					ret := testClientReturn{n, err}
					resp <- ret
				}

				err := driver.client.AsyncPublishEvents(cb, cmd.data)
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

func (t *testAsyncDriver) Publish(data []outputs.Data) {
	t.ch <- testDriverCommand{code: driverCmdPublish, data: data}
}

func (t *testAsyncDriver) Returns() []testClientReturn {
	return t.returns
}
