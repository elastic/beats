// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !integration
// +build !integration

package logstash

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
)

type testAsyncDriver struct {
	client  outputs.NetworkClient
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

func makeAsyncTestClient(conn *transport.Client) testClientDriver {
	config := defaultConfig()
	config.Timeout = 1 * time.Second
	config.Pipelining = 3
	client, err := newAsyncClient(beat.Info{}, conn, outputs.NewNilObserver(), &config)
	if err != nil {
		panic(err)
	}
	return newAsyncTestDriver(client)
}

func newAsyncTestDriver(client outputs.NetworkClient) *testAsyncDriver {
	driver := &testAsyncDriver{
		client:  client,
		ch:      make(chan testDriverCommand, 1),
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
				err := driver.client.Publish(context.Background(), cmd.batch)
				driver.returns = append(driver.returns, testClientReturn{cmd.batch, err})
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

func (t *testAsyncDriver) Publish(batch *outest.Batch) {
	t.ch <- testDriverCommand{code: driverCmdPublish, batch: batch}
}

func (t *testAsyncDriver) Returns() []testClientReturn {
	return t.returns
}
