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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/transport"
	"github.com/menderesk/beats/v7/libbeat/common/transport/transptest"
	"github.com/menderesk/beats/v7/libbeat/outputs/outest"
	v2 "github.com/menderesk/go-lumber/server/v2"
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
	enableLogging([]string{"*"})

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
	enableLogging([]string{"*"})
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
		Fields: common.MapStr{
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
	enableLogging([]string{"*"})
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
		Fields:    common.MapStr{"type": "test", "name": "me", "line": 10},
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

	//wait 10 seconds (ttl: 5 seconds) then send the event again
	time.Sleep(10 * time.Second)

	event = beat.Event{
		Timestamp: time.Now(),
		Fields:    common.MapStr{"type": "test", "name": "me", "line": 11},
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
	enableLogging([]string{"*"})
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

	event := beat.Event{Fields: common.MapStr{
		"type": "test",
		"name": "test",
		"struct": common.MapStr{
			"field1": 1,
			"field2": true,
			"field3": []int{1, 2, 3},
			"field4": []interface{}{
				1,
				"test",
				common.MapStr{
					"sub": "field",
				},
			},
			"field5": common.MapStr{
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
