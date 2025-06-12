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

// go : bui ld !integration

package logstash

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/transport/transptest"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/outputs/outputtest"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/transport"
	v2 "github.com/elastic/go-lumber/server/v2"
)

const (
	driverCmdQuit = iota
	driverCmdPublish
	driverCmdConnect
	driverCmdClose
)

type testClientDriver interface {
	Connect()
	Close()
	Stop()
	Publish(*outest.Batch)
	Returns() []testClientReturn
}

type clientFactory func(*transport.Client) testClientDriver

type testClientReturn struct {
	batch *outest.Batch
	err   error
}

type testDriverCommand struct {
	code  int
	batch *outest.Batch
}

const testMaxWindowSize = 64

func testSendZero(t *testing.T, factory clientFactory) {

	server := transptest.NewMockServerTCP(t, 1*time.Second, "", nil)
	defer server.Close()

	sock, transp, err := server.ConnectPair()
	if err != nil {
		t.Fatalf("Failed to connect server and client: %v", err)
	}

	client := factory(transp)
	defer sock.Close()
	defer transp.Close()

	client.Publish(outest.NewBatch())

	client.Stop()
	returns := client.Returns()

	assert.Equal(t, 1, len(returns))
	if len(returns) == 1 {
		assert.Equal(t, outest.BatchACK, returns[0].batch.Signals[0].Tag)
		assert.Nil(t, returns[0].err)
	}
}

func testSimpleEvent(t *testing.T, factory clientFactory) {
	mock := transptest.NewMockServerTCP(t, 1*time.Second, "", nil)
	server, _ := v2.NewWithListener(mock.Listener)
	defer server.Close()

	transp, err := mock.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	client := factory(transp)
	defer transp.Close()
	defer client.Stop()

	event := beat.Event{
		Fields: mapstr.M{
			"name": "me",
			"line": 10,
		},
	}
	go client.Publish(outest.NewBatch(event))

	// try to receive event from server
	batch := server.Receive()
	batch.ACK()

	// validate
	events := batch.Events
	assert.Equal(t, 1, len(events))
	msg := events[0].(map[string]interface{})
	assert.Equal(t, "me", msg["name"])
	assert.Equal(t, 10.0, msg["line"])
}

func testSimpleEventWithTTL(t *testing.T, factory clientFactory) {
	mock := transptest.NewMockServerTCP(t, 1*time.Second, "", nil)
	server, _ := v2.NewWithListener(mock.Listener)
	defer server.Close()

	transp, err := mock.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	client := factory(transp)
	defer transp.Close()
	defer client.Stop()

	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    mapstr.M{"type": "test", "name": "me", "line": 10},
	}
	go client.Publish(outest.NewBatch(event))

	// try to receive event from server
	batch := server.Receive()
	batch.ACK()

	// validate
	events := batch.Events
	assert.Equal(t, 1, len(events))
	msg := events[0].(map[string]interface{})
	assert.Equal(t, "me", msg["name"])
	assert.Equal(t, 10.0, msg["line"])

	// wait 10 seconds (ttl: 5 seconds) then send the event again
	time.Sleep(10 * time.Second)

	event = beat.Event{
		Timestamp: time.Now(),
		Fields:    mapstr.M{"type": "test", "name": "me", "line": 11},
	}
	go client.Publish(outest.NewBatch(event))

	// try to receive event from server
	batch = server.Receive()
	batch.ACK()

	// validate
	events = batch.Events
	assert.Equal(t, 1, len(events))
	msg = events[0].(map[string]interface{})
	assert.Equal(t, "me", msg["name"])
	assert.Equal(t, 11.0, msg["line"])
}

func testStructuredEvent(t *testing.T, factory clientFactory) {
	mock := transptest.NewMockServerTCP(t, 1*time.Second, "", nil)
	server, _ := v2.NewWithListener(mock.Listener)
	defer server.Close()

	transp, err := mock.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	client := factory(transp)
	defer transp.Close()
	defer client.Stop()

	event := beat.Event{Fields: mapstr.M{
		"type": "test",
		"name": "test",
		"struct": mapstr.M{
			"field1": 1,
			"field2": true,
			"field3": []int{1, 2, 3},
			"field4": []interface{}{
				1,
				"test",
				mapstr.M{
					"sub": "field",
				},
			},
			"field5": mapstr.M{
				"sub1": 2,
			},
		},
	}}
	go client.Publish(outest.NewBatch(event))
	defer client.Stop()

	// try to receive event from server
	batch := server.Receive()
	batch.ACK()

	events := batch.Events
	assert.Equal(t, 1, len(events))
	msg := events[0]
	assert.Equal(t, "test", eventGet(msg, "name"))
	assert.Equal(t, 1.0, eventGet(msg, "struct.field1"))
	assert.Equal(t, true, eventGet(msg, "struct.field2"))
	assert.Equal(t, 2.0, eventGet(msg, "struct.field5.sub1"))
}

func eventGet(event interface{}, path string) interface{} {
	doc := event.(map[string]interface{})
	elems := strings.Split(path, ".")
	for i := 0; i < len(elems)-1; i++ {
		doc = doc[elems[i]].(map[string]interface{})
	}
	return doc[elems[len(elems)-1]]
}

func TestClientOutputListener(t *testing.T) {
	tests := []struct {
		name        string
		newClient   func(beat.Info, *transport.Client, outputs.Observer, *Config) (outputs.NetworkClient, error)
		expectedErr bool
	}{
		{
			name: "syncClient",
			newClient: func(bi beat.Info, tc *transport.Client, obs outputs.Observer, cfg *Config) (outputs.NetworkClient, error) {
				return newSyncClient(bi, tc, obs, cfg)
			},
			expectedErr: true,
		},
		{
			name: "asyncClient",
			newClient: func(bi beat.Info, tc *transport.Client, obs outputs.Observer, cfg *Config) (outputs.NetworkClient, error) {
				return newAsyncClient(bi, tc, obs, cfg)
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := transptest.NewMockServerTCP(t, 0, "", nil)
			lumberSrv, err := v2.NewWithListener(mock.Listener)
			defer mock.Close()

			require.NoError(t, err, "failed to create lumberjack server")
			transp, err := mock.Connect()
			require.NoError(t, err, "failed to connect to mock server")

			go func() {
				// receive and ack 1st batch
				for {
					srvBatch := lumberSrv.Receive()
					if srvBatch == nil {
						continue
					}
					srvBatch.ACK()
					break
				}
				// receive but don't ack 2nd batch
				_ = lumberSrv.Receive()
				return
			}()

			reg := monitoring.NewRegistry()
			observer := outputs.NewStats(reg)
			beatInfo := beat.Info{
				Beat: "TestClientOutputListener_" + tt.name,
				Logger: logptest.NewTestingLogger(t, "",
					// only print stacktrace for errors above ErrorLevel.
					zap.AddStacktrace(zapcore.ErrorLevel+1))}

			cfg := defaultConfig()
			cfg.Timeout = 100 * time.Millisecond

			c, err := tt.newClient(beatInfo, transp, observer, &cfg)
			require.NoError(t, err, "failed to create client")
			defer c.Close() // Ensure client is closed eventually, e.g. on panic

			counter := &beat.CountOutputListener{}
			listener := publisher.OutputListener{Listener: counter}

			batch := outest.NewBatchWithObserver(listener,
				beat.Event{Fields: mapstr.M{"message": "event 1", "outcome": "success"}})

			require.NoError(t, c.Connect(context.Background()), "client connect failed")
			require.NoError(t, c.Publish(context.Background(), batch), "first publish (batch1) failed")

			err = c.Publish(context.Background(), batch)
			if tt.expectedErr {
				assert.Error(t, err, "second publish should have failed")
			} else {
				assert.NoError(t, err, "second publish should have succeeded")
			}

			c.Close()

			// Wait for metrics to be updated asynchronously. The async client
			// reports ACKs from a different goroutine, so depending on CPU
			// scheduling the callback may not have completed by the time we
			// reach this point. Give it a little time to avoid flakes in CI.
			require.Eventually(t, func() bool {
				return counter.AckedLoad() == 1
			}, 2*time.Second, 10*time.Millisecond,
				"timed out waiting for ACK to be processed")

			outputtest.AssertOutputMetrics(t,
				outputtest.Metrics{
					Total:     2,
					Acked:     1,
					Retryable: 1,
					Batches:   2,
				},
				counter, reg)
		})
	}
}
