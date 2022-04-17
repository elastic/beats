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

package amqp

import (
	"encoding/hex"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"

	"github.com/menderesk/beats/v7/packetbeat/procs"
	"github.com/menderesk/beats/v7/packetbeat/protos"
	"github.com/menderesk/beats/v7/packetbeat/publish"
)

type eventStore struct {
	events []beat.Event
}

func (e *eventStore) publish(event beat.Event) {
	publish.MarshalPacketbeatFields(&event, nil, nil)
	e.events = append(e.events, event)
}

func amqpModForTests() (*eventStore, *amqpPlugin) {
	var amqp amqpPlugin
	results := &eventStore{}
	config := defaultConfig
	amqp.init(results.publish, procs.ProcessesWatcher{}, &config)
	return results, &amqp
}

func testTCPTuple() *common.TCPTuple {
	t := &common.TCPTuple{
		IPLength: 4,
		BaseTuple: common.BaseTuple{
			SrcIP: net.IPv4(192, 168, 0, 1), DstIP: net.IPv4(192, 168, 0, 2),
			SrcPort: 6512, DstPort: 3306,
		},
	}
	t.ComputeHashables()
	return t
}

func expectTransaction(t *testing.T, e *eventStore) common.MapStr {
	if len(e.events) == 0 {
		t.Error("No transaction")
		return nil
	}

	event := e.events[0]
	e.events = e.events[1:]
	return event.Fields
}

func TestAmqp_UnknownMethod(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data, err := hex.DecodeString("0100010000000f006e000c0000075465737447657401ce")
	assert.NoError(t, err)
	stream := &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)

	if ok {
		t.Errorf("Parsing should return error")
	}
	if complete {
		t.Errorf("Message should not be complete")
	}
}

func TestAmqp_FrameSize(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	// incomplete frame
	data, err := hex.DecodeString("0100000000000c000a001fffff000200")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)

	if !ok {
		t.Errorf("Parsing should not raise an error")
	}
	if complete {
		t.Errorf("message should not be complete")
	}
}

// Test that the parser doesn't panic on a partial message that includes
// a client header
func TestAmqp_PartialFrameSize(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	// incomplete frame
	data, err := hex.DecodeString("414d515000060606010000000000")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)

	if !ok {
		t.Errorf("Parsing should not raise an error")
	}
	if complete {
		t.Errorf("message should not be complete")
	}
}

func TestAmqp_WrongShortStringSize(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data, err := hex.DecodeString("02000100000019003c000000000000000000058000ac" +
		"746578742f706c61696ece")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}
	ok, _ := amqp.amqpMessageParser(stream)

	if ok {
		t.Errorf("Parsing failed to detect error")
	}
}

func TestAmqp_QueueDeclaration(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data, err := hex.DecodeString("0100010000001a0032000a00000e5468697320697" +
		"3206120544553541800000000ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}

	m := stream.message
	ok, complete := amqp.amqpMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	assert.Equal(t, "This is a TEST", m.fields["queue"])
	assert.Equal(t, false, m.fields["passive"])
	assert.Equal(t, false, m.fields["durable"])
	assert.Equal(t, false, m.fields["exclusive"])
	assert.Equal(t, true, m.fields["auto-delete"])
	assert.Equal(t, true, m.fields["no-wait"])
	_, exists := m.fields["arguments"].(common.MapStr)
	if exists {
		t.Errorf("Arguments field should not be present")
	}
}

func TestAmqp_ExchangeDeclaration(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data, err := hex.DecodeString("0100010000001c0028000a00000a6c6f67735f746f7" +
		"0696305746f7069630200000000ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}

	m := stream.message
	ok, complete := amqp.amqpMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	assert.Equal(t, "exchange.declare", m.method)
	assert.Equal(t, "logs_topic", m.fields["exchange"])
	assert.Equal(t, "logs_topic", m.request)
	assert.Equal(t, true, m.fields["durable"])
	assert.Equal(t, false, m.fields["passive"])
	assert.Equal(t, false, m.fields["no-wait"])
	assert.Equal(t, "topic", m.fields["exchange-type"])
	_, exists := m.fields["arguments"].(common.MapStr)
	if exists {
		t.Errorf("Arguments field should not be present")
	}
}

func TestAmqp_BasicConsume(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data, err := hex.DecodeString("01000100000028003c001400000e4957616e74" +
		"546f436f6e73756d650d6d6973746572436f6e73756d650300000000ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}

	m := stream.message
	ok, complete := amqp.amqpMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	assert.Equal(t, "basic.consume", m.method)
	assert.Equal(t, "IWantToConsume", m.fields["queue"])
	assert.Equal(t, "misterConsume", m.fields["consumer-tag"])
	assert.Equal(t, true, m.fields["no-ack"])
	assert.Equal(t, false, m.fields["exclusive"])
	assert.Equal(t, true, m.fields["no-local"])
	assert.Equal(t, false, m.fields["no-wait"])
	_, exists := m.fields["arguments"].(common.MapStr)
	if exists {
		t.Errorf("Arguments field should not be present")
	}
}

func TestAmqp_ExchangeDeletion(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data, err := hex.DecodeString("010001000000100028001400000844656c65746" +
		"54d6501ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}

	m := stream.message
	ok, complete := amqp.amqpMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	assert.Equal(t, "exchange.delete", m.method)
	assert.Equal(t, "DeleteMe", m.fields["exchange"])
	assert.Equal(t, "DeleteMe", m.request)
	assert.Equal(t, true, m.fields["if-unused"])
	assert.Equal(t, false, m.fields["no-wait"])
}

// this method is exclusive to RabbitMQ
func TestAmqp_ExchangeBind(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data, err := hex.DecodeString("0100010000001c0028001e0000057465737431" +
		"057465737432044d5346540000000000ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}

	m := stream.message
	ok, complete := amqp.amqpMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	assert.Equal(t, "exchange.bind", m.method)
	assert.Equal(t, "test1", m.fields["destination"])
	assert.Equal(t, "test2", m.fields["source"])
	assert.Equal(t, "MSFT", m.fields["routing-key"])
	assert.Equal(t, "test2 test1", m.request)
	assert.Equal(t, false, m.fields["no-wait"])
	_, exists := m.fields["arguments"].(common.MapStr)
	if exists {
		t.Errorf("Arguments field should not be present")
	}
}

// this method is exclusive to RabbitMQ
func TestAmqp_ExchangeUnbindTransaction(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendRequest = true

	data, err := hex.DecodeString("0100010000001c00280028000005746573743105" +
		"7465737432044d5346540000000000ce")
	assert.NoError(t, err)
	data2, err := hex.DecodeString("0100010000000400280033ce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data2}
	amqp.Parse(&req, tcptuple, 1, private)

	trans := expectTransaction(t, results)
	assert.Equal(t, "exchange.unbind", trans["method"])
	assert.Equal(t, "exchange.unbind test2 test1", trans["request"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, common.OK_STATUS, trans["status"])
	fields, ok := trans["amqp"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, "test1", fields["destination"])
	assert.Equal(t, "test2", fields["source"])
	assert.Equal(t, "MSFT", fields["routing-key"])
	assert.Equal(t, false, fields["no-wait"])
}

func TestAmqp_PublishMessage(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendRequest = true

	data, err := hex.DecodeString("0100010000001b003c002800000a6c6f67735f746f70" +
		"696308414d51507465737400ce")
	assert.NoError(t, err)
	data2, err := hex.DecodeString("02000100000019003c0000000000000000001c800" +
		"00a746578742f706c61696ece")
	assert.NoError(t, err)
	data3, err := hex.DecodeString("0300010000001c48656c6c6f204461726c696e67" +
		"2049276d20686f6d6520616761696ece")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))

	// method frame
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data2}
	// header frame
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data3}
	// body frame
	amqp.Parse(&req, tcptuple, 0, private)

	trans := expectTransaction(t, results)

	body := "Hello Darling I'm home again"

	assert.Equal(t, "basic.publish", trans["method"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, body, trans["request"])
	assert.Equal(t, common.OK_STATUS, trans["status"])
	fields, ok := trans["amqp"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, "text/plain", fields["content-type"])
	assert.Equal(t, "logs_topic", fields["exchange"])
	assert.Equal(t, "AMQPtest", fields["routing-key"])
	assert.Equal(t, false, fields["immediate"])
	assert.Equal(t, false, fields["mandatory"])
}

func TestAmqp_DeliverMessage(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendResponse = true

	data, err := hex.DecodeString("01000100000034003c003c0d6d6973746572436f6e73" +
		"756d650000000000000002000c7465737445786368616e67650b7465737444656c697" +
		"66572ce")
	assert.NoError(t, err)
	data2, err := hex.DecodeString("02000100000019003c000000000000000000058" +
		"0000a746578742f706c61696ece")
	assert.NoError(t, err)
	data3, err := hex.DecodeString("030001000000056b696b6f6fce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))

	// method frame
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data2}
	// header frame
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data3}
	// body frame
	amqp.Parse(&req, tcptuple, 0, private)

	trans := expectTransaction(t, results)

	assert.Equal(t, "basic.deliver", trans["method"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, "kikoo", trans["response"])
	assert.Equal(t, common.OK_STATUS, trans["status"])
	fields, ok := trans["amqp"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, "misterConsume", fields["consumer-tag"])
	assert.Equal(t, "text/plain", fields["content-type"])
	assert.Equal(t, "testDeliver", fields["routing-key"])
	assert.Equal(t, false, fields["redelivered"])
}

func TestAmqp_MessagePropertiesFields(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()
	amqp.sendResponse = true

	data, err := hex.DecodeString("01000100000013003c00280000000a546573744865" +
		"6164657200ce02000100000061003c0000000000000000001ab8e00a746578742f706c" +
		"61696e0000002203796f70530000000468696869036e696c56066e756d626572644044" +
		"40000000000002060a656c206d656e73616a650000000055f81dc00c6c6f7665206d65" +
		"7373616765ce0300010000001a5465737420686561646572206669656c647320666f72" +
		"65766572ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)
	m := stream.message
	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	assert.Equal(t, "basic.publish", m.method)
	assert.Equal(t, "persistent", m.fields["delivery-mode"])
	assert.Equal(t, "el mensaje", m.fields["message-id"])
	assert.Equal(t, "love message", m.fields["type"])
	assert.Equal(t, "text/plain", m.fields["content-type"])
	// assert.Equal(t, "September 15 15:31:44 2015", m.Fields["timestamp"])
	priority, ok := m.fields["priority"].(uint8)
	if !ok {
		t.Errorf("Field should be present")
	} else if ok && priority != 6 {
		t.Errorf("Wrong argument")
	}
	headers, ok := m.fields["headers"].(common.MapStr)
	if !ok {
		t.Errorf("Headers should be present")
	}
	assert.Equal(t, "hihi", headers["yop"])
	assert.Equal(t, nil, headers["nil"])
	assert.Equal(t, 40.5, headers["number"])
}

func TestAmqp_ChannelError(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data1, err := hex.DecodeString("0100010000009000140028019685505245434f4e444" +
		"954494f4e5f4641494c4544202d20696e6571756976616c656e74206172672027617574" +
		"6f5f64656c6574652720666f722065786368616e676520277465737445786368616e676" +
		"52720696e2076686f737420272f273a207265636569766564202774727565272062757" +
		"42063757272656e74206973202766616c7365270028000ace")
	assert.NoError(t, err)

	stream := &amqpStream{data: data1, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)
	m := stream.message

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	assert.Equal(t, "channel.close", m.method)
	class, ok := m.fields["class-id"].(uint16)
	if !ok {
		t.Errorf("Field should be present")
	} else if ok && class != 40 {
		t.Errorf("Wrong argument")
	}
	method, ok := m.fields["method-id"].(uint16)
	if !ok {
		t.Errorf("Field should be present")
	} else if ok && method != 10 {
		t.Errorf("Wrong argument")
	}
	code, ok := m.fields["reply-code"].(uint16)
	if !ok {
		t.Errorf("Field should be present")
	} else if ok && code != 406 {
		t.Errorf("Wrong argument")
	}
	text := "PRECONDITION_FAILED - inequivalent arg 'auto_delete' for" +
		" exchange 'testExchange' in vhost '/': received 'true' but current is " +
		"'false'"
	assert.Equal(t, text, m.fields["reply-text"])
}

func TestAmqp_NoWaitQueueDeleteMethod(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendRequest = true

	data, err := hex.DecodeString("010001000000120032002800000a546573745468" +
		"6f6d617304ce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))

	amqp.Parse(&req, tcptuple, 0, private)

	trans := expectTransaction(t, results)

	assert.Equal(t, "queue.delete", trans["method"])
	assert.Equal(t, "queue.delete TestThomas", trans["request"])
	assert.Equal(t, "amqp", trans["type"])
	fields, ok := trans["amqp"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, true, fields["no-wait"])
	assert.Equal(t, false, fields["if-empty"])
	assert.Equal(t, false, fields["if-unused"])
	assert.Equal(t, "TestThomas", fields["queue"])
}

func TestAmqp_RejectMessage(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendRequest = true

	data, err := hex.DecodeString("0100010000000d003c005a000000000000000101ce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))

	// method frame
	amqp.Parse(&req, tcptuple, 0, private)

	trans := expectTransaction(t, results)

	assert.Equal(t, "basic.reject", trans["method"])
	assert.Equal(t, "basic.reject 1", trans["request"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, common.ERROR_STATUS, trans["status"])
	fields, ok := trans["amqp"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, true, fields["multiple"])
}

func TestAmqp_GetEmptyMethod(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendRequest = true

	data, err := hex.DecodeString("01000100000013003c004600000b526f626269" +
		"654b65616e6501ce")
	assert.NoError(t, err)
	data2, err := hex.DecodeString("01000100000005003c004800ce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data2}
	amqp.Parse(&req, tcptuple, 1, private)

	trans := expectTransaction(t, results)
	assert.Equal(t, "basic.get-empty", trans["method"])
	assert.Equal(t, "basic.get RobbieKeane", trans["request"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, common.OK_STATUS, trans["status"])
}

func TestAmqp_GetMethod(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendRequest = true
	amqp.sendResponse = true

	data, err := hex.DecodeString("0100010000000f003c0046000007546573744" +
		"7657401ce")
	assert.NoError(t, err)
	data2, err := hex.DecodeString("0100010000001a003c00470000000000000001" +
		"0000075465737447657400000001ce02000100000019003c000000000000000000" +
		"1280000a746578742f706c61696ece03000100000012476574206d6520696620796" +
		"f752064617265ce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data2}
	amqp.Parse(&req, tcptuple, 1, private)

	trans := expectTransaction(t, results)
	assert.Equal(t, "basic.get", trans["method"])
	assert.Equal(t, "basic.get TestGet", trans["request"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, common.OK_STATUS, trans["status"])
	assert.Equal(t, "Get me if you dare", trans["response"])
}

func TestAmqp_MaxBodyLength(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.maxBodyLength = 10
	amqp.sendRequest = true

	data, err := hex.DecodeString("01000100000010003c002800000007546573744d617" +
		"800ce02000100000019003c0000000000000000001680000a746578742f706c61696ece" +
		"0300010000001649276d2061207665727920626967206d657373616765ce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))

	// method frame
	amqp.Parse(&req, tcptuple, 0, private)

	trans := expectTransaction(t, results)

	assert.Equal(t, "basic.publish", trans["method"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, "I'm a very [...]", trans["request"])
	assert.Equal(t, common.OK_STATUS, trans["status"])
	fields, ok := trans["amqp"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, "text/plain", fields["content-type"])
	assert.Equal(t, "TestMax", fields["routing-key"])
	assert.Equal(t, false, fields["immediate"])
	assert.Equal(t, false, fields["mandatory"])
	_, exists := fields["exchange"]
	assert.False(t, exists)

	data, err = hex.DecodeString("01000100000010003c002800000007546573744d6" +
		"17800ce02000100000018003c0000000000000000003a800009696d6167652f676966" +
		"ce0300010000003a41414141414141414141414141414141414141414141414141414141" +
		"414141414141414141414141414141414141414141414141414141414141ce")
	assert.NoError(t, err)

	tcptuple = testTCPTuple()

	req = protos.Packet{Payload: data}
	private = protos.ProtocolData(new(amqpPrivateData))

	// method frame
	amqp.Parse(&req, tcptuple, 0, private)

	trans = expectTransaction(t, results)

	assert.Equal(t, "basic.publish", trans["method"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, "65 65 65 65 65 65 65 65 65 65 [...]", trans["request"])
	assert.Equal(t, common.OK_STATUS, trans["status"])
	fields, ok = trans["amqp"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, "image/gif", fields["content-type"])
	assert.Equal(t, "TestMax", fields["routing-key"])
	assert.Equal(t, false, fields["immediate"])
	assert.Equal(t, false, fields["mandatory"])
	_, exists = fields["exchange"]
	assert.False(t, exists)
}

func TestAmqp_HideArguments(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendRequest = true
	amqp.parseHeaders = false
	amqp.parseArguments = false

	// parse args
	data, err := hex.DecodeString("0100010000004d0032000a00000a5465737448656164" +
		"6572180000003704626f6f6c74010362697462050568656c6c6f530000001f4869206461" +
		"726c696e6720c3aac3aac3aac3aac3aac3aac3aae697a5e69cacce")
	assert.NoError(t, err)
	tcptuple := testTCPTuple()
	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))
	amqp.Parse(&req, tcptuple, 0, private)

	trans := expectTransaction(t, results)
	assert.Equal(t, "queue.declare", trans["method"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, "queue.declare TestHeader", trans["request"])
	fields, ok := trans["amqp"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, false, fields["durable"])
	assert.Equal(t, true, fields["auto-delete"])
	_, exists := fields["arguments"].(common.MapStr)
	if exists {
		t.Errorf("Arguments field should not be present")
	}

	// parse headers
	data, err = hex.DecodeString("01000100000013003c00280000000a546573744865616" +
		"4657200ce02000100000026003c0000000000000000001a98800a746578742f706c61696" +
		"e02060a656c206d656e73616a65ce0300010000001a54657374206865616465722066696" +
		"56c647320666f7265766572ce")
	assert.NoError(t, err)
	tcptuple = testTCPTuple()
	req = protos.Packet{Payload: data}
	private = protos.ProtocolData(new(amqpPrivateData))
	amqp.Parse(&req, tcptuple, 0, private)
	trans = expectTransaction(t, results)
	assert.Equal(t, "basic.publish", trans["method"])
	assert.Equal(t, "amqp", trans["type"])
	fields, ok = trans["amqp"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, "TestHeader", fields["routing-key"])
	_, exists = fields["exchange"]
	assert.False(t, exists)
	assert.Equal(t, false, fields["mandatory"])
	assert.Equal(t, false, fields["immediate"])
	assert.Equal(t, nil, fields["message-id"])
	assert.Equal(t, nil, fields["content-type"])
	assert.Equal(t, nil, fields["delivery-mode"])
	assert.Equal(t, nil, fields["priority"])
}

func TestAmqp_RecoverMethod(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendRequest = true

	data, err := hex.DecodeString("01000100000005003c006e01ce")
	assert.NoError(t, err)
	data2, err := hex.DecodeString("01000100000004003c006fce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data2}
	amqp.Parse(&req, tcptuple, 1, private)

	trans := expectTransaction(t, results)
	assert.Equal(t, "basic.recover", trans["method"])
	assert.Equal(t, "basic.recover", trans["request"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, common.OK_STATUS, trans["status"])
	assert.Equal(t, common.MapStr{"requeue": true}, trans["amqp"])
}

// this is a specific rabbitMQ method
func TestAmqp_BasicNack(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data1, err := hex.DecodeString("0100010000000d003c0078000000000000000102ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data1, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)
	m := stream.message

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	assert.Equal(t, "basic.nack", m.method)
	assert.Equal(t, false, m.fields["multiple"])
	assert.Equal(t, true, m.fields["requeue"])
}

func TestAmqp_GetTable(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data, err := hex.DecodeString("010001000000890032000a00000a5465737448656164" +
		"657218000000730974696d657374616d70540000000055f7e40903626974620507646563" +
		"696d616c440500ec49050568656c6c6f530000001f4869206461726c696e6720c3aac3aa" +
		"c3aac3aac3aac3aac3aae697a5e69cac06646f75626c656440453e100cbd7da405666c6f" +
		"6174664124cccd04626f6f6c7401ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)
	m := stream.message

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	args, ok := m.fields["arguments"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	double, ok := args["double"].(float64)
	if !ok {
		t.Errorf("Field should be present")
	} else if ok && double != 42.4848648 {
		t.Errorf("Wrong argument")
	}

	float, ok := args["float"].(float32)
	if !ok {
		t.Errorf("Field should be present")
	} else if ok && float != 10.3 {
		t.Errorf("Wrong argument")
	}

	argByte, ok := args["bit"].(int8)
	if !ok {
		t.Errorf("Field should be present")
	} else if ok && argByte != 5 {
		t.Errorf("Wrong argument")
	}

	assert.Equal(t, "Hi darling êêêêêêê日本", args["hello"])
	assert.Equal(t, true, args["bool"])
	assert.Equal(t, "154.85189", args["decimal"])
	assert.Equal(t, "queue.declare", m.method)
	assert.Equal(t, false, m.fields["durable"])
	assert.Equal(t, true, m.fields["no-wait"])
	assert.Equal(t, true, m.fields["auto-delete"])
	assert.Equal(t, false, m.fields["exclusive"])
	// assert.Equal(t, "September 15 11:25:29 2015", args["timestamp"])
	assert.Equal(t, "TestHeader", m.request)
}

func TestAmqp_TableInception(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	data, err := hex.DecodeString("010001000000860028000a000005746573743105" +
		"746f706963020000006f09696e63657074696f6e460000005006696e636570315300" +
		"000006445245414d5306696e6365703253000000064d4152494f4e056c696d626f46" +
		"00000021066c696d626f315300000004436f6262066c696d626f3253000000055361" +
		"69746f06626967496e746c00071afd498d0000ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)
	m := stream.message

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	assert.Equal(t, "exchange.declare", m.method)
	assert.Equal(t, "test1", m.fields["exchange"])

	args, ok := m.fields["arguments"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Nil(t, m.notes)

	bigInt, ok := args["bigInt"].(uint64)
	if !ok {
		t.Errorf("Field should be present")
	} else if ok && bigInt != 2000000000000000 {
		t.Errorf("Wrong argument")
	}
	inception, ok := args["inception"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, "DREAMS", inception["incep1"])
	assert.Equal(t, "MARION", inception["incep2"])

	limbo, ok := inception["limbo"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, "Cobb", limbo["limbo1"])
	assert.Equal(t, "Saito", limbo["limbo2"])
}

func TestAmqp_ArrayFields(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	// byte array, rabbitMQ specific field
	data, err := hex.DecodeString("010001000000260028000a0000057465737431057" +
		"46f706963020000000f05617272617978000000040a007dd2ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)
	m := stream.message

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	args, ok := m.fields["arguments"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Nil(t, m.notes)
	assert.Equal(t, "[10, 0, 125, 210]", args["array"])

	data, err = hex.DecodeString("010001000000b60028000a000005746573743105746" +
		"f706963020000009f0474657374530000001061206c6f74206f6620617272617973210a" +
		"6172726179666c6f6174410000001b64404540000000000064403ccccccccccccd64404" +
		"0a66666666666096172726179626f6f6c410000000a740174007400740174010b617272" +
		"6179737472696e674100000030530000000441414141530000000442424242530000001" +
		"9d090d0bdd0bdd0b020d09ad0b0d180d0b5d0bdd0b8d0bdd0b0ce")
	assert.NoError(t, err)

	stream = &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete = amqp.amqpMessageParser(stream)
	m = stream.message
	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	args, ok = m.fields["arguments"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}

	assert.Equal(t, "a lot of arrays!", args["test"])
	arrayFloat, ok := args["arrayfloat"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, 42.5, arrayFloat["0"])
	assert.Equal(t, 28.8, arrayFloat["1"])
	assert.Equal(t, 33.3, arrayFloat["2"])

	arrayBool, ok := args["arraybool"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, true, arrayBool["0"])
	assert.Equal(t, false, arrayBool["1"])
	assert.Equal(t, false, arrayBool["2"])
	assert.Equal(t, true, arrayBool["3"])
	assert.Equal(t, true, arrayBool["4"])

	arrayString, ok := args["arraystring"].(common.MapStr)
	if !ok {
		t.Errorf("Field should be present")
	}
	assert.Equal(t, "AAAA", arrayString["0"])
	assert.Equal(t, "BBBB", arrayString["1"])
	assert.Equal(t, "Анна Каренина", arrayString["2"])
}

func TestAmqp_WrongTable(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	_, amqp := amqpModForTests()

	// declared table size too big
	data, err := hex.DecodeString("010001000000890032000a00000a54657374486561646" +
		"57218000000da0974696d657374616d70540000000055f7e409036269746205076465636" +
		"96d616c440500ec49050568656c6c6f530000001f4869206461726c696e6720c3aac3aac" +
		"3aac3aac3aac3aac3aae697a5e69cac06646f75626c656440453e100cbd7da405666c6f6" +
		"174664124cccd04626f6f6c7401ce")
	assert.NoError(t, err)

	stream := &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete := amqp.amqpMessageParser(stream)
	m := stream.message
	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	_, exists := m.fields["arguments"].(common.MapStr)
	if exists {
		t.Errorf("Field should not exist")
	}
	assert.Equal(t, []string{"Failed to parse additional arguments"}, m.notes)

	// table size ok, but total non-sense inside
	data, err = hex.DecodeString("010001000000890032000a00000a54657374486561646" +
		"57218000000730974696d657374616d7054004400005521e409036269743705076400036" +
		"96d616c447600ec49180568036c6c0b536400001f480a2064076e6c696e0520c3aac3aac" +
		"34613aac3aac3aa01aae697a5e69cac3c780b75626c6564a4453e100cbd7da4320a6c0b0" +
		"90b664124cc1904626f6f6c7401ce")
	assert.NoError(t, err)

	stream = &amqpStream{data: data, message: new(amqpMessage)}
	ok, complete = amqp.amqpMessageParser(stream)
	m = stream.message
	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Message should be complete")
	}
	_, exists = m.fields["arguments"].(common.MapStr)
	if exists {
		t.Errorf("Field should not exist")
	}
	assert.Equal(t, []string{"Failed to parse additional arguments"}, m.notes)
}

func TestAmqp_isError(t *testing.T) {
	trans := &amqpTransaction{
		method: "channel.close",
		amqp: common.MapStr{
			"reply-code": 200,
		},
	}
	assert.Equal(t, false, isError(trans))
	trans.amqp["reply-code"] = uint16(300)
	assert.Equal(t, true, isError(trans))
	trans.amqp["reply-code"] = uint16(403)
	assert.Equal(t, true, isError(trans))
	trans.method = "basic.reject"
	assert.Equal(t, true, isError(trans))
	trans.method = "basic.return"
	assert.Equal(t, true, isError(trans))
	trans.method = "basic.publish"
	assert.Equal(t, false, isError(trans))
}

func TestAmqp_ChannelCloseErrorMethod(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()

	data, err := hex.DecodeString("0100010000009000140028019685505245434f4e444" +
		"954494f4e5f4641494c4544202d20696e6571756976616c656e74206172672027617574" +
		"6f5f64656c6574652720666f722065786368616e676520277465737445786368616e676" +
		"52720696e2076686f737420272f273a207265636569766564202774727565272062757" +
		"42063757272656e74206973202766616c7365270028000ace")
	assert.NoError(t, err)
	data2, err := hex.DecodeString("0100010000000400280033ce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data2}
	amqp.Parse(&req, tcptuple, 1, private)

	trans := expectTransaction(t, results)
	assert.Equal(t, "channel.close", trans["method"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, common.ERROR_STATUS, trans["status"])
	assert.Nil(t, trans["notes"])
}

func TestAmqp_ConnectionCloseNoError(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.hideConnectionInformation = false

	data, err := hex.DecodeString("01000000000012000a003200c8076b74687862616900000000ce")
	assert.NoError(t, err)
	data2, err := hex.DecodeString("01000000000004000a0033ce")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()

	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data2}
	amqp.Parse(&req, tcptuple, 1, private)

	trans := expectTransaction(t, results)
	assert.Equal(t, "connection.close", trans["method"])
	assert.Equal(t, "amqp", trans["type"])
	assert.Equal(t, common.OK_STATUS, trans["status"])
	assert.Nil(t, trans["notes"])

	fields, ok := trans["amqp"].(common.MapStr)
	assert.True(t, ok)
	code, ok := fields["reply-code"].(uint16)
	assert.True(t, ok)
	assert.Equal(t, uint16(200), code)
}

func TestAmqp_MultipleBodyFrames(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("amqp", "amqpdetailed"))

	results, amqp := amqpModForTests()
	amqp.sendRequest = true
	data, err := hex.DecodeString("0100010000000e003c00280000000568656c6c6f00ce" +
		"02000100000021003c0000000000000000002a80400a746578742f706c61696e00000000" +
		"56a22873ce030001000000202a2a2a68656c6c6f2049206c696b6520746f207075626c69" +
		"736820626967206dce")
	assert.NoError(t, err)
	data2, err := hex.DecodeString("0300010000000a657373616765732a2a2ace")
	assert.NoError(t, err)

	tcptuple := testTCPTuple()
	req := protos.Packet{Payload: data}
	private := protos.ProtocolData(new(amqpPrivateData))
	private = amqp.Parse(&req, tcptuple, 0, private)
	req = protos.Packet{Payload: data2}
	amqp.Parse(&req, tcptuple, 0, private)
	trans := expectTransaction(t, results)
	assert.Equal(t, "basic.publish", trans["method"])
	assert.Equal(t, "***hello I like to publish big messages***", trans["request"])
}
