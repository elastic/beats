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

// +build !integration

package orientdb

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
)

type eventStore struct {
	events []beat.Event
}

func (e *eventStore) publish(event beat.Event) {
	e.events = append(e.events, event)
}

func orientdbModForTests() (*eventStore, *orientdbPlugin) {
	var orientdb orientdbPlugin
	results := &eventStore{}
	config := defaultConfig
	orientdb.init(results.publish, &config)
	return results, &orientdb
}

func testTCPTuple() *common.TCPTuple {
	t := &common.TCPTuple{
		IPLength: 4,
		BaseTuple: common.BaseTuple{
			SrcIP: net.IPv4(192, 168, 0, 1), DstIP: net.IPv4(192, 168, 0, 2),
			SrcPort: 12424, DstPort: 2424,
		},
	}
	t.ComputeHashables()
	return t
}

func expectTransaction(t *testing.T, e *eventStore) common.MapStr {
	if len(e.events) == 0 {
		t.Errorf("No transaction")
		return nil
	}

	event := e.events[0]
	e.events = e.events[1:]
	return event.Fields
}

func TestSimpleReadRecord(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("orientdb", "orientdbdetailed"))

	results, orientdb := orientdbModForTests()

	reqData := []byte{30, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 3, 42, 58, 48, 0, 0}

	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}

	private := protos.ProtocolData(new(orientdbConnectionData))

	private = orientdb.Parse(&req, tcpTuple, 0, private)
	orientdb.Parse(&req, tcpTuple, 1, private)
	trans := expectTransaction(t, results)

	assert.Equal(t, "OK", trans["status"])
	assert.Equal(t, "recordLoad", trans["method"])
	assert.Equal(t, "orientdb", trans["type"])

	logp.Debug("orientdb", "Trans: %v", trans)
}
