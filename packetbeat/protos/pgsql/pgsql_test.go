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

package pgsql

import (
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/publish"
)

type eventStore struct {
	events []beat.Event
}

func (e *eventStore) publish(event beat.Event) {
	publish.MarshalPacketbeatFields(&event, nil, nil)
	e.events = append(e.events, event)
}

func pgsqlModForTests(store *eventStore) *pgsqlPlugin {
	callback := func(beat.Event) {}
	if store != nil {
		callback = store.publish
	}

	var pgsql pgsqlPlugin
	config := defaultConfig
	pgsql.init(callback, procs.ProcessesWatcher{}, &config)
	return &pgsql
}

// Test parsing a request with a single query
func TestPgsqlParser_simpleRequest(t *testing.T) {
	pgsql := pgsqlModForTests(nil)

	data := []byte(
		"510000001a53454c454354202a2046524f4d20466f6f6261723b00")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &pgsqlStream{data: message, message: new(pgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if !complete {
		t.Error("Expecting a complete message")
	}
	if !stream.message.isRequest {
		t.Error("Failed to parse postgres request")
	}
	if stream.message.query != "SELECT * FROM Foobar;" {
		t.Error("Failed to parse query")
	}
	if stream.message.size != 27 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}

// Test parsing a response with data attached
func TestPgsqlParser_dataResponse(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("pgsql", "pgsqldetailed"))

	pgsql := pgsqlModForTests(nil)
	data := []byte(
		"5400000033000269640000008fc40001000000170004ffffffff000076616c75650000008fc4000200000019ffffffffffff0000" +
			"44000000130002000000013100000004746f746f" +
			"440000001500020000000133000000066d617274696e" +
			"440000001300020000000134000000046a65616e" +
			"430000000b53454c45435400" +
			"5a0000000549")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &pgsqlStream{data: message, message: new(pgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if !complete {
		t.Error("Expecting a complete message")
	}
	if stream.message.isRequest {
		t.Error("Failed to parse postgres response")
	}
	if !stream.message.isOK || stream.message.isError {
		t.Error("Failed to parse postgres response")
	}
	if stream.message.numberOfFields != 2 {
		t.Error("Failed to parse the number of field")
	}
	if stream.message.numberOfRows != 3 {
		t.Error("Failed to parse the number of rows")
	}

	if stream.message.size != 126 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}

// Test parsing a pgsql response
func TestPgsqlParser_response(t *testing.T) {
	pgsql := pgsqlModForTests(nil)
	data := []byte(
		"54000000420003610000004009000100000413ffffffffffff0000620000004009000200000413ffffffffffff0000630000004009000300000413ffffffffffff0000" +
			"440000001b0003000000036d6561000000036d6562000000036d6563" +
			"440000001e0003000000046d656131000000046d656231000000046d656331" +
			"440000001e0003000000046d656132000000046d656232000000046d656332" +
			"440000001e0003000000046d656133000000046d656233000000046d656333" +
			"430000000d53454c454354203400" +
			"5a0000000549")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &pgsqlStream{data: message, message: new(pgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if !complete {
		t.Error("Expecting a complete message")
	}
	if stream.message.isRequest {
		t.Error("Failed to parse postgres response")
	}
	if !stream.message.isOK || stream.message.isError {
		t.Error("Failed to parse postgres response")
	}
	if stream.message.numberOfFields != 3 {
		t.Error("Failed to parse the number of field")
	}
	if stream.message.numberOfRows != 4 {
		t.Error("Failed to parse the number of rows")
	}

	if stream.message.size != 202 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}

// Test parsing an incomplete pgsql response
func TestPgsqlParser_incomplete_response(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("pgsql", "pgsqldetailed"))
	pgsql := pgsqlModForTests(nil)

	data := []byte(
		"54000000420003610000004009000100000413ffffffffffff0000620000004009000200000413ffffffffffff0000630000004009000300000413ffffffffffff0000" +
			"440000001b0003000000036d6561000000036d6562000000036d6563" +
			"440000001e0003000000046d656131000000046d656231000000046d656331" +
			"440000001e0003000000046d")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &pgsqlStream{data: message, message: new(pgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if complete {
		t.Error("Expecting an incomplete message")
	}
}

// Test 3 responses in a row
func TestPgsqlParser_threeResponses(t *testing.T) {
	pgsql := pgsqlModForTests(nil)

	data, err := hex.DecodeString(
		"5300000017446174655374796c650049534f2c204d445900430000000853455400430000000853455400540000005700036f696400000004eefffe0000001a0004ffffffff0000656e636f64696e6700000000000000000000130040ffffffff00006461746c6173747379736f696400000004ee00090000001a0004ffffffff0000440000002000030000000531313836350000000455544638000000053131383537430000000d53454c4543542031005a0000000549")
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	ts, err := time.Parse(time.RFC3339, "2000-12-26T01:15:06+04:20")
	if err != nil {
		t.Error("Failed to get ts")
	}
	pkt := protos.Packet{
		Payload: data,
		Ts:      ts,
	}
	var tuple common.TCPTuple
	var private pgsqlPrivateData
	countHandlePgsql := 0

	pgsql.handlePgsql = func(pgsql *pgsqlPlugin, m *pgsqlMessage, tcptuple *common.TCPTuple,
		dir uint8, raw_msg []byte,
	) {
		countHandlePgsql++
	}

	pgsql.Parse(&pkt, &tuple, 1, private)

	if countHandlePgsql != 3 {
		t.Error("handlePgsql not called three times")
	}
}

// Test parsing an error response
func TestPgsqlParser_errorResponse(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("pgsql", "pgsqldetailed"))

	pgsql := pgsqlModForTests(nil)
	data := []byte(
		"4500000088534552524f5200433235503032004d63757272656e74207472616e73616374696f6e2069732061626f727465642c20636f6d6d616e64732069676e6f72656420756e74696c20656e64206f66207472616e73616374696f6e20626c6f636b0046706f7374677265732e63004c3932310052657865635f73696d706c655f71756572790000")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &pgsqlStream{data: message, message: new(pgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if !complete {
		t.Error("Expecting a complete message")
	}

	if stream.message.isRequest {
		t.Error("Failed to parse postgres response")
	}
	if !stream.message.isError {
		t.Error("Failed to parse error response")
	}
	if stream.message.errorSeverity != "ERROR" {
		t.Error("Failed to parse severity")
	}
	if stream.message.errorCode != "25P02" {
		t.Error("Failed to parse error code")
	}
	if stream.message.errorInfo != "current transaction is aborted, commands ignored until end of transaction block" {
		t.Error("Failed to parse error message")
	}
	if stream.message.size != 137 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}

// Test parsing an error response
func TestPgsqlParser_invalidMessage(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("pgsql", "pgsqldetailed"))
	pgsql := pgsqlModForTests(nil)
	data := []byte(
		"4300000002")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &pgsqlStream{data: message, message: new(pgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

	if ok {
		t.Error("Parsing returned success instead of error")
	}
	if complete {
		t.Error("Expecting a non complete message")
	}
}

func testTCPTuple() *common.TCPTuple {
	t := &common.TCPTuple{
		IPLength: 4,
		BaseTuple: common.BaseTuple{
			SrcIP: net.IPv4(192, 168, 0, 1), DstIP: net.IPv4(192, 168, 0, 2),
			SrcPort: 6512, DstPort: 5432,
		},
	}
	t.ComputeHashables()
	return t
}

// Helper function to read from the Publisher Queue
func expectTransaction(t *testing.T, e *eventStore) common.MapStr {
	if len(e.events) == 0 {
		t.Error("No transaction")
		return nil
	}

	event := e.events[0]
	e.events = e.events[1:]
	return event.Fields
}

// Test that loss of data during the response (but not at the beginning)
// don't cause the whole transaction to be dropped.
func Test_gap_in_response(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("pgsql", "pgsqldetailed"))

	store := &eventStore{}
	pgsql := pgsqlModForTests(store)

	// request and response from tests/pcaps/pgsql_request_response.pcap
	// select * from test
	reqData, err := hex.DecodeString(
		"510000001873656c656374202a20" +
			"66726f6d20746573743b00")
	assert.NoError(t, err)

	// response is incomplete
	respData, err := hex.DecodeString(
		"5400000042000361000000410900" +
			"0100000413ffffffffffff0000620000" +
			"004009000200000413ffffffffffff00" +
			"00630000004009000300000413ffffff" +
			"ffffff0000440000001b000300000003" +
			"6d6561000000036d6562000000036d65" +
			"63440000001e0003000000046d656131" +
			"000000046d656231000000046d656331" +
			"440000001e0003000000046d65613200")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	resp := protos.Packet{Payload: respData}

	private := protos.ProtocolData(new(pgsqlPrivateData))

	private = pgsql.Parse(&req, tcptuple, 0, private)
	private = pgsql.Parse(&resp, tcptuple, 1, private)

	logp.Debug("pgsql", "Now sending gap..")

	_, drop := pgsql.GapInStream(tcptuple, 1, 10, private)
	assert.Equal(t, true, drop)

	trans := expectTransaction(t, store)
	assert.NotNil(t, trans)
	if m, err := trans.GetValue("error.message"); assert.NoError(t, err) {
		assert.Equal(t, m, "Packet loss while capturing the response")
	}
}
